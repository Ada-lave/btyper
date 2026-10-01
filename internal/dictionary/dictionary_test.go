package dictionary

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestParseFiltersWords(t *testing.T) {
	for language, input := range map[string]string{
		"en": "TRAIN 30\ntrain 20\na 10\ncan't 9\n123 8\nрусский 7\nextraordinary 6\nletter 5\n",
		"ru": "СЛОВО 30\nслово 20\nя 10\nиз-за 9\n123 8\nenglish 7\nэлектростанция 6\nбуква 5\n",
	} {
		words, err := Parse(strings.NewReader(input), language)
		want := []string{"train", "letter"}
		if language == "ru" {
			want = []string{"слово", "буква"}
		}
		if err != nil || !reflect.DeepEqual(words, want) {
			t.Fatalf("%s: words=%v err=%v", language, words, err)
		}
	}
}

func TestDownloadPreservesDictionaryOnFailure(t *testing.T) {
	dir := t.TempDir()
	status, body := http.StatusOK, "train 30\nletter 20\n"
	client := &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		want, _ := Source("en")
		if r.URL.String() != want {
			t.Fatalf("unexpected URL %s", r.URL)
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	if count, err := Download(context.Background(), client, dir, "en"); err != nil || count != 2 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	path := filepath.Join(dir, "dictionaries", "en.txt")
	original, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(original), "CC BY-SA 4.0") {
		t.Fatalf("missing attribution: %v", err)
	}
	for _, failure := range []struct {
		status int
		body   string
	}{
		{http.StatusNotFound, "missing"}, {http.StatusOK, "<html>error</html>"},
		{http.StatusOK, "train 0"}, {http.StatusOK, string([]byte{0xff})},
		{http.StatusOK, strings.Repeat("x", MaxBytes+1)},
	} {
		status, body = failure.status, failure.body
		if _, err := Download(context.Background(), client, dir, "en"); err == nil {
			t.Fatal("invalid download accepted")
		}
		current, err := os.ReadFile(path)
		if err != nil || string(current) != string(original) {
			t.Fatal("failed download changed installed dictionary")
		}
	}
	words, err := Load(dir, "en")
	if err != nil || !reflect.DeepEqual(words, []string{"train", "letter"}) {
		t.Fatalf("words=%v err=%v", words, err)
	}
	if words, err := Load(dir, "ru"); err != nil || words != nil {
		t.Fatalf("missing optional dictionary: %v %v", words, err)
	}
}
