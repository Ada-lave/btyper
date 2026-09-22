package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"btyper/internal/domain"
)

func TestBackupRoundTripAndInvalidImportRollback(t *testing.T) {
	s := testDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	r := sampleResult("portable", "en", now)
	r.Mode = domain.ModeAdaptive
	r.TargetSkill = "ee"
	skills := map[string]domain.Skill{"bigram:ee": {Language: "en", Kind: domain.SkillBigram, Pattern: "ee", Samples: 8, Accuracy: 1, Confidence: .8, Level: 2, LastPracticed: now, DueAt: now.Add(3 * 24 * time.Hour)}}
	progress := map[rune]domain.CharacterProgress{'e': {Language: "en", Rune: 'e', Samples: 8, Unlocked: true}}
	if err := s.SaveAdaptiveSession(r, progress, skills); err != nil {
		t.Fatal(err)
	}
	if err := s.SavePracticeTime([]domain.PracticeTime{{AttemptID: "portable", Day: now.Format("2006-01-02"), Duration: time.Second}}); err != nil {
		t.Fatal(err)
	}
	var encoded bytes.Buffer
	if err := s.ExportBackup(&encoded); err != nil {
		t.Fatal(err)
	}
	dst := testDB(t)
	if err := dst.ImportBackup(bytes.NewReader(encoded.Bytes())); err != nil {
		t.Fatal(err)
	}
	h, err := dst.History(domain.HistoryFilter{})
	if err != nil || len(h) != 1 {
		t.Fatal(h, err)
	}
	got, err := dst.LoadSkills("en")
	if err != nil || got["bigram:ee"].Level != 2 {
		t.Fatal(got, err)
	}
	if err = dst.ImportBackup(bytes.NewBufferString(`{"format_version":99}`)); err == nil {
		t.Fatal("accepted incompatible backup")
	}
	h, err = dst.History(domain.HistoryFilter{})
	if err != nil || len(h) != 1 {
		t.Fatal("invalid import changed data", h, err)
	}
}

func TestCSVExport(t *testing.T) {
	s := testDB(t)
	if err := s.SaveSession(sampleResult("csv", "en", time.Now()), nil); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := s.ExportCSV(dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sessions.csv", "practice_time.csv", "skills.csv"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || len(data) == 0 {
			t.Fatal(name, err)
		}
	}
}
