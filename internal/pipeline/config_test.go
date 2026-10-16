package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// diy.txt 由 main 单独读入；若它同时也出现在 links 里就会被注入两次。
func TestLinksDoNotSelfReferenceDIY(t *testing.T) {
	for _, name := range []string{"links.txt", "links_full.txt"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "config", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, l := range strings.Split(string(b), "\n") {
			if strings.Contains(l, "diy.txt") {
				t.Errorf("%s 引用了 diy.txt，会与 main 的本地加载重复: %s", name, l)
			}
		}
	}
}

// 指向自己仓库的 raw 链接会在仓库改名/归档后静默 404。
func TestLinksDoNotPointAtArchivedRepo(t *testing.T) {
	for _, name := range []string{"links.txt", "links_full.txt"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "config", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(b), "hosts-new") {
			t.Errorf("%s 仍指向已归档的 hosts-new 仓库", name)
		}
	}
}

func TestLinksHaveNoDuplicates(t *testing.T) {
	for _, name := range []string{"links.txt", "links_full.txt"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "config", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		seen := map[string]bool{}
		for _, l := range strings.Split(string(b), "\n") {
			l = strings.TrimSpace(l)
			if l == "" {
				continue
			}
			if seen[l] {
				t.Errorf("%s 重复链接: %s", name, l)
			}
			seen[l] = true
		}
	}
}
