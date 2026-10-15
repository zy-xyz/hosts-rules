package pipeline

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ReadLines reads a text file, tolerating CRLF and a leading BOM.
func ReadLines(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var out []string
	for sc.Scan() {
		s := strings.TrimSuffix(strings.TrimPrefix(sc.Text(), "\ufeff"), "\r")
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// StripComments drops blank lines and `#` comments (source list + diy rules).
func StripComments(lines []string) []string {
	out := lines[:0]
	for _, l := range lines {
		if t := strings.TrimSpace(l); t != "" && !strings.HasPrefix(t, "#") {
			out = append(out, t)
		}
	}
	return out
}

// PreviousCounts reads the last history record so the gate can compare runs.
func PreviousCounts(statsDir, suffix string) (BucketStats, bool) {
	name := "history.jsonl"
	if suffix == "_full" {
		name = "history_full.jsonl"
	}
	f, err := os.Open(filepath.Join(statsDir, name))
	if err != nil {
		return BucketStats{}, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 16*1024*1024)
	last := ""
	for sc.Scan() {
		if t := strings.TrimSpace(sc.Text()); len(t) > 2 {
			last = t
		}
	}
	if last == "" {
		return BucketStats{}, false
	}
	var s Stats
	if json.Unmarshal([]byte(last), &s) != nil {
		return BucketStats{}, false
	}
	return s.Buckets, true
}
