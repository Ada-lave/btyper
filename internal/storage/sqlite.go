package storage

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
)

type SQLite struct{ db *sql.DB }

func Open(dir string) (*SQLite, error) {
	if dir == "" {
		dir = defaultDataDir()
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "btyper.db"))
	if err != nil {
		return nil, err
	}
	s := &SQLite{db: db}
	if _, err = db.Exec("PRAGMA foreign_keys=ON; PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;"); err != nil {
		db.Close()
		return nil, err
	}
	if err = s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}
func defaultDataDir() string {
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, "btyper")
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".local", "share", "btyper")
}
func (s *SQLite) Close() error { return s.db.Close() }
