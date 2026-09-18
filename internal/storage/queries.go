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
	rows, err := s.db.Query("SELECT rune,samples,errors,latency_ms,accuracy,confidence,mastery_streak,unlocked,mastered FROM progress WHERE language=?", language)
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
	if r.AttemptID == "" {
		return errors.New("missing attempt ID")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO sessions(started_at,mode,language,target_rune,text,duration_ms,correct,attempts,errors,corrections,wpm,cpm,accuracy,attempt_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(attempt_id) DO NOTHING`, r.StartedAt.UTC().Format(time.RFC3339Nano), string(r.Mode), r.Language, string(r.TargetRune), r.Text, r.Duration.Milliseconds(), r.Correct, r.Attempts, r.Errors, r.Corrections, r.WPM, r.CPM, r.Accuracy, r.AttemptID)
	if err != nil {
		return err
	}
	inserted, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if inserted == 0 {
		return tx.Commit()
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	for key, c := range r.Chars {
		if _, err = tx.Exec("INSERT INTO character_stats(session_id,rune,samples,errors,latency_ms,latency_samples) VALUES(?,?,?,?,?,?)", id, string(key), c.Samples, c.Errors, c.LatencyMS, c.LatencySamples); err != nil {
			return err
		}
	}
	for key, p := range progress {
		_, err = tx.Exec(`INSERT INTO progress(language,rune,samples,errors,latency_ms,accuracy,confidence,mastery_streak,unlocked,mastered) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(language,rune) DO UPDATE SET samples=excluded.samples,errors=excluded.errors,latency_ms=excluded.latency_ms,accuracy=excluded.accuracy,confidence=excluded.confidence,mastery_streak=excluded.mastery_streak,unlocked=MAX(progress.unlocked,excluded.unlocked),mastered=MAX(progress.mastered,excluded.mastered)`, p.Language, string(key), p.Samples, p.Errors, p.LatencyMS, p.Accuracy, p.Confidence, p.MasteryStreak, p.Unlocked, p.Mastered)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
func historyWhere(f domain.HistoryFilter) (string, []any) {
	where := " WHERE 1=1"
	var args []any
	if f.Language != "" {
		where += " AND language=?"
		args = append(args, f.Language)
	}
	if f.Mode != "" {
		where += " AND mode=?"
		args = append(args, string(f.Mode))
	}
	if !f.Since.IsZero() {
		where += " AND julianday(started_at)>=julianday(?)"
		args = append(args, f.Since.UTC().Format(time.RFC3339Nano))
	}
	return where, args
}

func (s *SQLite) History(f domain.HistoryFilter) ([]domain.HistoryEntry, error) {
	where, args := historyWhere(f)
	limit := f.Limit
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	args = append(args, limit, max(0, f.Offset))
	rows, err := s.db.Query("SELECT id,started_at,mode,language,wpm,accuracy,errors,duration_ms FROM sessions"+where+" ORDER BY id DESC LIMIT ? OFFSET ?", args...)
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
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec("DELETE FROM character_stats; DELETE FROM sessions; DELETE FROM progress; DELETE FROM practice_time;"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLite) SavePracticeTime(entries []domain.PracticeTime) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, entry := range entries {
		if entry.AttemptID == "" || entry.Duration < 0 {
			return errors.New("invalid practice time")
		}
		if _, err = time.Parse("2006-01-02", entry.Day); err != nil {
			return err
		}
		if _, err = tx.Exec(`INSERT INTO practice_time(attempt_id,day,duration_ns) VALUES(?,?,?) ON CONFLICT(attempt_id,day) DO UPDATE SET duration_ns=MAX(practice_time.duration_ns,excluded.duration_ns)`, entry.AttemptID, entry.Day, int64(entry.Duration)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLite) PracticeTime(day string) (time.Duration, error) {
	var ns int64
	err := s.db.QueryRow(`SELECT COALESCE(SUM(duration_ns),0) FROM practice_time WHERE day=?`, day).Scan(&ns)
	return time.Duration(ns), err
}

func (s *SQLite) Summary(f domain.HistoryFilter) (domain.HistorySummary, error) {
	where, args := historyWhere(f)
	var out domain.HistorySummary
	var ms, correct, attempts, errorsCount int64
	err := s.db.QueryRow(`SELECT COUNT(*),COALESCE(SUM(duration_ms),0),COALESCE(SUM(correct),0),COALESCE(SUM(attempts),0),COALESCE(SUM(errors),0) FROM sessions`+where, args...).Scan(&out.Sessions, &ms, &correct, &attempts, &errorsCount)
	out.Duration = time.Duration(ms) * time.Millisecond
	if ms > 0 {
		out.WPM = float64(correct) * 12000 / float64(ms)
	}
	if attempts > 0 {
		out.Accuracy = float64(attempts-errorsCount) / float64(attempts)
	}
	return out, err
}
