package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DefaultUA avoids upstream rate-limiting that triggers on empty user agents.
const DefaultUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"

// SourceResult carries one upstream fetch outcome.
type SourceResult struct {
	URL      string
	Path     string
	Bytes    int
	Domains  int
	Duration time.Duration
	Err      error
}

// CachePath maps a URL to its cache file name.
func CachePath(cacheDir, url string) string {
	sum := sha256.Sum256([]byte(url))
	return filepath.Join(cacheDir, hex.EncodeToString(sum[:8])+".txt")
}

// FetchAll downloads urls concurrently and writes each body into cacheDir.
// Cached non-empty files are reused, so a rerun only refetches what is missing.
func FetchAll(urls []string, cacheDir string, workers int) []SourceResult {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir %s: %v\n", cacheDir, err)
	}
	results := make([]SourceResult, len(urls))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	client := &http.Client{Timeout: 30 * time.Second}

	for i, u := range urls {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			path := CachePath(cacheDir, u)
			if fi, err := os.Stat(path); err == nil && fi.Size() > 0 {
				b, _ := os.ReadFile(path)
				results[i] = SourceResult{URL: u, Path: path, Bytes: len(b)}
				return
			}

			start := time.Now()
			req, err := http.NewRequest("GET", u, nil)
			if err != nil {
				results[i] = SourceResult{URL: u, Path: path, Err: err}
				return
			}
			req.Header.Set("User-Agent", DefaultUA)
			resp, err := client.Do(req)
			if err != nil {
				results[i] = SourceResult{URL: u, Path: path, Duration: time.Since(start), Err: err}
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				results[i] = SourceResult{URL: u, Path: path, Duration: time.Since(start),
					Err: fmt.Errorf("HTTP %d", resp.StatusCode)}
				return
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				results[i] = SourceResult{URL: u, Path: path, Duration: time.Since(start), Err: err}
				return
			}
			// Atomic write: a crash never leaves a half file that a rerun would trust.
			tmp := path + ".tmp"
			if err := os.WriteFile(tmp, body, 0o644); err != nil {
				results[i] = SourceResult{URL: u, Path: path, Duration: time.Since(start), Err: err}
				return
			}
			if err := os.Rename(tmp, path); err != nil {
				results[i] = SourceResult{URL: u, Path: path, Duration: time.Since(start), Err: err}
				return
			}
			results[i] = SourceResult{URL: u, Path: path, Bytes: len(body), Duration: time.Since(start)}
		}(i, u)
	}
	wg.Wait()
	return results
}
