package pipeline

import (
	"os"
	"path/filepath"
	"strings"
)

// Dedupe removes duplicate lines while preserving first-seen order.
func Dedupe(lines []string) []string {
	seen := make(map[string]struct{}, len(lines))
	out := lines[:0:0]
	for _, l := range lines {
		if l == "" {
			continue
		}
		if _, ok := seen[l]; ok {
			continue
		}
		seen[l] = struct{}{}
		out = append(out, l)
	}
	return out
}

// WriteOutput creates the parent directory, dedupes and appends a trailing newline.
func WriteOutput(path string, lines []string) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	lines = Dedupe(lines)
	if len(lines) == 0 {
		return os.WriteFile(path, nil, 0o644)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

// DomainOf extracts the hostname token a consumer would act on.
func DomainOf(line string) string {
	s := line
	if strings.HasPrefix(s, "@@||") {
		s = s[4:]
	} else if strings.HasPrefix(s, "||") {
		s = s[2:]
	} else {
		f := strings.Fields(s)
		if len(f) >= 2 {
			return strings.ToLower(f[1])
		}
		return ""
	}
	if i := strings.IndexAny(s, "^/$"); i >= 0 {
		s = s[:i]
	}
	return strings.ToLower(s)
}

// DomainSet collects the distinct domain tokens of a bucket.
func DomainSet(lines []string) map[string]struct{} {
	m := make(map[string]struct{}, len(lines))
	for _, l := range lines {
		if d := DomainOf(l); d != "" {
			m[d] = struct{}{}
		}
	}
	return m
}

// CountDomains counts distinct domain tokens in one source's lines.
func CountDomains(lines []string) int {
	seen := make(map[string]struct{}, 256)
	for _, l := range lines {
		if d := DomainOf(l); d != "" {
			seen[d] = struct{}{}
		}
	}
	return len(seen)
}
