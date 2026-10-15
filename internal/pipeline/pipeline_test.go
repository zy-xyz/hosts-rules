package pipeline

import "testing"

func one(in string) (Bucket, string) {
	rs := RouteLine(in)
	if len(rs) == 0 {
		return -1, ""
	}
	return rs[0].Bucket, rs[0].Line
}

func TestRouting(t *testing.T) {
	cases := []struct {
		in     string
		bucket Bucket
		want   string
	}{
		{"127.0.0.1 ads.example.com", BucketEasylist, "||ads.example.com^"},
		{"0.0.0.0 ads.example.com", BucketEasylist, "||ads.example.com^"},
		{"0.0.0.0\tads.example.com", BucketEasylist, "||ads.example.com^"},
		{"0.0.0.0 Ads.Example.COM", BucketEasylist, "||ads.example.com^"},
		{":: ads.example.com", BucketEasylist, "||ads.example.com^"},
		{"::1 owu.se", BucketEasylist, "||owu.se^"},
		{"1.2.3.4 ad-maven.com", BucketAccelerate, "1.2.3.4 ad-maven.com"},
		{"0.0.0.0 ads.example.com # note", BucketEasylist, "||ads.example.com^"},

		{"::1 ip6-localhost", -1, ""},
		{"127.0.0.2 localhost", -1, ""},
		{"0.0.0.0 broadcasthost", -1, ""},
		{"0.0.0.0 com", -1, ""},

		{"||domain.com^", BucketEasylist, "||domain.com^"},
		{"@@||domain.com^", BucketEasylist, "@@||domain.com^"},
		{"||domain.com^$important", BucketEasylist, "||domain.com^$important"},
		{"@@||good.com^$important", BucketEasylist, "@@||good.com^$important"},
		{"||domain.com^$dnsrewrite=1.2.3.4", BucketEasylist, "||domain.com^$dnsrewrite=1.2.3.4"},
		{"||domain.com^$third-party", BucketAdblock, "||domain.com^$third-party"},
		{"@@||domain.com^$generichide", BucketAdblock, "@@||domain.com^$generichide"},
		{"||domain.com/path.js$important", BucketAdblock, "||domain.com/path.js$important"},
		{"||*.doubleclick.net^", BucketAdblock, "||*.doubleclick.net^"},
		{"/adframe[0-9]+/", BucketEasylist, "/adframe[0-9]+/"},
		{"example.org##.ad", BucketAdblock, "example.org##.ad"},
		{"! c", -1, ""},
		{"# c", -1, ""},
	}
	for _, c := range cases {
		gb, gl := one(c.in)
		if gb != c.bucket || gl != c.want {
			t.Errorf("RouteLine(%q) = (%v,%q), want (%v,%q)", c.in, gb, gl, c.bucket, c.want)
		}
	}
}

func TestMultiHostExpansion(t *testing.T) {
	rs := RouteLine("127.0.0.1 a.com b.com c.com")
	if len(rs) != 3 {
		t.Fatalf("got %d rules, want 3", len(rs))
	}
	for i, want := range []string{"||a.com^", "||b.com^", "||c.com^"} {
		if rs[i].Line != want {
			t.Errorf("rule %d = %q, want %q", i, rs[i].Line, want)
		}
	}
}

func TestDomainOf(t *testing.T) {
	for in, want := range map[string]string{
		"||a.com^":       "a.com",
		"@@||b.com^":     "b.com",
		"@@||c.com^$x":   "c.com",
		"1.2.3.4 d.com":  "d.com",
		"||e.com/path$y": "e.com",
	} {
		if got := DomainOf(in); got != want {
			t.Errorf("DomainOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDedupe(t *testing.T) {
	got := Dedupe([]string{"a", "b", "a", "", "c", "b"})
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d = %q want %q", i, got[i], want[i])
		}
	}
}
