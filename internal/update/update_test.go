package update

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestCheck(t *testing.T) {
	tests := []struct {
		current, body, want string
		status              int
	}{
		{"v1.2.9", `{"tag_name":"v1.3.0"}`, "v1.3.0", 200},
		{"v1.10.0", `{"tag_name":"v1.9.0"}`, "", 200},
		{"v1.3.0", `{"tag_name":"v1.3.0"}`, "", 200},
		{"dev", `{"tag_name":"v2.0.0"}`, "", 200},
		{"v1.0.0", `{"tag_name":"v2.0.0","prerelease":true}`, "", 200},
		{"v1.0.0", `{"tag_name":"v2.0.0"}`, "", 403},
		{"v1.0.0", `invalid`, "", 200},
	}
	for _, tt := range tests {
		t.Run(tt.current+tt.body, func(t *testing.T) {
			client := &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tt.status, Body: io.NopCloser(strings.NewReader(tt.body))}, nil
			})}
			if got := Check(context.Background(), client, tt.current); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInstallVerifiesBeforeReplacing(t *testing.T) {
	for _, valid := range []bool{true, false} {
		t.Run(fmt.Sprint(valid), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "btyper")
			if err := os.WriteFile(path, []byte("old"), 0755); err != nil {
				t.Fatal(err)
			}
			binary := []byte("new binary")
			sum := sha256.Sum256(binary)
			if !valid {
				sum[0] ^= 1
			}
			client := &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
				var body string
				switch {
				case r.URL.Host == "api.github.com":
					body = `{"tag_name":"v2.0.0"}`
				case strings.HasSuffix(r.URL.Path, "/checksums.txt"):
					body = fmt.Sprintf("%x  btyper-linux-amd64\n", sum)
				default:
					body = string(binary)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}
			version, err := Install(context.Background(), client, "v1.0.0", path, "linux", "amd64")
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if valid && (err != nil || version != "v2.0.0" || string(data) != string(binary)) {
				t.Fatalf("version=%q err=%v data=%q", version, err, data)
			}
			if !valid && (err == nil || string(data) != "old") {
				t.Fatalf("failed verification replaced binary: %q, %v", data, err)
			}
		})
	}
}
