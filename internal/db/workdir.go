package db

import (
	"strings"
)

// AddWorkDir 把路徑寫進清單；已存在則略過。空字串不寫。
func (db *DB) AddWorkDir(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	_, err := db.Exec(`INSERT OR IGNORE INTO work_dirs (path) VALUES (?)`, path)
	return err
}

// ListWorkDirs 回傳已記住的工作目錄路徑。
func (db *DB) ListWorkDirs() ([]string, error) {
	rows, err := db.Query(`SELECT path FROM work_dirs ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// seedWorkDirsFromSessions 把現有 session 的 work_dir 補進清單（升級既有 DB 用）。
func (db *DB) seedWorkDirsFromSessions() error {
	_, err := db.Exec(`
		INSERT OR IGNORE INTO work_dirs (path)
		SELECT DISTINCT trim(work_dir) FROM sessions WHERE trim(work_dir) != ''
	`)
	return err
}
