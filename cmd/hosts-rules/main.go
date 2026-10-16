// Command hosts-rules builds the three consumer rule files from upstream lists.
//
// Every rule is routed by what the consumer can actually parse, never by a
// string feature of the rule itself. See docs/adr/0003.
package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/zy-xyz/hosts-rules/internal/pipeline"
)

const workers = 32

func main() {
	full := flag.Bool("full", false, "use the extended source list (config/links_full.txt)")
	configDir := flag.String("config", "config", "directory holding links*.txt and diy.txt")
	statsDir := flag.String("stats", "docs/static/data", "frontend data directory")
	cacheDir := flag.String("cache", "cache", "upstream download cache directory")
	outDir := flag.String("out", "dist", "directory to write the three rule files into")
	noGate := flag.Bool("no-gate", false, "skip the output-ratio safety gate")
	flag.Parse()

	linksFile, suffix := "links.txt", ""
	if *full {
		linksFile, suffix = "links_full.txt", "_full"
	}

	urls := pipeline.StripComments(pipeline.ReadLines(filepath.Join(*configDir, linksFile)))
	if len(urls) == 0 {
		log.Fatalf("no sources in %s", filepath.Join(*configDir, linksFile))
	}
	fmt.Printf("sources=%d full=%v\n", len(urls), *full)

	start := time.Now()
	results := pipeline.FetchAll(urls, *cacheDir, workers)

	var all []string
	for i := range results {
		if results[i].Err != nil || results[i].Path == "" {
			continue
		}
		ls := pipeline.ReadLines(results[i].Path)
		results[i].Domains = pipeline.CountDomains(ls)
		all = append(all, ls...)
	}
	diy := pipeline.StripComments(pipeline.ReadLines(filepath.Join(*configDir, "diy.txt")))
	all = append(all, diy...)
	fmt.Printf("raw_lines=%d diy=%d\n", len(all), len(diy))

	fetchElapsed := time.Since(start)

	acc, easy, ad := pipeline.Classify(all)

	// A domain may never be accelerated and blocked at once.
	blocked := pipeline.DomainSet(easy)
	for d := range pipeline.DomainSet(ad) {
		blocked[d] = struct{}{}
	}
	kept, conflicts := acc[:0], 0
	for _, l := range acc {
		if _, bad := blocked[pipeline.DomainOf(l)]; bad {
			conflicts++
			continue
		}
		kept = append(kept, l)
	}
	acc = kept
	fmt.Printf("acc_conflicts_removed=%d\n", conflicts)

	acc, easy, ad = pipeline.Dedupe(acc), pipeline.Dedupe(easy), pipeline.Dedupe(ad)
	buckets := pipeline.BucketStats{Accelerate: len(acc), Easylist: len(easy), Adblock: len(ad)}

	// Safety gate: refuse to publish a run that lost over half of any bucket.
	if prev, havePrev := pipeline.PreviousCounts(*statsDir, suffix); !*noGate && havePrev {
		for _, c := range []struct {
			name     string
			now, had int
		}{
			{"accelerate", buckets.Accelerate, prev.Accelerate},
			{"easylist", buckets.Easylist, prev.Easylist},
			{"adblock", buckets.Adblock, prev.Adblock},
		} {
			if c.had > 100 && c.now < c.had/2 {
				log.Fatalf("GATE: %s.txt dropped from %d to %d lines (%.1f%% loss) — refusing to publish",
					c.name, c.had, c.now, 100*(1-float64(c.now)/float64(c.had)))
			}
		}
	}

	stamp := time.Now()
	files := []struct {
		name  string
		kind  string
		lines []string
	}{
		{"accelerate" + suffix + ".txt", "accelerate", acc},
		{"easylist" + suffix + ".txt", "easylist", easy},
		{"adblock" + suffix + ".txt", "adblock", ad},
	}
	for _, f := range files {
		banner := pipeline.Banner(f.kind, *full, len(f.lines), len(urls), stamp)
		if err := pipeline.WriteOutput(filepath.Join(*outDir, f.name), banner, f.lines); err != nil {
			log.Fatalf("write %s: %v", f.name, err)
		}
	}

	st := pipeline.BuildStats(*full, results, buckets, len(all), time.Since(start))
	if err := pipeline.WriteStats(*statsDir, st); err != nil {
		log.Fatalf("stats: %v", err)
	}
	if err := pipeline.AppendHistory(*statsDir, st); err != nil {
		log.Fatalf("history: %v", err)
	}

	failed := len(st.FailedSources)
	fmt.Printf("fetch=%.1fs total=%.1fs failed=%d\n", fetchElapsed.Seconds(), st.DurationS, failed)
	fmt.Printf("accelerate=%d easylist=%d adblock=%d\n", buckets.Accelerate, buckets.Easylist, buckets.Adblock)
	for i, f := range st.FailedSources {
		if i >= 5 {
			fmt.Printf("  ... and %d more failed sources\n", failed-5)
			break
		}
		fmt.Printf("  FAILED %s\n", f)
	}
}
