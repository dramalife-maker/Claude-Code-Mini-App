package db

type Message struct {
	ID        int64  `json:"id"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

func (db *DB) AddMessage(sessionID, role, content string) error {
	_, err := db.Exec(
		`INSERT INTO messages (session_id, role, content) VALUES (?, ?, ?)`,
		sessionID, role, content,
	)
	return err
}

func (db *DB) ClearMessages(sessionID string) error {
	_, err := db.Exec(`DELETE FROM messages WHERE session_id = ?`, sessionID)
	return err
}

func (db *DB) ListMessages(sessionID string) ([]*Message, error) {
	rows, err := db.Query(
		`SELECT id, session_id, role, content, created_at FROM messages WHERE session_id = ? ORDER BY id ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, &m)
	}
	return msgs, rows.Err()
}
