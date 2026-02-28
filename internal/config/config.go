package config

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

var (
	AssistantName         = getEnv("ASSISTANT_NAME", "Andy")
	PollInterval          = 2000  // ms
	SchedulerPollInterval = 60000 // ms
	StoreDir              = filepath.Join(projectRoot(), "store")
	GroupsDir             = filepath.Join(projectRoot(), "groups")
	DataDir               = filepath.Join(projectRoot(), "data")
	MainGroupFolder       = "main"
	IdleTimeout           = 1800000 // ms (30 min)
	MaxConcurrentAgents   = 5
	TriggerPattern        = compileTriggerPattern(AssistantName)
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func projectRoot() string {
	// Walk up from this file's location to find the module root (go.mod).
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

func compileTriggerPattern(name string) *regexp.Regexp {
	escaped := regexp.QuoteMeta(name)
	return regexp.MustCompile(`(?i)^@` + escaped + `\b`)
}

func DBPath() string {
	return filepath.Join(DataDir, "nanoclaw.db")
}
