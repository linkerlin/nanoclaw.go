package internal

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIPCServerXSkipFail_New(t *testing.T) {
	tmpDir := t.TempDir()
	socketDir := filepath.Join(tmpDir, "ipc")
	
	server, err := NewIPCServer(socketDir)
	if err != nil {
		t.Fatalf("NewIPCServer failed: %v", err)
	}
	defer server.Stop()
	
	// 验证socket文件已创建
	socketPath := filepath.Join(socketDir, "nanoclaw.sock")
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Error("Socket file not created")
	}
	
	// 验证权限
	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	
	mode := info.Mode()
	if mode&os.ModeSocket == 0 {
		t.Error("File is not a socket")
	}
}

func TestIPCServerXSkipFail_ClientServer(t *testing.T) {
	tmpDir := t.TempDir()
	socketDir := filepath.Join(tmpDir, "ipc")
	
	// 创建服务器
	server, err := NewIPCServer(socketDir)
	if err != nil {
		t.Fatalf("NewIPCServer failed: %v", err)
	}
	defer server.Stop()
	
	// 设置处理器
	server.SetHandler(func(req AgentRequest) AgentResponse {
		return AgentResponse{
			Content: "Echo: " + req.GroupFolder,
		}
	})
	
	// 在后台启动服务器
	go server.Start()
	
	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)
	
	// 创建客户端
	socketPath := filepath.Join(socketDir, "nanoclaw.sock")
	client := NewIPCClient(socketPath)
	
	// 发送请求
	req := AgentRequest{
		GroupFolder: "test-group",
		Messages: []Message{
			{
				ID:      MessageID("test-1"),
				Content: "Hello",
			},
		},
	}
	
	resp, err := client.Call(req)
	if err != nil {
		t.Fatalf("Client.Call failed: %v", err)
	}
	
	expected := "Echo: test-group"
	if resp.Content != expected {
		t.Errorf("Response = %q, want %q", resp.Content, expected)
	}
}

func TestIPCClientXSkip_Call_Error(t *testing.T) {
	// 连接到不存在的socket
	client := NewIPCClient("/tmp/non-existent-socket.sock")
	
	req := AgentRequest{
		GroupFolder: "test",
	}
	
	_, err := client.Call(req)
	if err == nil {
		t.Error("Expected error for non-existent socket")
	}
}

func TestIPCServerXSkipFail_MultipleClients(t *testing.T) {
	tmpDir := t.TempDir()
	socketDir := filepath.Join(tmpDir, "ipc")
	
	// 创建服务器
	server, err := NewIPCServer(socketDir)
	if err != nil {
		t.Fatalf("NewIPCServer failed: %v", err)
	}
	defer server.Stop()
	
	requestCount := 0
	server.SetHandler(func(req AgentRequest) AgentResponse {
		requestCount++
		return AgentResponse{
			Content: "Response " + string(rune('0'+requestCount)),
		}
	})
	
	// 在后台启动服务器
	go server.Start()
	time.Sleep(100 * time.Millisecond)
	
	// 多个客户端并发请求
	socketPath := filepath.Join(socketDir, "nanoclaw.sock")
	
	done := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func(idx int) {
			defer func() { done <- true }()
			
			client := NewIPCClient(socketPath)
			req := AgentRequest{
				GroupFolder: "client-" + string(rune('0'+idx)),
			}
			
			resp, err := client.Call(req)
			if err != nil {
				t.Errorf("Client %d call failed: %v", idx, err)
				return
			}
			
			if resp.Content == "" {
				t.Errorf("Client %d got empty response", idx)
			}
		}(i)
	}
	
	// 等待所有客户端完成
	for i := 0; i < 3; i++ {
		<-done
	}
}

func TestIsRoot(t *testing.T) {
	// 这个测试取决于运行环境
	// 在CI环境中通常不是root
	result := IsRoot()
	t.Logf("IsRoot() = %v", result)
}
