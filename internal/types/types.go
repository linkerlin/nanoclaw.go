package types

// RegisteredGroup represents a group registered with the bot.
type RegisteredGroup struct {
	JID             string `db:"jid"`
	Name            string `db:"name"`
	Folder          string `db:"folder"`
	TriggerPattern  string `db:"trigger_pattern"`
	AddedAt         string `db:"added_at"`
	RequiresTrigger bool   `db:"requires_trigger"`
}

// NewMessage represents an incoming message.
type NewMessage struct {
	ID           string `db:"id"`
	ChatJID      string `db:"chat_jid"`
	Sender       string `db:"sender"`
	SenderName   string `db:"sender_name"`
	Content      string `db:"content"`
	Timestamp    string `db:"timestamp"`
	IsFromMe     bool   `db:"is_from_me"`
	IsBotMessage bool   `db:"is_bot_message"`
}

// ScheduledTask represents a recurring or one-time scheduled task.
type ScheduledTask struct {
	ID            string  `db:"id"`
	GroupFolder   string  `db:"group_folder"`
	ChatJID       string  `db:"chat_jid"`
	Prompt        string  `db:"prompt"`
	ScheduleType  string  `db:"schedule_type"` // cron | interval | once
	ScheduleValue string  `db:"schedule_value"`
	ContextMode   string  `db:"context_mode"` // group | isolated
	NextRun       *string `db:"next_run"`
	LastRun       *string `db:"last_run"`
	LastResult    *string `db:"last_result"`
	Status        string  `db:"status"` // active | paused | completed
	CreatedAt     string  `db:"created_at"`
}
