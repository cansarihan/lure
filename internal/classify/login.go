package classify

import (
	"strings"

	"github.com/cansarihan/lure/internal/event"
)

// defaultPairs are factory credentials that automated login attempts cycle
// through first.
var defaultPairs = map[string]bool{
	"admin:admin": true, "admin:password": true, "admin:1234": true, "admin:12345": true,
	"root:root": true, "root:123456": true, "root:toor": true, "root:password": true,
	"user:user": true, "guest:guest": true, "pi:raspberry": true, "ubnt:ubnt": true,
	"support:support": true, "admin:": true, "root:": true,
}

func init() {
	add("default-credentials", Alert, "tries a factory username and password pair",
		func(e event.Event) bool {
			if e.Kind != event.KindLogin {
				return false
			}
			pair := strings.ToLower(field(e, "user")) + ":" + strings.ToLower(field(e, "password"))
			return defaultPairs[pair]
		})
	add("privileged-account", Notice, "tries to log in as root or admin",
		func(e event.Event) bool {
			u := strings.ToLower(field(e, "user"))
			return e.Kind == event.KindLogin && (u == "root" || u == "admin" || u == "administrator")
		})
	add("login-attempt", Notice, "submits a username and password",
		func(e event.Event) bool { return e.Kind == event.KindLogin })
}
