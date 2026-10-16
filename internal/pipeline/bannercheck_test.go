package pipeline

import (
	"os"
	"strings"
	"testing"
	"time"
)

// The banner must be inert: re-feeding it to the router must produce zero rules,
// otherwise a downstream tool that re-reads our own output would emit garbage.
func TestBannerLinesAreNeverRouted(t *testing.T) {
	at := time.Date(2026, 10, 16, 3, 4, 5, 0, time.UTC)
	for _, kind := range []string{"accelerate", "easylist", "adblock"} {
		for _, l := range Banner(kind, false, 10, 2, at) {
			if got := RouteLine(l); len(got) != 0 {
				t.Errorf("%s banner line %q produced a rule: %+v", kind, l, got)
			}
		}
	}
}

// The banner must survive WriteOutput untouched (not deduped away, not reordered
// after the rules) and must be followed by a trailing newline.
func TestWriteOutputKeepsBannerFirst(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/sub/e.txt" // parent must be created too
	banner := Banner("easylist", false, 2, 1, time.Now())
	if err := WriteOutput(path, banner, []string{"||a.com^", "||b.com^"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.HasSuffix(s, "\n") {
		t.Error("missing trailing newline")
	}
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	for i, want := range banner {
		if lines[i] != want {
			t.Errorf("line %d = %q, want %q", i, lines[i], want)
		}
	}
	if lines[len(banner)] != "||a.com^" {
		t.Errorf("rules must follow the banner, got %q", lines[len(banner)])
	}
}

// An empty bucket still gets a banner, so consumers see which file is empty.
func TestEmptyBucketStillGetsBanner(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/empty.txt"
	if err := WriteOutput(path, Banner("accelerate", false, 0, 1, time.Now()), nil); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "Rules: 0") {
		t.Errorf("empty output should still carry the banner, got %q", string(b))
	}
}
