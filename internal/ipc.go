package internal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// IPCServer Unix Socket服务器
type IPCServer struct {
	socketPath string
	listener   net.Listener
	handler    func(AgentRequest) AgentResponse
}

// AgentRequest Agent请求
type AgentRequest struct {
	GroupFolder string    `json:"group_folder"`
	Messages    []Message `json:"messages"`
}

// AgentResponse Agent响应
type AgentResponse struct {
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// NewIPCServer 创建IPC服务器
func NewIPCServer(socketDir string) (*IPCServer, error) {
	// 创建目录
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	socketPath := filepath.Join(socketDir, "nanoclaw.sock")

	// 删除已存在的socket
	os.Remove(socketPath)

	// 创建Unix Socket
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	// 设置权限
	if err := os.Chmod(socketPath, 0770); err != nil {
		listener.Close()
		return nil, fmt.Errorf("chmod: %w", err)
	}

	return &IPCServer{
		socketPath: socketPath,
		listener:   listener,
	}, nil
}

// SetHandler 设置请求处理器
func (s *IPCServer) SetHandler(h func(AgentRequest) AgentResponse) {
	s.handler = h
}

// Start 启动服务器
func (s *IPCServer) Start() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			continue
		}
		go s.handleConn(conn)
	}
}

// Stop 停止服务器
func (s *IPCServer) Stop() error {
	return s.listener.Close()
}

func (s *IPCServer) handleConn(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	decoder := json.NewDecoder(reader)
	encoder := json.NewEncoder(conn)

	var req AgentRequest
	if err := decoder.Decode(&req); err != nil {
		encoder.Encode(AgentResponse{Error: err.Error()})
		return
	}

	if s.handler != nil {
		resp := s.handler(req)
		encoder.Encode(resp)
	} else {
		encoder.Encode(AgentResponse{Error: "no handler"})
	}
}

// IPCClient Unix Socket客户端
type IPCClient struct {
	socketPath string
}

// NewIPCClient 创建IPC客户端
func NewIPCClient(socketPath string) *IPCClient {
	return &IPCClient{socketPath: socketPath}
}

// Call 调用Agent
func (c *IPCClient) Call(req AgentRequest) (AgentResponse, error) {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return AgentResponse{}, fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	if err := encoder.Encode(req); err != nil {
		return AgentResponse{}, err
	}

	var resp AgentResponse
	if err := decoder.Decode(&resp); err != nil {
		return AgentResponse{}, err
	}

	return resp, nil
}

// IsRoot 检查是否以root运行
func IsRoot() bool {
	return os.Getuid() == 0
}

// DropPrivileges 切换到nanoclaw用户（简化版，实际使用setuid）
func DropPrivileges() error {
	// 注意：实际实现需要使用syscall.Setuid
	// 这里仅作演示
	return nil
}
