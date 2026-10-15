package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// BucketStats is the per-output line count.
type BucketStats struct {
	Accelerate int `json:"accelerate"`
	Easylist   int `json:"easylist"`
	Adblock    int `json:"adblock"`
}

// SourceContrib records how much one upstream contributed.
type SourceContrib struct {
	URL      string  `json:"url"`
	Bytes    int     `json:"bytes"`
	Domains  int     `json:"domains"`
	Duration float64 `json:"duration_s"`
	Err      string  `json:"error,omitempty"`
}

// Stats is the machine-readable run summary consumed by the Hugo frontend.
type Stats struct {
	Timestamp     string          `json:"timestamp"`
	DurationS     float64         `json:"duration_s"`
	Full          bool            `json:"full"`
	TotalSources  int             `json:"total_sources"`
	FailedSources []string        `json:"failed_sources"`
	Buckets       BucketStats     `json:"buckets"`
	RawLines      int             `json:"raw_lines"`
	SourceContrib []SourceContrib `json:"source_contrib"`
}

// BuildStats summarises one run.
func BuildStats(full bool, results []SourceResult, buckets BucketStats, rawLines int, elapsed time.Duration) Stats {
	s := Stats{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		DurationS:    elapsed.Seconds(),
		Full:         full,
		TotalSources: len(results),
		Buckets:      buckets,
		RawLines:     rawLines,
	}
	for _, r := range results {
		sc := SourceContrib{URL: r.URL, Bytes: r.Bytes, Domains: r.Domains, Duration: r.Duration.Seconds()}
		if r.Err != nil {
			sc.Err = r.Err.Error()
			s.FailedSources = append(s.FailedSources, r.URL)
		}
		s.SourceContrib = append(s.SourceContrib, sc)
	}
	sort.Slice(s.SourceContrib, func(i, j int) bool {
		return s.SourceContrib[i].Domains > s.SourceContrib[j].Domains
	})
	return s
}

func statsName(s Stats) string {
	if s.Full {
		return "stats_full.json"
	}
	return "stats.json"
}

func historyName(s Stats) string {
	if s.Full {
		return "history_full.jsonl"
	}
	return "history.jsonl"
}

// WriteStats writes the latest-run summary the frontend reads.
func WriteStats(dir string, s Stats) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, statsName(s)), append(b, '\n'), 0o644)
}

// AppendHistory adds one JSONL record for the trend chart.
func AppendHistory(dir string, s Stats) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, historyName(s)), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}
