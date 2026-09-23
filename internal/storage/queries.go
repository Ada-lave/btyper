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
	_, err = saveSession(tx, r, progress)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func saveSession(tx *sql.Tx, r domain.SessionResult, progress map[rune]domain.CharacterProgress) (bool, error) {
	res, err := tx.Exec(`INSERT INTO sessions(started_at,mode,language,target_rune,text,duration_ms,correct,attempts,errors,corrections,wpm,cpm,accuracy,attempt_id,target_skill) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(attempt_id) DO NOTHING`, r.StartedAt.UTC().Format(time.RFC3339Nano), string(r.Mode), r.Language, string(r.TargetRune), r.Text, r.Duration.Milliseconds(), r.Correct, r.Attempts, r.Errors, r.Corrections, r.WPM, r.CPM, r.Accuracy, r.AttemptID, r.TargetSkill)
	if err != nil {
		return false, err
	}
	inserted, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if inserted == 0 {
		return false, nil
	}
	id, err := res.LastInsertId()
	if err != nil {
		return false, err
	}
	for key, c := range r.Chars {
		if _, err = tx.Exec("INSERT INTO character_stats(session_id,rune,samples,errors,latency_ms,latency_samples) VALUES(?,?,?,?,?,?)", id, string(key), c.Samples, c.Errors, c.LatencyMS, c.LatencySamples); err != nil {
			return false, err
		}
	}
	for key, p := range progress {
		_, err = tx.Exec(`INSERT INTO progress(language,rune,samples,errors,latency_ms,accuracy,confidence,mastery_streak,unlocked,mastered) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(language,rune) DO UPDATE SET samples=excluded.samples,errors=excluded.errors,latency_ms=excluded.latency_ms,accuracy=excluded.accuracy,confidence=excluded.confidence,mastery_streak=excluded.mastery_streak,unlocked=MAX(progress.unlocked,excluded.unlocked),mastered=MAX(progress.mastered,excluded.mastered)`, p.Language, string(key), p.Samples, p.Errors, p.LatencyMS, p.Accuracy, p.Confidence, p.MasteryStreak, p.Unlocked, p.Mastered)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func (s *SQLite) LoadSkills(language string) (map[string]domain.Skill, error) {
	rows, err := s.db.Query(`SELECT kind,pattern,samples,errors,latency_samples,latency_ms,accuracy,confidence,level,last_practiced,due_at FROM skills WHERE language=?`, language)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]domain.Skill{}
	for rows.Next() {
		var skill domain.Skill
		var kind, last, due string
		skill.Language = language
		if err := rows.Scan(&kind, &skill.Pattern, &skill.Samples, &skill.Errors, &skill.LatencySamples, &skill.LatencyMS, &skill.Accuracy, &skill.Confidence, &skill.Level, &last, &due); err != nil {
			return nil, err
		}
		skill.Kind = domain.SkillKind(kind)
		if last != "" {
			skill.LastPracticed, err = time.Parse(time.RFC3339Nano, last)
			if err != nil {
				return nil, err
			}
		}
		if due != "" {
			skill.DueAt, err = time.Parse(time.RFC3339Nano, due)
			if err != nil {
				return nil, err
			}
		}
		out[kind+":"+skill.Pattern] = skill
	}
	return out, rows.Err()
}

func (s *SQLite) SaveAdaptiveSession(r domain.SessionResult, progress map[rune]domain.CharacterProgress, skills map[string]domain.Skill) error {
	if r.AttemptID == "" {
		return errors.New("missing attempt ID")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	inserted, err := saveSession(tx, r, progress)
	if err != nil {
		return err
	}
	if inserted {
		for _, skill := range skills {
			last, due := "", ""
			if !skill.LastPracticed.IsZero() {
				last = skill.LastPracticed.UTC().Format(time.RFC3339Nano)
			}
			if !skill.DueAt.IsZero() {
				due = skill.DueAt.UTC().Format(time.RFC3339Nano)
			}
			_, err = tx.Exec(`INSERT INTO skills(language,kind,pattern,samples,errors,latency_samples,latency_ms,accuracy,confidence,level,last_practiced,due_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(language,kind,pattern) DO UPDATE SET samples=excluded.samples,errors=excluded.errors,latency_samples=excluded.latency_samples,latency_ms=excluded.latency_ms,accuracy=excluded.accuracy,confidence=excluded.confidence,level=excluded.level,last_practiced=excluded.last_practiced,due_at=excluded.due_at`, skill.Language, string(skill.Kind), skill.Pattern, skill.Samples, skill.Errors, skill.LatencySamples, skill.LatencyMS, skill.Accuracy, skill.Confidence, skill.Level, last, due)
			if err != nil {
				return err
			}
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
	if _, err = tx.Exec("DELETE FROM character_stats; DELETE FROM sessions; DELETE FROM progress; DELETE FROM practice_time; DELETE FROM skills;"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLite) Trends(language string, since time.Time) ([]domain.TrendPoint, error) {
	rows, err := s.db.Query(`WITH latency AS (SELECT session_id,SUM(latency_ms) latency_ms,SUM(latency_samples) latency_samples FROM character_stats GROUP BY session_id)
SELECT date(s.started_at,'localtime'),COUNT(*),COALESCE(SUM(s.duration_ms),0),
CASE WHEN SUM(s.duration_ms)>0 THEN SUM(s.correct)*12000.0/SUM(s.duration_ms) ELSE 0 END,
CASE WHEN SUM(s.attempts)>0 THEN SUM(s.attempts-s.errors)*1.0/SUM(s.attempts) ELSE 0 END,
COALESCE(SUM(latency.latency_ms)/NULLIF(SUM(latency.latency_samples),0),0)
FROM sessions s LEFT JOIN latency ON latency.session_id=s.id
WHERE s.language=? AND julianday(s.started_at)>=julianday(?)
GROUP BY date(s.started_at,'localtime') ORDER BY date(s.started_at,'localtime')`, language, since.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.TrendPoint
	for rows.Next() {
		var p domain.TrendPoint
		var ms int64
		if err := rows.Scan(&p.Day, &p.Sessions, &ms, &p.WPM, &p.Accuracy, &p.Latency); err != nil {
			return nil, err
		}
		p.Duration = time.Duration(ms) * time.Millisecond
		out = append(out, p)
	}
	return out, rows.Err()
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

func (s *SQLite) PracticeDays() (map[string]time.Duration, error) {
	rows, err := s.db.Query(`SELECT day,SUM(duration_ns) FROM practice_time GROUP BY day`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	days := map[string]time.Duration{}
	for rows.Next() {
		var day string
		var ns int64
		if err := rows.Scan(&day, &ns); err != nil {
			return nil, err
		}
		days[day] = time.Duration(ns)
	}
	return days, rows.Err()
}

func (s *SQLite) ProgressReview(now time.Time, language string) (domain.ProgressReview, error) {
	year, month, day := now.Date()
	today := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
	weekStart := today.AddDate(0, 0, -6)
	previousStart := weekStart.AddDate(0, 0, -7)
	tomorrow := today.AddDate(0, 0, 1)
	var review domain.ProgressReview
	practice := func(start, end time.Time) (time.Duration, error) {
		var ns int64
		err := s.db.QueryRow(`SELECT COALESCE(SUM(duration_ns),0) FROM practice_time WHERE day>=? AND day<?`, start.Format("2006-01-02"), end.Format("2006-01-02")).Scan(&ns)
		return time.Duration(ns), err
	}
	var err error
	if review.WeekPractice, err = practice(weekStart, tomorrow); err != nil {
		return review, err
	}
	if review.PreviousPractice, err = practice(previousStart, weekStart); err != nil {
		return review, err
	}
	var weekDuration, weekCorrect int64
	err = s.db.QueryRow(`SELECT COUNT(*),COALESCE(SUM(duration_ms),0),COALESCE(SUM(correct),0) FROM sessions WHERE language=? AND julianday(started_at)>=julianday(?) AND julianday(started_at)<julianday(?)`, language, weekStart.UTC().Format(time.RFC3339Nano), tomorrow.UTC().Format(time.RFC3339Nano)).Scan(&review.WeekSessions, &weekDuration, &weekCorrect)
	if err != nil {
		return review, err
	}
	if weekDuration > 0 {
		review.WeekWPM = float64(weekCorrect) * 12000 / float64(weekDuration)
	}
	var baselineDuration, baselineCorrect int64
	err = s.db.QueryRow(`SELECT COUNT(*),COALESCE(SUM(duration_ms),0),COALESCE(SUM(correct),0) FROM sessions WHERE language=? AND julianday(started_at)<julianday(?)`, language, weekStart.UTC().Format(time.RFC3339Nano)).Scan(&review.BaselineSessions, &baselineDuration, &baselineCorrect)
	if err != nil {
		return review, err
	}
	if baselineDuration > 0 {
		review.BaselineWPM = float64(baselineCorrect) * 12000 / float64(baselineDuration)
	}
	return review, nil
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
