// Package update checks for newer published releases.
package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const ReleasesURL = "https://github.com/Ada-lave/btyper/releases/latest"
const apiURL = "https://api.github.com/repos/Ada-lave/btyper/releases/latest"

// Check returns the latest version only when it is newer than current.
// Development and non-release version strings are ignored.
func Check(ctx context.Context, client *http.Client, current string) string {
	if _, ok := parse(current); !ok {
		return ""
	}
	tag, err := latest(ctx, client)
	if err != nil {
		return ""
	}

	a, ok := parse(current)
	if !ok {
		return ""
	}
	b, ok := parse(tag)
	if !ok {
		return ""
	}
	for i := range a {
		if b[i] > a[i] {
			return tag
		}
		if b[i] < a[i] {
			return ""
		}
	}
	return ""
}

func latest(ctx context.Context, client *http.Client) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("release lookup: HTTP %d", resp.StatusCode)
	}
	var release struct {
		Tag        string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&release); err != nil {
		return "", err
	}
	if _, ok := parse(release.Tag); !ok || release.Draft || release.Prerelease || !strings.HasPrefix(release.Tag, "v") {
		return "", errors.New("invalid stable release")
	}
	return release.Tag, nil
}

func parse(v string) ([3]uint64, bool) {
	var result [3]uint64
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) != 3 {
		return result, false
	}
	for i, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return result, false
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return result, false
			}
		}
		n, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return result, false
		}
		result[i] = n
	}
	return result, true
}
