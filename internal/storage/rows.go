package storage

import (
	"btyper/internal/domain"
	"fmt"
	"time"
)

type scanner interface{ Scan(...any) error }

func scanProgress(row scanner, language string) (domain.CharacterProgress, error) {
	var key string
	var p domain.CharacterProgress
	p.Language = language
	if err := row.Scan(&key, &p.Samples, &p.Errors, &p.LatencyMS, &p.Accuracy, &p.Confidence, &p.MasteryStreak, &p.Unlocked, &p.Mastered); err != nil {
		return p, err
	}
	rs := []rune(key)
	if len(rs) != 1 {
		return p, fmt.Errorf("invalid rune in progress row")
	}
	p.Rune = rs[0]
	return p, nil
}
func scanHistory(row scanner) (domain.HistoryEntry, error) {
	var h domain.HistoryEntry
	var stamp, mode string
	var ms int64
	if err := row.Scan(&h.ID, &stamp, &mode, &h.Language, &h.WPM, &h.Accuracy, &h.Errors, &ms); err != nil {
		return h, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, stamp)
	if err != nil {
		return h, fmt.Errorf("session timestamp: %w", err)
	}
	h.StartedAt = parsed
	h.Mode = domain.Mode(mode)
	h.Duration = time.Duration(ms) * time.Millisecond
	return h, nil
}
