package storage

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"btyper/internal/domain"
)

const BackupFormatVersion = 1

type Backup struct {
	FormatVersion int                        `json:"format_version"`
	ExportedAt    time.Time                  `json:"exported_at"`
	Settings      domain.Settings            `json:"settings"`
	Progress      []domain.CharacterProgress `json:"progress"`
	Skills        []domain.Skill             `json:"skills"`
	Sessions      []domain.SessionResult     `json:"sessions"`
	PracticeTime  []domain.PracticeTime      `json:"practice_time"`
}

func (s *SQLite) ExportBackup(w io.Writer) error {
	b := Backup{FormatVersion: BackupFormatVersion, ExportedAt: time.Now().UTC()}
	var err error
	if b.Settings, err = s.LoadSettings(); err != nil {
		return err
	}
	rows, err := s.db.Query(`SELECT language,rune,samples,errors,latency_ms,accuracy,confidence,mastery_streak,unlocked,mastered FROM progress ORDER BY language,rune`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var p domain.CharacterProgress
		var key string
		if err = rows.Scan(&p.Language, &key, &p.Samples, &p.Errors, &p.LatencyMS, &p.Accuracy, &p.Confidence, &p.MasteryStreak, &p.Unlocked, &p.Mastered); err != nil {
			rows.Close()
			return err
		}
		rs := []rune(key)
		if len(rs) != 1 {
			rows.Close()
			return errors.New("invalid progress rune")
		}
		p.Rune = rs[0]
		b.Progress = append(b.Progress, p)
	}
	if err = rows.Close(); err != nil {
		return err
	}
	rows, err = s.db.Query(`SELECT language,kind,pattern,samples,errors,latency_samples,latency_ms,accuracy,confidence,level,last_practiced,due_at FROM skills ORDER BY language,kind,pattern`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var skill domain.Skill
		var kind, last, due string
		if err = rows.Scan(&skill.Language, &kind, &skill.Pattern, &skill.Samples, &skill.Errors, &skill.LatencySamples, &skill.LatencyMS, &skill.Accuracy, &skill.Confidence, &skill.Level, &last, &due); err != nil {
			rows.Close()
			return err
		}
		skill.Kind = domain.SkillKind(kind)
		if last != "" {
			skill.LastPracticed, err = time.Parse(time.RFC3339Nano, last)
			if err != nil {
				rows.Close()
				return err
			}
		}
		if due != "" {
			skill.DueAt, err = time.Parse(time.RFC3339Nano, due)
			if err != nil {
				rows.Close()
				return err
			}
		}
		b.Skills = append(b.Skills, skill)
	}
	if err = rows.Close(); err != nil {
		return err
	}
	rows, err = s.db.Query(`SELECT id,attempt_id,started_at,mode,language,target_rune,target_skill,text,duration_ms,correct,attempts,errors,corrections,wpm,cpm,accuracy FROM sessions ORDER BY id`)
	if err != nil {
		return err
	}
	var sessionIDs []int64
	usedAttempts := map[string]bool{}
	var legacySessions []int
	for rows.Next() {
		var r domain.SessionResult
		var id, ms int64
		var stamp, mode, target string
		var attempt sql.NullString
		if err = rows.Scan(&id, &attempt, &stamp, &mode, &r.Language, &target, &r.TargetSkill, &r.Text, &ms, &r.Correct, &r.Attempts, &r.Errors, &r.Corrections, &r.WPM, &r.CPM, &r.Accuracy); err != nil {
			rows.Close()
			return err
		}
		r.AttemptID = attempt.String
		if r.AttemptID == "" {
			legacySessions = append(legacySessions, len(b.Sessions))
		} else {
			usedAttempts[r.AttemptID] = true
		}
		r.StartedAt, err = time.Parse(time.RFC3339Nano, stamp)
		if err != nil {
			rows.Close()
			return err
		}
		r.Mode, r.Duration, r.Chars = domain.Mode(mode), time.Duration(ms)*time.Millisecond, map[rune]*domain.CharacterStat{}
		if rs := []rune(target); len(rs) == 1 {
			r.TargetRune = rs[0]
		}
		b.Sessions = append(b.Sessions, r)
		sessionIDs = append(sessionIDs, id)
	}
	if err = rows.Close(); err != nil {
		return err
	}
	for _, index := range legacySessions {
		candidate := fmt.Sprintf("legacy-session-%d", sessionIDs[index])
		for usedAttempts[candidate] {
			candidate += "-legacy"
		}
		b.Sessions[index].AttemptID = candidate
		usedAttempts[candidate] = true
	}
	for i, id := range sessionIDs {
		stats, statErr := s.db.Query(`SELECT rune,samples,errors,latency_ms,latency_samples FROM character_stats WHERE session_id=?`, id)
		if statErr != nil {
			return statErr
		}
		for stats.Next() {
			var key string
			var c domain.CharacterStat
			var latencySamples sql.NullInt64
			if err = stats.Scan(&key, &c.Samples, &c.Errors, &c.LatencyMS, &latencySamples); err != nil {
				stats.Close()
				return err
			}
			c.LatencySamples = int(latencySamples.Int64)
			rs := []rune(key)
			if len(rs) == 1 {
				c.Rune = rs[0]
				b.Sessions[i].Chars[c.Rune] = &c
			}
		}
		if err = stats.Close(); err != nil {
			return err
		}
	}
	rows, err = s.db.Query(`SELECT attempt_id,day,duration_ns FROM practice_time ORDER BY day,attempt_id`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var p domain.PracticeTime
		var ns int64
		if err = rows.Scan(&p.AttemptID, &p.Day, &ns); err != nil {
			rows.Close()
			return err
		}
		p.Duration = time.Duration(ns)
		b.PracticeTime = append(b.PracticeTime, p)
	}
	if err = rows.Close(); err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(b)
}

func (s *SQLite) ImportBackup(r io.Reader) error {
	var b Backup
	dec := json.NewDecoder(io.LimitReader(r, 64<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&b); err != nil {
		return fmt.Errorf("decode backup: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return errors.New("backup must contain exactly one JSON document")
	}
	if b.FormatVersion != BackupFormatVersion {
		return fmt.Errorf("unsupported backup format %d", b.FormatVersion)
	}
	seen := map[string]bool{}
	for _, session := range b.Sessions {
		if session.AttemptID == "" || seen[session.AttemptID] || session.StartedAt.IsZero() || (session.Mode != domain.ModeAdaptive && session.Mode != domain.ModeLearn && session.Mode != domain.ModeImprove && session.Mode != domain.ModeText) {
			return errors.New("invalid or duplicate session attempt ID")
		}
		seen[session.AttemptID] = true
	}
	for _, skill := range b.Skills {
		n := len([]rune(skill.Pattern))
		validKind := skill.Kind == domain.SkillRune || skill.Kind == domain.SkillBigram || skill.Kind == domain.SkillNumber || skill.Kind == domain.SkillUppercase || skill.Kind == domain.SkillPunctuation
		if skill.Language == "" || !validKind || skill.Samples < 0 || skill.Errors < 0 || skill.Errors > skill.Samples || skill.Level < 0 || skill.Level > 5 || (skill.Kind == domain.SkillBigram && n != 2) || (skill.Kind != domain.SkillBigram && n != 1) {
			return errors.New("invalid skill in backup")
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`DELETE FROM character_stats; DELETE FROM sessions; DELETE FROM progress; DELETE FROM practice_time; DELETE FROM skills; DELETE FROM settings;`); err != nil {
		return err
	}
	for _, session := range b.Sessions {
		if _, err = saveSession(tx, session, nil); err != nil {
			return err
		}
	}
	for _, p := range b.Progress {
		_, err = tx.Exec(`INSERT INTO progress(language,rune,samples,errors,latency_ms,accuracy,confidence,mastery_streak,unlocked,mastered) VALUES(?,?,?,?,?,?,?,?,?,?)`, p.Language, string(p.Rune), p.Samples, p.Errors, p.LatencyMS, p.Accuracy, p.Confidence, p.MasteryStreak, p.Unlocked, p.Mastered)
		if err != nil {
			return err
		}
	}
	for _, skill := range b.Skills {
		last, due := "", ""
		if !skill.LastPracticed.IsZero() {
			last = skill.LastPracticed.UTC().Format(time.RFC3339Nano)
		}
		if !skill.DueAt.IsZero() {
			due = skill.DueAt.UTC().Format(time.RFC3339Nano)
		}
		_, err = tx.Exec(`INSERT INTO skills(language,kind,pattern,samples,errors,latency_samples,latency_ms,accuracy,confidence,level,last_practiced,due_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, skill.Language, string(skill.Kind), skill.Pattern, skill.Samples, skill.Errors, skill.LatencySamples, skill.LatencyMS, skill.Accuracy, skill.Confidence, skill.Level, last, due)
		if err != nil {
			return err
		}
	}
	for _, p := range b.PracticeTime {
		if p.AttemptID == "" || p.Duration < 0 {
			return errors.New("invalid practice time")
		}
		if _, err = time.Parse("2006-01-02", p.Day); err != nil {
			return err
		}
		_, err = tx.Exec(`INSERT INTO practice_time(attempt_id,day,duration_ns) VALUES(?,?,?)`, p.AttemptID, p.Day, int64(p.Duration))
		if err != nil {
			return err
		}
	}
	raw, err := json.Marshal(b.Settings)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO settings(id,value) VALUES(1,?)`, string(raw)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLite) ExportCSV(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	type report struct {
		name   string
		header []string
		query  string
	}
	reports := []report{
		{"sessions.csv", []string{"started_at", "mode", "language", "wpm", "accuracy", "errors", "duration_ms"}, `SELECT started_at,mode,language,wpm,accuracy,errors,duration_ms FROM sessions ORDER BY id`},
		{"practice_time.csv", []string{"day", "attempt_id", "duration_ns"}, `SELECT day,attempt_id,duration_ns FROM practice_time ORDER BY day,attempt_id`},
		{"skills.csv", []string{"language", "kind", "pattern", "samples", "errors", "latency_ms", "accuracy", "confidence", "level", "last_practiced", "due_at"}, `SELECT language,kind,pattern,samples,errors,latency_ms,accuracy,confidence,level,last_practiced,due_at FROM skills ORDER BY language,kind,pattern`},
	}
	for _, rep := range reports {
		if err := s.writeCSV(filepath.Join(dir, rep.name), rep.header, rep.query); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLite) writeCSV(path string, header []string, query string) error {
	rows, err := s.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	b := bufio.NewWriter(f)
	w := csv.NewWriter(b)
	if err = w.Write(header); err != nil {
		f.Close()
		return err
	}
	cols, err := rows.Columns()
	if err != nil {
		f.Close()
		return err
	}
	values := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range values {
		ptrs[i] = &values[i]
	}
	for rows.Next() {
		if err = rows.Scan(ptrs...); err != nil {
			f.Close()
			return err
		}
		record := make([]string, len(values))
		for i, v := range values {
			switch x := v.(type) {
			case nil:
			case []byte:
				record[i] = string(x)
			case int64:
				record[i] = strconv.FormatInt(x, 10)
			case float64:
				record[i] = strconv.FormatFloat(x, 'f', -1, 64)
			default:
				record[i] = fmt.Sprint(x)
			}
		}
		if err = w.Write(record); err != nil {
			f.Close()
			return err
		}
	}
	w.Flush()
	if err = w.Error(); err != nil {
		f.Close()
		return err
	}
	if err = b.Flush(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
