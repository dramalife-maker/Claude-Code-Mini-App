package db

func (db *DB) IsUserAllowed(tgID int64) (bool, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE tg_id = ?`, tgID).Scan(&count)
	return count > 0, err
}

func (db *DB) AddUser(tgID int64, username string) error {
	_, err := db.Exec(
		`INSERT OR IGNORE INTO users (tg_id, username) VALUES (?, ?)`,
		tgID, username,
	)
	return err
}

func (db *DB) ListUsers() ([]int64, error) {
	rows, err := db.Query(`SELECT tg_id FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
