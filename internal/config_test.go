package internal

import (
	"os"
	"testing"
)

func TestLoadConfig_FromEnv(t *testing.T) {
	// 保存原始环境变量
	origAPIKey := os.Getenv("OPENAI_API_KEY")
	origBaseURL := os.Getenv("OPENAI_BASE_URL")
	origModel := os.Getenv("OPENAI_MODEL")
	origName := os.Getenv("NANOCLAW_NAME")
	
	// 测试结束后恢复
	defer func() {
		os.Setenv("OPENAI_API_KEY", origAPIKey)
		os.Setenv("OPENAI_BASE_URL", origBaseURL)
		os.Setenv("OPENAI_MODEL", origModel)
		os.Setenv("NANOCLAW_NAME", origName)
	}()
	
	// 设置测试环境变量
	os.Setenv("OPENAI_API_KEY", "test-api-key")
	os.Setenv("OPENAI_BASE_URL", "https://test.example.com/v1")
	os.Setenv("OPENAI_MODEL", "test-model")
	os.Setenv("NANOCLAW_NAME", "TestBot")
	
	cfg := LoadConfig()
	
	// 验证 LLM 配置
	if cfg.LLM.APIKey != "test-api-key" {
		t.Errorf("APIKey = %q, want %q", cfg.LLM.APIKey, "test-api-key")
	}
	if cfg.LLM.BaseURL != "https://test.example.com/v1" {
		t.Errorf("BaseURL = %q, want %q", cfg.LLM.BaseURL, "https://test.example.com/v1")
	}
	if cfg.LLM.Model != "test-model" {
		t.Errorf("Model = %q, want %q", cfg.LLM.Model, "test-model")
	}
	
	// 验证 App 配置
	if cfg.App.Name != "TestBot" {
		t.Errorf("Name = %q, want %q", cfg.App.Name, "TestBot")
	}
	
	// 验证触发词正则已编译
	if cfg.App.TriggerPattern == nil {
		t.Error("TriggerPattern should be compiled")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	// 清除相关环境变量
	for _, key := range []string{"OPENAI_API_KEY", "OPENAI_BASE_URL", "OPENAI_MODEL", "NANOCLAW_NAME"} {
		os.Unsetenv(key)
	}
	
	cfg := LoadConfig()
	
	// 验证默认值
	if cfg.App.Name != "Andy" {
		t.Errorf("Default Name = %q, want %q", cfg.App.Name, "Andy")
	}
	if cfg.LLM.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("Default BaseURL = %q, want %q", cfg.LLM.BaseURL, "https://api.openai.com/v1")
	}
	if cfg.LLM.Model != "gpt-4o-mini" {
		t.Errorf("Default Model = %q, want %q", cfg.LLM.Model, "gpt-4o-mini")
	}
	if cfg.App.MaxConcurrent != 5 {
		t.Errorf("Default MaxConcurrent = %d, want %d", cfg.App.MaxConcurrent, 5)
	}
}

func TestLoadConfig_TriggerPattern(t *testing.T) {
	os.Setenv("NANOCLAW_NAME", "Bob")
	defer os.Unsetenv("NANOCLAW_NAME")
	
	cfg := LoadConfig()
	
	// 测试触发词匹配
	tests := []string{
		"@Bob hello",
		"@BOB hello",
		"@bob hello",
	}
	
	for _, content := range tests {
		if !cfg.App.TriggerPattern.MatchString(content) {
			t.Errorf("TriggerPattern should match %q", content)
		}
	}
	
	// 测试不应匹配
	nonMatches := []string{
		"@Bobby hello",
		"hello @Bob",
		"@Andy hello",
	}
	
	for _, content := range nonMatches {
		if cfg.App.TriggerPattern.MatchString(content) {
			t.Errorf("TriggerPattern should not match %q", content)
		}
	}
}

func TestConfig_DBPath(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			DataDir: "/tmp/test",
		},
	}
	
	want := "/tmp/test/nanoclaw.db"
	if got := cfg.DBPath(); got != want {
		t.Errorf("DBPath() = %q, want %q", got, want)
	}
}
