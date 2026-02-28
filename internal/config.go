package internal

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
)

// Config 应用配置
type Config struct {
	App       AppConfig
	LLM       LLMConfig
	Scheduler SchedulerConfig
}

// AppConfig 应用配置
type AppConfig struct {
	Name            string
	DataDir         string
	GroupsDir       string
	TriggerPattern  *regexp.Regexp
	MaxConcurrent   int64
}

// LLMConfig LLM配置（从环境变量读取）
type LLMConfig struct {
	APIKey  string // OPENAI_API_KEY
	BaseURL string // OPENAI_BASE_URL
	Model   string // OPENAI_MODEL
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	PollInterval int // 秒
}

// LoadConfig 从环境变量加载配置
func LoadConfig() *Config {
	cfg := &Config{
		App: AppConfig{
			Name:          getEnv("NANOCLAW_NAME", "Andy"),
			DataDir:       getEnv("NANOCLAW_DATA_DIR", defaultDataDir()),
			GroupsDir:     getEnv("NANOCLAW_GROUPS_DIR", defaultGroupsDir()),
			MaxConcurrent: int64(getEnvInt("NANOCLAW_MAX_CONCURRENT", 5)),
		},
		LLM: LLMConfig{
			APIKey:  getEnv("OPENAI_API_KEY", ""),
			BaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
			Model:   getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		},
		Scheduler: SchedulerConfig{
			PollInterval: getEnvInt("NANOCLAW_SCHEDULER_INTERVAL", 60),
		},
	}

	// 编译触发词正则
	cfg.App.TriggerPattern = regexp.MustCompile(`(?i)^@` + regexp.QuoteMeta(cfg.App.Name) + `\b`)

	return cfg
}

// DBPath 返回数据库路径
func (c *Config) DBPath() string {
	return filepath.Join(c.App.DataDir, "nanoclaw.db")
}

// SocketPath 返回Unix Socket路径
func (c *Config) SocketPath() string {
	return "/var/run/nanoclaw/nanoclaw.sock"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func defaultDataDir() string {
	return filepath.Join(projectRoot(), "data")
}

func defaultGroupsDir() string {
	return filepath.Join(projectRoot(), "groups")
}

func projectRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}
