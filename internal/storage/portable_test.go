package storage

import (
	"bytes"
	"encoding/json"
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
	skills := map[string]domain.Skill{
		"bigram:ee": {Language: "en", Kind: domain.SkillBigram, Pattern: "ee", Samples: 8, Accuracy: 1, Confidence: .8, Level: 2, LastPracticed: now, DueAt: now.Add(3 * 24 * time.Hour)},
		"number:7":  {Language: "en", Kind: domain.SkillNumber, Pattern: "7", Samples: 6, Accuracy: 1, Confidence: .7, Level: 1},
	}
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
	if err != nil || got["bigram:ee"].Level != 2 || got["number:7"].Samples != 6 {
		t.Fatal(got, err)
	}
	if err = dst.ImportBackup(bytes.NewBufferString(`{"format_version":99}`)); err == nil {
		t.Fatal("accepted incompatible backup")
	}
	h, err = dst.History(domain.HistoryFilter{})
	if err != nil || len(h) != 1 {
		t.Fatal("invalid import changed data", h, err)
	}
	var backup Backup
	if err := json.Unmarshal(encoded.Bytes(), &backup); err != nil {
		t.Fatal(err)
	}
	backup.Progress = append(backup.Progress, backup.Progress[0])
	malformed, err := json.Marshal(backup)
	if err != nil {
		t.Fatal(err)
	}
	if err := dst.ImportBackup(bytes.NewReader(malformed)); err == nil {
		t.Fatal("accepted duplicate progress after deleting existing rows")
	}
	h, err = dst.History(domain.HistoryFilter{})
	if err != nil || len(h) != 1 {
		t.Fatal("failed import did not roll back existing history", h, err)
	}
	got, err = dst.LoadSkills("en")
	if err != nil || got["bigram:ee"].Level != 2 {
		t.Fatal("failed import did not roll back skills", got, err)
	}
}

func TestImportLegacyBackupDefaultsOptionalTrainingCategoriesOn(t *testing.T) {
	s := testDB(t)
	legacy := `{"format_version":1,"settings":{"Language":"en"}}`
	if err := s.ImportBackup(bytes.NewBufferString(legacy)); err != nil {
		t.Fatal(err)
	}
	settings, err := s.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !settings.TrainNumbers || !settings.TrainUppercase || !settings.TrainPunctuation {
		t.Fatalf("legacy backup disabled new training categories: %+v", settings)
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
