package classify

import "github.com/cansarihan/lure/internal/event"

func init() {
	add("automated-client", Info, "the user agent belongs to a scanner or a scripting library, not a browser",
		func(e event.Event) bool {
			return containsAny(field(e, "ua"), "zgrab", "masscan", "nmap", "nuclei", "go-http-client",
				"python-requests", "curl/", "wget/", "libwww", "censys", "shodan")
		})
	add("missing-user-agent", Info, "an HTTP request with no user agent at all",
		func(e event.Event) bool { return e.Kind == event.KindRequest && field(e, "ua") == "" })
	add("config-probe", Notice, "asks for a file that holds secrets or source, such as .env or .git/config",
		func(e event.Event) bool {
			return containsAny(field(e, "path"), "/.env", "/.git/", "/config.json", "/.aws/", "/wp-config", "/.ssh/", "/backup")
		})
	add("admin-probe", Notice, "looks for a well-known admin or login page",
		func(e event.Event) bool {
			return containsAny(field(e, "path"), "/wp-login", "/wp-admin", "/phpmyadmin", "/admin", "/manager/html", "/login")
		})
	add("device-probe", Notice, "targets paths used by routers, cameras and other embedded devices",
		func(e event.Event) bool {
			return containsAny(field(e, "path"), "/cgi-bin/", "/boaform", "/hnap1", "/goform", "/setup.cgi")
		})
	add("path-traversal", Alert, "climbs out of the web root with ../ sequences",
		func(e event.Event) bool {
			p := field(e, "path") + "?" + field(e, "query")
			return containsAny(p, "../", "..%2f", "%2e%2e")
		})
	add("lookup-injection", Alert, "carries a ${jndi: lookup string meant for vulnerable loggers",
		func(e event.Event) bool {
			return containsAny(field(e, "path")+field(e, "query")+field(e, "ua")+field(e, "body"), "${jndi:")
		})
	add("sql-injection-probe", Alert, "carries SQL meant to break out of a query",
		func(e event.Event) bool {
			return containsAny(field(e, "query")+field(e, "body"), "' or '1'='1", "union select", "' or 1=1", "sleep(")
		})
	add("command-injection-probe", Alert, "carries shell syntax meant to run a command on the server",
		func(e event.Event) bool {
			return containsAny(field(e, "query")+field(e, "body"), ";wget", ";curl", "|sh", "$(", "`id`", ";id")
		})
}
