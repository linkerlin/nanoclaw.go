package internal

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// DB SQLite数据库封装
type DB struct {
	*sql.DB
}

// OpenDB 打开数据库
func OpenDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := initSchema(db); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &DB{db}, nil
}

func initSchema(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    chat_jid TEXT NOT NULL,
    sender TEXT,
    sender_name TEXT,
    content TEXT,
    timestamp TEXT,
    is_bot INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_messages_chat ON messages(chat_jid, timestamp);

CREATE TABLE IF NOT EXISTS groups (
    jid TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    folder TEXT NOT NULL UNIQUE,
    trigger_pattern TEXT,
    requires_trigger INTEGER DEFAULT 1,
    added_at TEXT
);

CREATE TABLE IF NOT EXISTS sessions (
    group_folder TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    updated_at TEXT
);

CREATE TABLE IF NOT EXISTS tasks (
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
    created_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_tasks_next_run ON tasks(next_run) WHERE status = 'active';
`
	_, err := db.Exec(schema)
	return err
}

// SaveMessage 保存消息
func (d *DB) SaveMessage(m *Message) error {
	_, err := d.Exec(
		`INSERT OR REPLACE INTO messages (id, chat_jid, sender, sender_name, content, timestamp, is_bot) 
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.ChatJID, m.Sender, m.SenderName, m.Content, m.Timestamp.Format(time.RFC3339), boolToInt(m.IsBotMessage),
	)
	return err
}

// GetMessages 获取消息
func (d *DB) GetMessages(chatJID ChatJID, limit int) ([]Message, error) {
	rows, err := d.Query(
		`SELECT id, chat_jid, sender, sender_name, content, timestamp, is_bot 
		 FROM messages WHERE chat_jid = ? ORDER BY timestamp DESC LIMIT ?`,
		chatJID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		var ts string
		var isBot int
		if err := rows.Scan(&m.ID, &m.ChatJID, &m.Sender, &m.SenderName, &m.Content, &ts, &isBot); err != nil {
			return nil, err
		}
		m.Timestamp, _ = time.Parse(time.RFC3339, ts)
		m.IsBotMessage = isBot == 1
		msgs = append(msgs, m)
	}

	// 反转为时间顺序
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, rows.Err()
}

// GetGroup 获取群组
func (d *DB) GetGroup(jid ChatJID) (*Group, error) {
	var g Group
	var reqTrigger int
	var addedAt string
	err := d.QueryRow(
		`SELECT jid, name, folder, trigger_pattern, requires_trigger, added_at FROM groups WHERE jid = ?`,
		jid,
	).Scan(&g.JID, &g.Name, &g.Folder, &g.TriggerPattern, &reqTrigger, &addedAt)
	if err != nil {
		return nil, err
	}
	g.RequiresTrigger = reqTrigger == 1
	g.AddedAt, _ = time.Parse(time.RFC3339, addedAt)
	return &g, nil
}

// SaveGroup 保存群组
func (d *DB) SaveGroup(g *Group) error {
	_, err := d.Exec(
		`INSERT OR REPLACE INTO groups (jid, name, folder, trigger_pattern, requires_trigger, added_at) 
		 VALUES (?, ?, ?, ?, ?, ?)`,
		g.JID, g.Name, g.Folder, g.TriggerPattern, boolToInt(g.RequiresTrigger), g.AddedAt.Format(time.RFC3339),
	)
	return err
}

// GetSession 获取会话
func (d *DB) GetSession(groupFolder string) (*Session, error) {
	var s Session
	var updatedAt string
	err := d.QueryRow(
		`SELECT group_folder, session_id, updated_at FROM sessions WHERE group_folder = ?`,
		groupFolder,
	).Scan(&s.GroupFolder, &s.SessionID, &updatedAt)
	if err != nil {
		return nil, err
	}
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &s, nil
}

// SaveSession 保存会话
func (d *DB) SaveSession(s *Session) error {
	_, err := d.Exec(
		`INSERT OR REPLACE INTO sessions (group_folder, session_id, updated_at) VALUES (?, ?, ?)`,
		s.GroupFolder, s.SessionID, time.Now().Format(time.RFC3339),
	)
	return err
}

// GetDueTasks 获取到期任务
func (d *DB) GetDueTasks(now time.Time) ([]Task, error) {
	rows, err := d.Query(
		`SELECT id, group_folder, chat_jid, prompt, schedule_type, schedule_value, next_run, last_run, last_result, status, created_at 
		 FROM tasks WHERE status = 'active' AND next_run <= ?`,
		now.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasks(rows)
}

// SaveTask 保存任务
func (d *DB) SaveTask(t *Task) error {
	var nextRun *string
	if t.NextRun != nil {
		s := t.NextRun.Format(time.RFC3339)
		nextRun = &s
	}
	_, err := d.Exec(
		`INSERT OR REPLACE INTO tasks (id, group_folder, chat_jid, prompt, schedule_type, schedule_value, next_run, status, created_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.GroupFolder, t.ChatJID, t.Prompt, t.ScheduleType, t.ScheduleValue, nextRun, t.Status, t.CreatedAt.Format(time.RFC3339),
	)
	return err
}

// UpdateTaskRun 更新任务执行结果
func (d *DB) UpdateTaskRun(id string, result string, nextRun *time.Time) error {
	var nr *string
	if nextRun != nil {
		s := nextRun.Format(time.RFC3339)
		nr = &s
	}
	_, err := d.Exec(
		`UPDATE tasks SET last_run = ?, last_result = ?, next_run = ? WHERE id = ?`,
		time.Now().Format(time.RFC3339), result, nr, id,
	)
	return err
}

func scanTasks(rows *sql.Rows) ([]Task, error) {
	var tasks []Task
	for rows.Next() {
		var t Task
		var nextRun, lastRun, lastResult *string
		var createdAt string
		if err := rows.Scan(&t.ID, &t.GroupFolder, &t.ChatJID, &t.Prompt, &t.ScheduleType, &t.ScheduleValue, &nextRun, &lastRun, &lastResult, &t.Status, &createdAt); err != nil {
			return nil, err
		}
		if lastResult != nil {
			t.LastResult = *lastResult
		}
		if nextRun != nil && *nextRun != "" {
			tr, _ := time.Parse(time.RFC3339, *nextRun)
			t.NextRun = &tr
		}
		if lastRun != nil && *lastRun != "" {
			lr, _ := time.Parse(time.RFC3339, *lastRun)
			t.LastRun = &lr
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
