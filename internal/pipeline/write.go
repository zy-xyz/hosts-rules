package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const homepage = "https://github.com/zy-xyz/hosts-rules"

// consumerOf names the client each output is written for.
func consumerOf(kind string) string {
	switch kind {
	case "accelerate":
		return "hosts"
	case "easylist":
		return "AdGuard Home"
	case "adblock":
		return "uBlock Origin"
	}
	return kind
}

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

// WriteOutput creates the parent directory, prepends the banner, dedupes the
// body and appends a trailing newline. The banner is never counted, deduped or
// gated — callers pass the rule count they want advertised in it.
func WriteOutput(path string, header, lines []string) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	lines = Dedupe(lines)
	if len(lines) == 0 && len(header) == 0 {
		return os.WriteFile(path, nil, 0o644)
	}
	out := make([]string, 0, len(header)+len(lines))
	out = append(out, header...)
	out = append(out, lines...)
	return os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o644)
}

// Banner builds the banner block prepended to one output file. The comment
// syntax is consumer-specific: hosts files take "#", AdGuard Home and uBlock
// Origin take "!", and uBlock additionally wants its metadata line first.
func Banner(kind string, full bool, rules, sources int, at time.Time) []string {
	scope := "base"
	if full {
		scope = "full"
	}
	prefix, rule := "# ", "#"
	if kind != "accelerate" {
		prefix, rule = "! ", "!"
	}
	line := func(s string) string {
		if s == "" {
			return rule
		}
		return prefix + s
	}
	bar := line(strings.Repeat("=", 58))
	body := []string{
		bar,
		line(fmt.Sprintf("hosts-rules · %s · %s · %s", kind, consumerOf(kind), scope)),
		line("多上游广告/跟踪域名规则，按消费端语法路由"),
		line("Homepage: " + homepage),
		line(fmt.Sprintf("Version: %s · Expires: 1 day", at.UTC().Format("20060102150405"))),
		line(fmt.Sprintf("Rules: %d · Sources: %d (%s) · License: MIT", rules, sources, scope)),
		bar,
	}
	if kind == "adblock" {
		body = append([]string{"[Adblock Plus 2.0]"}, body...)
	}
	return body
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
