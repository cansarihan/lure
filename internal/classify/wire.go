package classify

import (
	"strings"

	"github.com/cansarihan/lure/internal/event"
)

func init() {
	add("tls-on-plain-port", Notice, "opens with a TLS handshake on a port that speaks plain text",
		func(e event.Event) bool {
			b := field(e, "hex")
			return e.Sensor != "http" && strings.HasPrefix(b, "1603")
		})
	add("http-on-other-port", Notice, "sends an HTTP request to a service that is not HTTP",
		func(e event.Event) bool {
			s := field(e, "text")
			return e.Sensor != "http" && (strings.HasPrefix(s, "GET ") || strings.HasPrefix(s, "POST ") || strings.HasPrefix(s, "HEAD "))
		})
	add("redis-probe", Notice, "speaks the Redis protocol, looking for an open cache",
		func(e event.Event) bool {
			s := strings.ToUpper(field(e, "text"))
			return strings.HasPrefix(s, "*1\r\n") || strings.HasPrefix(s, "INFO") || strings.Contains(s, "CONFIG GET")
		})
	add("ssh-client", Info, "identifies itself as an SSH client and its software",
		func(e event.Event) bool { return strings.HasPrefix(field(e, "client"), "SSH-") })
}
