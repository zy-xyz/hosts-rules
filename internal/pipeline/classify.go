package pipeline

import (
	"net"
	"regexp"
	"strings"
)

// Bucket is the consumer-facing output a rule belongs to.
type Bucket int

const (
	BucketAccelerate Bucket = iota // hosts format, /etc/hosts or accelerate DNS
	BucketEasylist                 // AdGuard Home DNS-level rules
	BucketAdblock                  // uBlock Origin full syntax
)

// aghModifiers is the AGH DNS syntax whitelist (8 items).
// A rule carrying any modifier outside this set is ignored wholesale by AGH.
var aghModifiers = map[string]bool{
	"client": true, "denyallow": true, "dnstype": true, "dnsrewrite": true,
	"important": true, "badfilter": true, "ctag": true, "respgeo": true,
}

// reservedNames must never appear in any output bucket.
var reservedNames = map[string]bool{
	"localhost": true, "ip6-localhost": true, "ip6-loopback": true,
	"broadcasthost": true, "localdomain": true, "local": true,
}

var sinkRe = regexp.MustCompile(`^(0\.0\.0\.0|::1?|127\.0\.0\.\d{1,3})\s+`)

var domainRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9_-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9_-]*[a-z0-9])?)+$`)

// Routed is one input line's classification result. Empty Line means dropped.
type Routed struct {
	Bucket Bucket
	Line   string
}

func isSinkIP(ip string) bool {
	if ip == "0.0.0.0" || ip == "::" || ip == "::1" {
		return true
	}
	if p := net.ParseIP(ip); p != nil && p.IsLoopback() {
		return true
	}
	return false
}

// validDomain rejects wildcards, bare IPs and single-label tokens.
// punycode (xn--) is kept: AGH and uBO both consume it natively.
func validDomain(d string) bool {
	if d == "" || strings.ContainsAny(d, "*?") {
		return false
	}
	if net.ParseIP(d) != nil {
		return false
	}
	if !strings.Contains(d, ".") {
		return false
	}
	return domainRe.MatchString(d)
}

func isReserved(d string) bool { return reservedNames[d] }

// parseAdblockRule splits `||domain^$mods` / `@@||domain^$mods`.
// Returns ok=false when the line is not a ||-anchored host rule.
func parseAdblockRule(s string) (exc bool, domain, suffix, rawMods string, mods []string, ok bool) {
	body := s
	if strings.HasPrefix(body, "@@") {
		exc = true
		body = body[2:]
	}
	if !strings.HasPrefix(body, "||") {
		return false, "", "", "", nil, false
	}
	body = body[2:]
	modStr := ""
	if i := strings.IndexByte(body, '$'); i >= 0 {
		modStr, body = body[i+1:], body[:i]
	}
	if i := strings.IndexByte(body, '^'); i >= 0 {
		domain, suffix = body[:i], body[i:]
	} else {
		domain = body
	}
	if strings.Contains(domain, "/") {
		return exc, domain, suffix, "", nil, false // path-bearing: uBO only
	}
	if modStr != "" {
		for _, m := range strings.Split(modStr, ",") {
			mods = append(mods, strings.TrimPrefix(strings.TrimSpace(m), "~"))
		}
	}
	return exc, domain, suffix, modStr, mods, true
}

func modsAreAGH(mods []string) bool {
	for _, m := range mods {
		name := m
		if i := strings.IndexByte(name, '='); i >= 0 {
			name = name[:i]
		}
		if !aghModifiers[name] {
			return false
		}
	}
	return true
}

// RouteLine classifies one raw input line by consumer syntax.
func RouteLine(raw string) []Routed {
	line := strings.TrimSpace(raw)
	if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") ||
		strings.HasPrefix(line, "[") || strings.HasPrefix(line, "<") {
		return nil
	}
	line = strings.SplitN(line, " #", 2)[0]
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	// 1. hosts-style: IP hostname [hostname...]
	if sinkRe.MatchString(line) {
		return routeHostsLine(sinkRe.ReplaceAllString(line, "127.0.0.1 "))
	}
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		if ip := net.ParseIP(fields[0]); ip != nil {
			return routeHostsLine(line)
		}
	}

	// 2. ||domain^ / @@||domain^
	if exc, domain, suffix, rawMods, mods, ok := parseAdblockRule(line); ok {
		d := strings.ToLower(domain)
		if !validDomain(d) || isReserved(d) {
			return []Routed{{BucketAdblock, line}}
		}
		prefix := "||"
		if exc {
			prefix = "@@||"
		}
		norm := prefix + d + suffix
		if rawMods != "" {
			norm += "$" + rawMods
		}
		if len(mods) == 0 || modsAreAGH(mods) {
			return []Routed{{BucketEasylist, norm}}
		}
		return []Routed{{BucketAdblock, line}}
	}

	// 3. /regex/ — AGH supports regex patterns
	if strings.HasPrefix(line, "/") && strings.HasSuffix(line, "/") && !strings.Contains(line, "$") {
		return []Routed{{BucketEasylist, line}}
	}

	// 4. everything else (cosmetic, browser-only, fragments) -> uBO
	return []Routed{{BucketAdblock, line}}
}

// routeHostsLine expands a hosts line into one rule per hostname.
func routeHostsLine(line string) []Routed {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil
	}
	ip := fields[0]
	sink := isSinkIP(ip)
	if !sink && net.ParseIP(ip) == nil {
		return nil
	}
	var out []Routed
	for _, h := range fields[1:] {
		if strings.HasPrefix(h, "#") {
			break
		}
		d := strings.ToLower(strings.TrimSuffix(h, "."))
		if isReserved(d) || !validDomain(d) {
			continue
		}
		if sink {
			out = append(out, Routed{BucketEasylist, "||" + d + "^"})
		} else {
			out = append(out, Routed{BucketAccelerate, ip + " " + d})
		}
	}
	return out
}

// Classify routes all lines into the three consumer buckets.
func Classify(lines []string) (acc, easy, ad []string) {
	for _, l := range lines {
		for _, r := range RouteLine(l) {
			switch r.Bucket {
			case BucketAccelerate:
				acc = append(acc, r.Line)
			case BucketEasylist:
				easy = append(easy, r.Line)
			case BucketAdblock:
				ad = append(ad, r.Line)
			}
		}
	}
	return
}
