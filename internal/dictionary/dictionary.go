// Package dictionary manages optional FrequencyWords word lists.
package dictionary

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"btyper/internal/trainer"
)

const MaxBytes = 4 << 20

const attribution = "# FrequencyWords by Hermit Dave, OpenSubtitles 2018\n# https://github.com/hermitdave/FrequencyWords\n# CC BY-SA 4.0: https://creativecommons.org/licenses/by-sa/4.0/\n# btyper filters words by alphabet and length; frequencies are not used.\n"

func Source(language string) (string, error) {
	if language != "en" && language != "ru" {
		return "", fmt.Errorf("unsupported dictionary language %q (en, ru)", language)
	}
	return "https://raw.githubusercontent.com/hermitdave/FrequencyWords/master/content/2018/" + language + "/" + language + "_50k.txt", nil
}

// Parse keeps unique words of 2–10 letters in frequency order.
func Parse(reader io.Reader, language string) ([]string, error) {
	if _, err := Source(language); err != nil {
		return nil, err
	}
	alphabet := string(trainer.Profiles()[language].UnlockOrder)
	data, err := io.ReadAll(io.LimitReader(reader, MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxBytes || !utf8.Valid(data) {
		return nil, errors.New("dictionary exceeds size limit or contains invalid UTF-8")
	}
	var words []string
	seen := map[string]bool{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, errors.New("invalid dictionary row: expected word and frequency")
		}
		frequency, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil || frequency == 0 {
			return nil, errors.New("invalid dictionary frequency")
		}
		word := strings.ToLower(fields[0])
		valid := utf8.RuneCountInString(word) >= 2 && utf8.RuneCountInString(word) <= 10
		for _, r := range word {
			valid = valid && strings.ContainsRune(alphabet, r)
		}
		if valid && !seen[word] {
			seen[word] = true
			words = append(words, word)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(words) == 0 {
		return nil, errors.New("dictionary contains no usable words")
	}
	return words, nil
}

func Load(dir, language string) ([]string, error) {
	if _, err := Source(language); err != nil {
		return nil, err
	}
	file, err := os.Open(filepath.Join(dir, "dictionaries", language+".txt"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return Parse(file, language)
}

func Download(ctx context.Context, client *http.Client, dir, language string) (int, error) {
	url, err := Source(language)
	if err != nil {
		return 0, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("dictionary download: HTTP %d", response.StatusCode)
	}
	words, err := Parse(response.Body, language)
	if err != nil {
		return 0, err
	}
	directory := filepath.Join(dir, "dictionaries")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return 0, err
	}
	file, err := os.CreateTemp(directory, ".download-*")
	if err != nil {
		return 0, err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	var data strings.Builder
	data.WriteString(attribution)
	for _, word := range words {
		data.WriteString(word + " 1\n")
	}
	if _, err = io.WriteString(file, data.String()); err != nil {
		_ = file.Close()
		return 0, err
	}
	if err := file.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(file.Name(), filepath.Join(directory, language+".txt")); err != nil {
		return 0, err
	}
	return len(words), nil
}
