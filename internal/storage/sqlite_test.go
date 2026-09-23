package storage

import (
	"btyper/internal/domain"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func testDB(t *testing.T) *SQLite {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func sampleResult(id, lang string, now time.Time) domain.SessionResult {
	return domain.SessionResult{AttemptID: id, StartedAt: now, Mode: domain.ModeLearn, Language: lang, Text: "ee", Duration: time.Second, Correct: 2, Attempts: 3, Errors: 1, WPM: 24, Accuracy: 2.0 / 3, Chars: map[rune]*domain.CharacterStat{'e': {Rune: 'e', Samples: 3, Errors: 1, LatencyMS: 1000, LatencySamples: 1}}}
}

func TestSessionIdempotencyAndRollback(t *testing.T) {
	s := testDB(t)
	r := sampleResult("one", "en", time.Now())
	p := map[rune]domain.CharacterProgress{'e': {Language: "en", Rune: 'e', Samples: 3, Unlocked: true}}
	if err := s.SaveSession(r, p); err != nil {
		t.Fatal(err)
	}
	p['e'] = domain.CharacterProgress{Language: "en", Rune: 'e', Samples: 999}
	if err := s.SaveSession(r, p); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadProgress("en")
	if err != nil || got['e'].Samples != 3 {
		t.Fatalf("duplicate rewrote progress: %v %v", got, err)
	}
	if _, err = s.db.Exec(`CREATE TRIGGER fail_progress BEFORE UPDATE ON progress BEGIN SELECT RAISE(ABORT,'test failure'); END;`); err != nil {
		t.Fatal(err)
	}
	r.AttemptID = "two"
	if err = s.SaveSession(r, p); err == nil {
		t.Fatal("expected failure")
	}
	h, err := s.History(domain.HistoryFilter{})
	if err != nil || len(h) != 1 {
		t.Fatalf("partial save: %v %v", h, err)
	}
	var count int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM character_stats`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("partial stats: %d %v", count, err)
	}
}

func TestResetRollbackAndLanguages(t *testing.T) {
	s := testDB(t)
	for _, lang := range []string{"en", "ru"} {
		r := sampleResult(lang, lang, time.Now())
		if err := s.SaveSession(r, map[rune]domain.CharacterProgress{'e': {Language: lang, Rune: 'e', Samples: 4}}); err != nil {
			t.Fatal(err)
		}
	}
	for _, lang := range []string{"en", "ru"} {
		p, err := s.LoadProgress(lang)
		if err != nil || p['e'].Samples != 4 {
			t.Fatal(p, err)
		}
	}
	if _, err := s.db.Exec(`CREATE TRIGGER fail_reset BEFORE DELETE ON progress BEGIN SELECT RAISE(ABORT,'test failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := s.Reset(); err == nil {
		t.Fatal("expected failure")
	}
	h, err := s.History(domain.HistoryFilter{})
	if err != nil || len(h) != 2 {
		t.Fatal("reset was not atomic", h, err)
	}
	if _, err = s.db.Exec(`DROP TRIGGER fail_reset`); err != nil {
		t.Fatal(err)
	}
	if err = s.Reset(); err != nil {
		t.Fatal(err)
	}
	h, err = s.History(domain.HistoryFilter{})
	if err != nil || len(h) != 0 {
		t.Fatal(h, err)
	}
}

func TestHistoryFiltersPagesAndWeightedSummary(t *testing.T) {
	s := testDB(t)
	now := time.Now().UTC()
	for i := 0; i < 55; i++ {
		r := sampleResult(time.Duration(i).String(), "en", now)
		if err := s.SaveSession(r, nil); err != nil {
			t.Fatal(err)
		}
	}
	r := sampleResult("ru", "ru", now)
	r.Duration = 3 * time.Second
	r.Correct = 1
	r.Attempts = 1
	r.Errors = 0
	r.Mode = domain.ModeText
	if err := s.SaveSession(r, nil); err != nil {
		t.Fatal(err)
	}
	f := domain.HistoryFilter{Language: "en", Mode: domain.ModeLearn, Since: now.Add(-time.Minute), Limit: 50}
	h, err := s.History(f)
	if err != nil || len(h) != 50 {
		t.Fatal(len(h), err)
	}
	f.Offset = 50
	h, err = s.History(f)
	if err != nil || len(h) != 5 {
		t.Fatal(len(h), err)
	}
	summary, err := s.Summary(f)
	if err != nil || summary.Sessions != 55 || summary.Duration != 55*time.Second || summary.WPM != 24 {
		t.Fatal(summary, err)
	}
	f = domain.HistoryFilter{Language: "ru", Mode: domain.ModeText}
	summary, err = s.Summary(f)
	if err != nil || summary.Sessions != 1 || summary.WPM != 4 || summary.Accuracy != 1 {
		t.Fatal(summary, err)
	}
	f.Since = now.Add(time.Hour)
	h, err = s.History(f)
	if err != nil || len(h) != 0 {
		t.Fatal(h, err)
	}
}

func TestDailyTimePersistsWithoutCompletedLesson(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	entries := []domain.PracticeTime{{AttemptID: "abandoned", Day: "2026-09-18", Duration: 3 * time.Second}, {AttemptID: "abandoned", Day: "2026-09-19", Duration: time.Second}}
	if err = s.SavePracticeTime(entries); err != nil {
		t.Fatal(err)
	}
	entries[0].Duration = time.Second
	if err = s.SavePracticeTime(entries); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if d, err := s.PracticeTime("2026-09-18"); err != nil || d != 3*time.Second {
		t.Fatal(d, err)
	}
	if d, err := s.PracticeTime("2026-09-19"); err != nil || d != time.Second {
		t.Fatal(d, err)
	}
	if days, err := s.PracticeDays(); err != nil || days["2026-09-18"] != 3*time.Second || days["2026-09-19"] != time.Second {
		t.Fatal(days, err)
	}
	if err = s.Reset(); err != nil {
		t.Fatal(err)
	}
	if d, err := s.PracticeTime("2026-09-18"); err != nil || d != 0 {
		t.Fatal(d, err)
	}
}

func TestMigrateLegacyDatabase(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "btyper.db"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY);` + schemaV1 + `
INSERT INTO settings VALUES(1,'{"Language":"ru"}');
INSERT INTO progress VALUES('en','a',30,0,200,1,1,2);
INSERT INTO sessions VALUES(1,'2026-09-18T12:00:00Z','learn','en','a','aaa',1000,3,3,0,0,36,180,1);
INSERT INTO character_stats VALUES(1,'a',3,0,400);`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	p, err := s.LoadProgress("en")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range "enitrlsa" {
		if !p[r].Unlocked {
			t.Fatalf("%c was not restored", r)
		}
	}
	if !p['a'].Mastered || p['a'].Samples != 30 {
		t.Fatal(p['a'])
	}
	v, err := s.LoadSettings()
	if err != nil || v.Language != "ru" {
		t.Fatal(v, err)
	}
	h, err := s.History(domain.HistoryFilter{})
	if err != nil || len(h) != 1 {
		t.Fatal(h, err)
	}
	var latency sql.NullInt64
	if err = s.db.QueryRow(`SELECT latency_samples FROM character_stats`).Scan(&latency); err != nil || latency.Valid {
		t.Fatal("invented legacy latency measurements", latency, err)
	}
	if err = s.migrate(); err != nil {
		t.Fatal("migration not repeatable", err)
	}
	var encoded bytes.Buffer
	if err = s.ExportBackup(&encoded); err != nil {
		t.Fatal("legacy export failed", err)
	}
	var backup Backup
	if err = json.Unmarshal(encoded.Bytes(), &backup); err != nil {
		t.Fatal(err)
	}
	if len(backup.Sessions) != 1 || backup.Sessions[0].AttemptID == "" || backup.Sessions[0].Chars['a'].LatencySamples != 0 {
		t.Fatalf("invalid legacy backup: %+v", backup.Sessions)
	}
	restored := testDB(t)
	if err = restored.ImportBackup(bytes.NewReader(encoded.Bytes())); err != nil {
		t.Fatal("legacy restore failed", err)
	}
	if sessions, err := restored.History(domain.HistoryFilter{}); err != nil || len(sessions) != 1 {
		t.Fatalf("legacy history was not restored: %v %v", sessions, err)
	}
}

func TestMigrationFailureRollsBack(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "btyper.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY);` + schemaV1 + `CREATE TABLE practice_time(conflict TEXT);`); err != nil {
		t.Fatal(err)
	}
	if s, err := Open(dir); err == nil {
		s.Close()
		t.Fatal("expected migration failure")
	}
	var version int
	if err = db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil || version != 1 {
		t.Fatal(version, err)
	}
	if _, err = db.Exec(`SELECT unlocked FROM progress`); err == nil {
		t.Fatal("schema changes were not rolled back")
	}
}

func TestMigrateFromIntermediateSchemas(t *testing.T) {
	for _, version := range []int{2, 3, 4} {
		t.Run(fmt.Sprintf("v%d", version), func(t *testing.T) {
			dir := t.TempDir()
			db, err := sql.Open("sqlite", filepath.Join(dir, "btyper.db"))
			if err != nil {
				t.Fatal(err)
			}
			_, err = db.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY);` + schemaV1 + `
ALTER TABLE progress ADD COLUMN unlocked INTEGER NOT NULL DEFAULT 0;
ALTER TABLE progress ADD COLUMN mastered INTEGER NOT NULL DEFAULT 0;
ALTER TABLE character_stats ADD COLUMN latency_samples INTEGER;
ALTER TABLE sessions ADD COLUMN attempt_id TEXT;
CREATE UNIQUE INDEX sessions_attempt_id ON sessions(attempt_id);
CREATE INDEX sessions_filter ON sessions(language,mode,started_at);
INSERT INTO schema_migrations(version) VALUES(2);
INSERT INTO settings VALUES(1,'{"Language":"en"}');
INSERT INTO progress(language,rune,samples,errors,latency_ms,accuracy,confidence,mastery_streak,unlocked,mastered) VALUES('en','e',30,0,200,1,1,2,1,1);
INSERT INTO sessions(started_at,mode,language,target_rune,text,duration_ms,correct,attempts,errors,corrections,wpm,cpm,accuracy,attempt_id) VALUES('2026-09-18T12:00:00Z','learn','en','e','ee',1000,2,2,0,0,24,120,1,'old-attempt');`)
			if err != nil {
				t.Fatal(err)
			}
			if version >= 3 {
				_, err = db.Exec(`CREATE TABLE practice_time(attempt_id TEXT NOT NULL, day TEXT NOT NULL, duration_ns INTEGER NOT NULL CHECK(duration_ns>=0), PRIMARY KEY(attempt_id,day));
INSERT INTO practice_time VALUES('old-attempt','2026-09-18',1000000000);
INSERT INTO schema_migrations(version) VALUES(3);`)
				if err != nil {
					t.Fatal(err)
				}
			}
			if version == 4 {
				_, err = db.Exec(`ALTER TABLE sessions ADD COLUMN target_skill TEXT NOT NULL DEFAULT '';
CREATE TABLE skills(language TEXT NOT NULL,kind TEXT NOT NULL CHECK(kind IN ('rune','bigram')),pattern TEXT NOT NULL,samples INTEGER NOT NULL,errors INTEGER NOT NULL,latency_samples INTEGER NOT NULL,latency_ms REAL NOT NULL,accuracy REAL NOT NULL,confidence REAL NOT NULL,level INTEGER NOT NULL,last_practiced TEXT NOT NULL DEFAULT '',due_at TEXT NOT NULL DEFAULT '',PRIMARY KEY(language,kind,pattern));
INSERT INTO skills VALUES('en','rune','e',30,0,30,200,1,1,1,'','');
INSERT INTO schema_migrations(version) VALUES(4);`)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err = db.Close(); err != nil {
				t.Fatal(err)
			}
			s, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			var current int
			if err = s.db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&current); err != nil || current != 5 {
				t.Fatalf("schema version %d: %v", current, err)
			}
			if h, err := s.History(domain.HistoryFilter{}); err != nil || len(h) != 1 {
				t.Fatalf("history lost: %v %v", h, err)
			}
			if skills, err := s.LoadSkills("en"); err != nil || skills["rune:e"].Samples != 30 {
				t.Fatalf("rune skill lost: %v %v", skills, err)
			}
			if duration, err := s.PracticeTime("2026-09-18"); err != nil || duration != time.Second {
				t.Fatalf("practice time lost: %v %v", duration, err)
			}
		})
	}
}
