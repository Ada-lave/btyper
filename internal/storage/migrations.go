package storage

import (
	"btyper/internal/trainer"
	"fmt"
)

const schemaV1 = `CREATE TABLE settings(id INTEGER PRIMARY KEY CHECK(id=1), value TEXT NOT NULL);
CREATE TABLE progress(language TEXT NOT NULL, rune TEXT NOT NULL, samples INTEGER NOT NULL, errors INTEGER NOT NULL, latency_ms REAL NOT NULL, accuracy REAL NOT NULL, confidence REAL NOT NULL, mastery_streak INTEGER NOT NULL, PRIMARY KEY(language,rune));
CREATE TABLE sessions(id INTEGER PRIMARY KEY AUTOINCREMENT, started_at TEXT NOT NULL, mode TEXT NOT NULL, language TEXT NOT NULL, target_rune TEXT NOT NULL, text TEXT NOT NULL, duration_ms INTEGER NOT NULL, correct INTEGER NOT NULL, attempts INTEGER NOT NULL, errors INTEGER NOT NULL, corrections INTEGER NOT NULL, wpm REAL NOT NULL, cpm REAL NOT NULL, accuracy REAL NOT NULL);
CREATE TABLE character_stats(session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE, rune TEXT NOT NULL, samples INTEGER NOT NULL, errors INTEGER NOT NULL, latency_ms REAL NOT NULL, PRIMARY KEY(session_id,rune));
INSERT INTO schema_migrations(version) VALUES(1);`

func (s *SQLite) migrate() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY)`); err != nil {
		return err
	}
	var version int
	if err = tx.QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&version); err != nil {
		return err
	}
	if version > 5 {
		return fmt.Errorf("database schema %d is newer than supported schema 5", version)
	}
	if version < 1 {
		if _, err = tx.Exec(schemaV1); err != nil {
			return err
		}
	}
	if version < 2 {
		_, err = tx.Exec(`ALTER TABLE progress ADD COLUMN unlocked INTEGER NOT NULL DEFAULT 0;
ALTER TABLE progress ADD COLUMN mastered INTEGER NOT NULL DEFAULT 0;
ALTER TABLE character_stats ADD COLUMN latency_samples INTEGER;
ALTER TABLE sessions ADD COLUMN attempt_id TEXT;
CREATE UNIQUE INDEX sessions_attempt_id ON sessions(attempt_id);
CREATE INDEX sessions_filter ON sessions(language,mode,started_at);
UPDATE progress SET mastered=1 WHERE mastery_streak>=2;`)
		if err != nil {
			return err
		}
		for language, profile := range trainer.Profiles() {
			last := min(5, len(profile.UnlockOrder)-1)
			for i, r := range profile.UnlockOrder {
				var samples int
				if err = tx.QueryRow(`SELECT COALESCE((SELECT samples FROM progress WHERE language=? AND rune=?),0)`, language, string(r)).Scan(&samples); err != nil {
					return err
				}
				if samples > 0 && i > last {
					last = i
				}
			}
			for _, r := range profile.UnlockOrder[:last+1] {
				_, err = tx.Exec(`INSERT INTO progress(language,rune,samples,errors,latency_ms,accuracy,confidence,mastery_streak,unlocked,mastered) VALUES(?,?,0,0,0,0,0,0,1,0) ON CONFLICT(language,rune) DO UPDATE SET unlocked=1`, language, string(r))
				if err != nil {
					return err
				}
			}
		}
		if _, err = tx.Exec(`INSERT INTO schema_migrations(version) VALUES(2)`); err != nil {
			return err
		}
	}
	if version < 3 {
		_, err = tx.Exec(`CREATE TABLE practice_time(attempt_id TEXT NOT NULL, day TEXT NOT NULL, duration_ns INTEGER NOT NULL CHECK(duration_ns>=0), PRIMARY KEY(attempt_id,day));
INSERT INTO practice_time(attempt_id,day,duration_ns) SELECT COALESCE(attempt_id,'legacy-'||id),date(started_at,'localtime'),MAX(0,duration_ms)*1000000 FROM sessions;
INSERT INTO schema_migrations(version) VALUES(3);`)
		if err != nil {
			return err
		}
	}
	if version < 4 {
		_, err = tx.Exec(`ALTER TABLE sessions ADD COLUMN target_skill TEXT NOT NULL DEFAULT '';
CREATE TABLE skills(
language TEXT NOT NULL,
kind TEXT NOT NULL CHECK(kind IN ('rune','bigram')),
pattern TEXT NOT NULL,
samples INTEGER NOT NULL,
errors INTEGER NOT NULL,
latency_samples INTEGER NOT NULL,
latency_ms REAL NOT NULL,
accuracy REAL NOT NULL,
confidence REAL NOT NULL,
level INTEGER NOT NULL,
last_practiced TEXT NOT NULL DEFAULT '',
due_at TEXT NOT NULL DEFAULT '',
PRIMARY KEY(language,kind,pattern));
INSERT INTO skills(language,kind,pattern,samples,errors,latency_samples,latency_ms,accuracy,confidence,level)
SELECT language,'rune',rune,samples,errors,COALESCE(samples-errors,0),latency_ms,accuracy,confidence,CASE WHEN mastered=1 THEN 1 ELSE 0 END FROM progress;
INSERT INTO schema_migrations(version) VALUES(4);`)
		if err != nil {
			return err
		}
	}
	if version < 5 {
		_, err = tx.Exec(`CREATE TABLE skills_v5(
language TEXT NOT NULL,
kind TEXT NOT NULL CHECK(kind IN ('rune','bigram','number','uppercase','punctuation')),
pattern TEXT NOT NULL,
samples INTEGER NOT NULL,
errors INTEGER NOT NULL,
latency_samples INTEGER NOT NULL,
latency_ms REAL NOT NULL,
accuracy REAL NOT NULL,
confidence REAL NOT NULL,
level INTEGER NOT NULL,
last_practiced TEXT NOT NULL DEFAULT '',
due_at TEXT NOT NULL DEFAULT '',
PRIMARY KEY(language,kind,pattern));
INSERT INTO skills_v5 SELECT * FROM skills;
DROP TABLE skills;
ALTER TABLE skills_v5 RENAME TO skills;
INSERT INTO schema_migrations(version) VALUES(5);`)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
