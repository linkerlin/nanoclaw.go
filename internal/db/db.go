package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/linkerlin/nanoclaw.go/internal/types"
	_ "modernc.org/sqlite"
)

// DB wraps a *sql.DB with nanoclaw-specific operations.
type DB struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS chats (
  jid TEXT PRIMARY KEY,
  name TEXT,
  last_message_time TEXT,
  channel TEXT,
  is_group INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS messages (
  id TEXT,
  chat_jid TEXT,
  sender TEXT,
  sender_name TEXT,
  content TEXT,
  timestamp TEXT,
  is_from_me INTEGER,
  is_bot_message INTEGER DEFAULT 0,
  PRIMARY KEY (id, chat_jid)
);

CREATE TABLE IF NOT EXISTS scheduled_tasks (
  id TEXT PRIMARY KEY,
  group_folder TEXT NOT NULL,
  chat_jid TEXT NOT NULL,
  prompt TEXT NOT NULL,
  schedule_type TEXT NOT NULL,
  schedule_value TEXT NOT NULL,
  next_run TEXT,
  last_run TEXT,
  last_result TEXT,
  status TEXT DEFAULT 'active',
  created_at TEXT NOT NULL,
  context_mode TEXT DEFAULT 'isolated'
);

CREATE TABLE IF NOT EXISTS router_state (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
  group_folder TEXT PRIMARY KEY,
  session_id TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS registered_groups (
  jid TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  folder TEXT NOT NULL UNIQUE,
  trigger_pattern TEXT NOT NULL,
  added_at TEXT NOT NULL,
  requires_trigger INTEGER DEFAULT 1
);
`

// Open opens (or creates) the SQLite database at the given path.
func Open(path string) (*DB, error) {
	sqldb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if _, err := sqldb.Exec(schema); err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return &DB{db: sqldb}, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// SaveMessage inserts a message, ignoring conflicts.
func (d *DB) SaveMessage(m types.NewMessage) error {
	_, err := d.db.Exec(`
		INSERT OR IGNORE INTO messages (id, chat_jid, sender, sender_name, content, timestamp, is_from_me, is_bot_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.ChatJID, m.Sender, m.SenderName, m.Content, m.Timestamp,
		boolInt(m.IsFromMe), boolInt(m.IsBotMessage),
	)
	if err != nil {
		return fmt.Errorf("save message: %w", err)
	}
	// Upsert chat record.
	_, err = d.db.Exec(`
		INSERT INTO chats (jid, name, last_message_time, channel, is_group)
		VALUES (?, ?, ?, 'tui', 1)
		ON CONFLICT(jid) DO UPDATE SET last_message_time=excluded.last_message_time`,
		m.ChatJID, m.ChatJID, m.Timestamp,
	)
	return err
}

// GetNewMessages returns messages that are not from the bot, ordered by timestamp.
func (d *DB) GetNewMessages(chatJID string, since string) ([]types.NewMessage, error) {
	rows, err := d.db.Query(`
		SELECT id, chat_jid, sender, sender_name, content, timestamp, is_from_me, is_bot_message
		FROM messages
		WHERE chat_jid = ? AND timestamp > ? AND is_bot_message = 0
		ORDER BY timestamp ASC`, chatJID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows)
}

// GetRecentMessages returns the N most recent messages for a chat.
func (d *DB) GetRecentMessages(chatJID string, limit int) ([]types.NewMessage, error) {
	rows, err := d.db.Query(`
		SELECT id, chat_jid, sender, sender_name, content, timestamp, is_from_me, is_bot_message
		FROM messages
		WHERE chat_jid = ?
		ORDER BY timestamp DESC
		LIMIT ?`, chatJID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	msgs, err := scanMessages(rows)
	if err != nil {
		return nil, err
	}
	// Reverse to chronological order.
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

// GetRegisteredGroups returns all registered groups.
func (d *DB) GetRegisteredGroups() ([]types.RegisteredGroup, error) {
	rows, err := d.db.Query(`
		SELECT jid, name, folder, trigger_pattern, added_at, requires_trigger
		FROM registered_groups`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []types.RegisteredGroup
	for rows.Next() {
		var g types.RegisteredGroup
		var req int
		if err := rows.Scan(&g.JID, &g.Name, &g.Folder, &g.TriggerPattern, &g.AddedAt, &req); err != nil {
			return nil, err
		}
		g.RequiresTrigger = req != 0
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// RegisterGroup inserts or replaces a group registration.
func (d *DB) RegisterGroup(g types.RegisteredGroup) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO registered_groups (jid, name, folder, trigger_pattern, added_at, requires_trigger)
		VALUES (?, ?, ?, ?, ?, ?)`,
		g.JID, g.Name, g.Folder, g.TriggerPattern, g.AddedAt, boolInt(g.RequiresTrigger),
	)
	return err
}

// GetOrCreateSession returns an existing session ID for the group folder, or creates a new one.
func (d *DB) GetOrCreateSession(groupFolder string) (string, error) {
	var sessionID string
	err := d.db.QueryRow(`SELECT session_id FROM sessions WHERE group_folder = ?`, groupFolder).Scan(&sessionID)
	if err == nil {
		return sessionID, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("query session: %w", err)
	}
	sessionID = fmt.Sprintf("%s-%d", groupFolder, time.Now().UnixNano())
	_, err = d.db.Exec(`INSERT INTO sessions (group_folder, session_id) VALUES (?, ?)`, groupFolder, sessionID)
	if err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}
	return sessionID, nil
}

// GetActiveTasks returns all active scheduled tasks.
func (d *DB) GetActiveTasks() ([]types.ScheduledTask, error) {
	rows, err := d.db.Query(`
		SELECT id, group_folder, chat_jid, prompt, schedule_type, schedule_value,
		       next_run, last_run, last_result, status, created_at, context_mode
		FROM scheduled_tasks WHERE status = 'active'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

// UpdateTaskRun updates last_run, last_result, and next_run for a task.
func (d *DB) UpdateTaskRun(id, lastRun, lastResult string, nextRun *string) error {
	_, err := d.db.Exec(`
		UPDATE scheduled_tasks SET last_run = ?, last_result = ?, next_run = ? WHERE id = ?`,
		lastRun, lastResult, nextRun, id,
	)
	return err
}

func scanMessages(rows *sql.Rows) ([]types.NewMessage, error) {
	var msgs []types.NewMessage
	for rows.Next() {
		var m types.NewMessage
		var fromMe, botMsg int
		if err := rows.Scan(&m.ID, &m.ChatJID, &m.Sender, &m.SenderName, &m.Content, &m.Timestamp, &fromMe, &botMsg); err != nil {
			return nil, err
		}
		m.IsFromMe = fromMe != 0
		m.IsBotMessage = botMsg != 0
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func scanTasks(rows *sql.Rows) ([]types.ScheduledTask, error) {
	var tasks []types.ScheduledTask
	for rows.Next() {
		var t types.ScheduledTask
		if err := rows.Scan(&t.ID, &t.GroupFolder, &t.ChatJID, &t.Prompt,
			&t.ScheduleType, &t.ScheduleValue,
			&t.NextRun, &t.LastRun, &t.LastResult,
			&t.Status, &t.CreatedAt, &t.ContextMode); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
