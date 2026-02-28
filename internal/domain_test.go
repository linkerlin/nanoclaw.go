package internal

import (
	"regexp"
	"testing"
	"time"
)

func TestMessage_HasTrigger(t *testing.T) {
	pattern := regexp.MustCompile(`(?i)^@Andy\b`)
	
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "exact trigger",
			content: "@Andy hello",
			want:    true,
		},
		{
			name:    "case insensitive",
			content: "@ANDY help",
			want:    true,
		},
		{
			name:    "no trigger",
			content: "hello world",
			want:    false,
		},
		{
			name:    "partial match",
			content: "@Andrew hello",
			want:    false,
		},
		{
			name:    "trigger in middle",
			content: "hello @Andy there",
			want:    false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Message{
				Content:   tt.content,
				Timestamp: time.Now(),
			}
			got := m.HasTrigger(pattern)
			if got != tt.want {
				t.Errorf("HasTrigger() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessage_IsSystem(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "system command",
			content: "/task list",
			want:    true,
		},
		{
			name:    "regular message",
			content: "hello world",
			want:    false,
		},
		{
			name:    "slash in middle",
			content: "hello/world",
			want:    false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Message{
				Content:   tt.content,
				Timestamp: time.Now(),
			}
			got := m.IsSystem()
			if got != tt.want {
				t.Errorf("IsSystem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTask_IsDue(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	
	tests := []struct {
		name   string
		task   Task
		want   bool
	}{
		{
			name: "due now",
			task: Task{
				Status:  "active",
				NextRun: &past,
			},
			want: true,
		},
		{
			name: "not due yet",
			task: Task{
				Status:  "active",
				NextRun: &future,
			},
			want: false,
		},
		{
			name: "no next run",
			task: Task{
				Status:  "active",
				NextRun: nil,
			},
			want: false,
		},
		{
			name: "paused task",
			task: Task{
				Status:  "paused",
				NextRun: &past,
			},
			want: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.task.IsDue(now)
			if got != tt.want {
				t.Errorf("IsDue() = %v, want %v", got, tt.want)
			}
		})
	}
}
