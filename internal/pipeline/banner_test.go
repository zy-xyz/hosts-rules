package pipeline

import (
	"strings"
	"testing"
	"time"
)

var stamp = time.Date(2026, 10, 16, 3, 4, 5, 0, time.UTC)

// hosts files only understand '#'; AdGuard Home and uBlock understand '!'.
func TestBannerCommentSyntaxMatchesConsumer(t *testing.T) {
	cases := []struct{ kind, want string }{
		{"accelerate", "#"},
		{"easylist", "!"},
		{"adblock", "!"},
	}
	for _, c := range cases {
		for _, line := range Banner(c.kind, false, 10, 2, stamp) {
			if c.kind == "adblock" && strings.HasPrefix(line, "[") {
				continue // "[Adblock Plus 2.0]" metadata line
			}
			if !strings.HasPrefix(line, c.want+" ") && line != c.want {
				t.Errorf("%s: banner line %q does not start with %q", c.kind, line, c.want)
			}
		}
	}
}

// uBlock ignores a list without its metadata line.
func TestAdblockBannerStartsWithMetadataLine(t *testing.T) {
	b := Banner("adblock", false, 10, 2, stamp)
	if len(b) == 0 || b[0] != "[Adblock Plus 2.0]" {
		t.Fatalf("adblock banner must open with [Adblock Plus 2.0], got %q", b[0])
	}
}

// A banner that mentions the rule count must advertise the real count, and the
// banner itself must never be counted as a rule.
func TestBannerIsNotCountedAsRules(t *testing.T) {
	rules := []string{"||a.com^", "||b.com^"}
	banner := Banner("easylist", false, len(rules), 1, stamp)
	var got []string
	for _, l := range append(append([]string{}, banner...), rules...) {
		if strings.HasPrefix(l, "!") || strings.HasPrefix(l, "[") || l == "" {
			continue
		}
		got = append(got, l)
	}
	if len(got) != len(rules) {
		t.Errorf("banner leaked into rule count: got %d, want %d", len(got), len(rules))
	}
	if !strings.Contains(strings.Join(banner, "\n"), "Rules: 2") {
		t.Errorf("banner must advertise the real rule count")
	}
}

func TestBannerReportsScope(t *testing.T) {
	if s := strings.Join(Banner("easylist", false, 1, 1, stamp), "\n"); !strings.Contains(s, "base") {
		t.Error("base scope missing")
	}
	if s := strings.Join(Banner("easylist", true, 1, 1, stamp), "\n"); !strings.Contains(s, "full") {
		t.Error("full scope missing")
	}
}
