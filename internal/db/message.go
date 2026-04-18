package db

const (
	MessageStatusPending = "pending"
	MessageStatusDone    = "done"
)

type Message struct {
	ID        int64  `json:"id"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func (db *DB) AddMessage(sessionID, role, content string) error {
	_, err := db.Exec(
		`INSERT INTO messages (session_id, role, content, status) VALUES (?, ?, ?, ?)`,
		sessionID, role, content, MessageStatusDone,
	)
	return err
}

// CreatePendingMessage inserts an empty assistant row with status=pending.
func (db *DB) CreatePendingMessage(sessionID string) (int64, error) {
	return db.CreatePendingMessageWithRole(sessionID, "claude")
}

// CreatePendingMessageWithRole inserts an empty row for the given role (claude, shell, ...).
func (db *DB) CreatePendingMessageWithRole(sessionID, role string) (int64, error) {
	res, err := db.Exec(
		`INSERT INTO messages (session_id, role, content, status) VALUES (?, ?, '', ?)`,
		sessionID, role, MessageStatusPending,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// AppendMessageContent appends text to a pending message.
func (db *DB) AppendMessageContent(msgID int64, delta string) error {
	if delta == "" {
		return nil
	}
	_, err := db.Exec(
		`UPDATE messages SET content = content || ? WHERE id = ? AND status = ?`,
		delta, msgID, MessageStatusPending,
	)
	return err
}

// FinalizeMessage marks a message as done.
func (db *DB) FinalizeMessage(msgID int64) error {
	_, err := db.Exec(`UPDATE messages SET status = ? WHERE id = ?`, MessageStatusDone, msgID)
	return err
}

// ResetPendingMessages marks all pending rows done (server startup cleanup).
func (db *DB) ResetPendingMessages() error {
	_, err := db.Exec(
		`UPDATE messages SET status = ? WHERE status = ?`,
		MessageStatusDone, MessageStatusPending,
	)
	return err
}

// FinalizePendingMessagesForSession marks pending rows done for one session.
func (db *DB) FinalizePendingMessagesForSession(sessionID string) error {
	_, err := db.Exec(
		`UPDATE messages SET status = ? WHERE session_id = ? AND status = ?`,
		MessageStatusDone, sessionID, MessageStatusPending,
	)
	return err
}

func (db *DB) ClearMessages(sessionID string) error {
	_, err := db.Exec(`DELETE FROM messages WHERE session_id = ?`, sessionID)
	return err
}

func (db *DB) ListMessages(sessionID string) ([]*Message, error) {
	rows, err := db.Query(
		`SELECT id, session_id, role, content, status, created_at FROM messages WHERE session_id = ? ORDER BY id ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.Status, &m.CreatedAt); err != nil {
			return nil, err
		}
		if m.Status == "" {
			m.Status = MessageStatusDone
		}
		msgs = append(msgs, &m)
	}
	return msgs, rows.Err()
}
