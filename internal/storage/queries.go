package storage

import (
	"btyper/internal/domain"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

func (s *SQLite) LoadSettings() (domain.Settings, error) {
	d := domain.DefaultSettings()
	var raw string
	err := s.db.QueryRow("SELECT value FROM settings WHERE id=1").Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return d, nil
	}
	if err != nil {
		return d, err
	}
	if err = json.Unmarshal([]byte(raw), &d); err != nil {
		return domain.DefaultSettings(), fmt.Errorf("settings: %w", err)
	}
	return d, nil
}
func (s *SQLite) SaveSettings(v domain.Settings) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("INSERT INTO settings(id,value) VALUES(1,?) ON CONFLICT(id) DO UPDATE SET value=excluded.value", string(b))
	return err
}
func (s *SQLite) LoadProgress(language string) (map[rune]domain.CharacterProgress, error) {
	rows, err := s.db.Query("SELECT rune,samples,errors,latency_ms,accuracy,confidence,mastery_streak FROM progress WHERE language=?", language)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[rune]domain.CharacterProgress{}
	for rows.Next() {
		p, err := scanProgress(rows, language)
		if err != nil {
			return nil, err
		}
		out[p.Rune] = p
	}
	return out, rows.Err()
}
func (s *SQLite) SaveSession(r domain.SessionResult, progress map[rune]domain.CharacterProgress) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO sessions(started_at,mode,language,target_rune,text,duration_ms,correct,attempts,errors,corrections,wpm,cpm,accuracy) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, r.StartedAt.Format(time.RFC3339Nano), string(r.Mode), r.Language, string(r.TargetRune), r.Text, r.Duration.Milliseconds(), r.Correct, r.Attempts, r.Errors, r.Corrections, r.WPM, r.CPM, r.Accuracy)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	for key, c := range r.Chars {
		if _, err = tx.Exec("INSERT INTO character_stats(session_id,rune,samples,errors,latency_ms) VALUES(?,?,?,?,?)", id, string(key), c.Samples, c.Errors, c.LatencyMS); err != nil {
			return err
		}
	}
	for key, p := range progress {
		_, err = tx.Exec(`INSERT INTO progress(language,rune,samples,errors,latency_ms,accuracy,confidence,mastery_streak) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(language,rune) DO UPDATE SET samples=excluded.samples,errors=excluded.errors,latency_ms=excluded.latency_ms,accuracy=excluded.accuracy,confidence=excluded.confidence,mastery_streak=excluded.mastery_streak`, p.Language, string(key), p.Samples, p.Errors, p.LatencyMS, p.Accuracy, p.Confidence, p.MasteryStreak)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *SQLite) History(limit int) ([]domain.HistoryEntry, error) {
	rows, err := s.db.Query("SELECT id,started_at,mode,language,wpm,accuracy,errors,duration_ms FROM sessions ORDER BY id DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.HistoryEntry
	for rows.Next() {
		h, err := scanHistory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
func (s *SQLite) Reset() error {
	_, err := s.db.Exec("DELETE FROM character_stats; DELETE FROM sessions; DELETE FROM progress;")
	return err
}
