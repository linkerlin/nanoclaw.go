package internal

import (
	"os"
	"testing"
)

// TestTempDB 创建临时数据库用于测试
func TestTempDB(t *testing.T) *DB {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to open temp db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})
	return db
}

// TestConfig 返回测试配置
func TestConfig(t *testing.T) *Config {
	cfg := LoadConfig()
	cfg.App.DataDir = t.TempDir()
	cfg.App.GroupsDir = t.TempDir() + "/groups"
	return cfg
}

// SkipIfNoLLM 如果没有配置LLM则跳过测试
func SkipIfNoLLM(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping LLM integration test")
	}
}

// HasLLMConfig 检查是否有LLM配置
func HasLLMConfig() bool {
	return os.Getenv("OPENAI_API_KEY") != ""
}

// UniqueID 生成唯一ID
func UniqueID(prefix string) string {
	return prefix + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[os.Getpid()%len(letters)]
	}
	return string(b)
}
