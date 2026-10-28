package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Install downloads and verifies the latest release before atomically replacing executable.
// It returns an empty version if the installed release is already current or newer.
func Install(ctx context.Context, client *http.Client, current, executable, goos, arch string) (string, error) {
	currentVersion, ok := parse(current)
	if !ok {
		return "", errors.New("self-update requires a release build")
	}
	if (goos != "linux" && goos != "darwin") || (arch != "amd64" && arch != "arm64") {
		return "", fmt.Errorf("self-update is unsupported on %s/%s", goos, arch)
	}
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	tag, err := latest(ctx, client)
	if err != nil {
		return "", err
	}
	newVersion, _ := parse(tag)
	newer := false
	for i := range currentVersion {
		if newVersion[i] != currentVersion[i] {
			newer = newVersion[i] > currentVersion[i]
			break
		}
	}
	if !newer {
		return "", nil
	}
	name := "btyper-" + goos + "-" + arch
	base := "https://github.com/Ada-lave/btyper/releases/download/" + tag + "/"
	checksums, err := download(ctx, client, base+"checksums.txt", 64<<10)
	if err != nil {
		return "", err
	}
	var expected []byte
	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			if expected != nil {
				return "", errors.New("duplicate checksum")
			}
			expected, err = hex.DecodeString(fields[0])
			if err != nil || len(expected) != sha256.Size {
				return "", errors.New("invalid checksum")
			}
		}
	}
	if expected == nil {
		return "", errors.New("release checksum missing")
	}
	binary, err := download(ctx, client, base+name, 100<<20)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(binary)
	if !equal(sum[:], expected) {
		return "", errors.New("checksum verification failed")
	}
	path, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("executable is not a regular file")
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".btyper-update-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(temp.Name()) }()
	if _, err = temp.Write(binary); err == nil {
		err = temp.Chmod(info.Mode().Perm())
	}
	if err == nil {
		err = temp.Sync()
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	if err = os.Rename(temp.Name(), path); err != nil {
		return "", err
	}
	return tag, nil
}

func download(ctx context.Context, client *http.Client, url string, max int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, errors.New("download exceeds size limit")
	}
	return data, nil
}
func equal(a, b []byte) bool { return string(a) == string(b) }

// CurrentPlatform returns the running binary's target platform.
func CurrentPlatform() (string, string) { return runtime.GOOS, runtime.GOARCH }
