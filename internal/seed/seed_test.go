package seed

import (
	"net/netip"
	"testing"
	"time"
)

func TestEventsDeterministicAndDocumented(t *testing.T) {
	now := time.Date(2026, 3, 1, 18, 0, 0, 0, time.UTC)
	a, b := Events(now, 300), Events(now, 300)
	if len(a) != 300 {
		t.Fatalf("got %d events", len(a))
	}
	docs := []netip.Prefix{
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"),
	}
	for i := range a {
		if a[i].Remote != b[i].Remote || !a[i].Time.Equal(b[i].Time) {
			t.Fatal("seed is not deterministic")
		}
		addr := netip.MustParseAddr(a[i].Remote)
		ok := false
		for _, p := range docs {
			ok = ok || p.Contains(addr)
		}
		if !ok {
			t.Errorf("%s is outside the RFC 5737 documentation ranges", addr)
		}
		if a[i].Time.After(now) || now.Sub(a[i].Time) > 24*time.Hour {
			t.Errorf("event at %v is outside the last day", a[i].Time)
		}
	}
}
