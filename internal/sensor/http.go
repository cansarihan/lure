package sensor

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cansarihan/lure/internal/event"
)

const loginPage = `<!doctype html><html><head><title>Router Administration</title></head>
<body style="font-family:sans-serif;background:#f2f2f2"><form method="post" action="/login"
style="width:280px;margin:120px auto;padding:24px;background:#fff;border:1px solid #ccc">
<h3>Administration</h3><p><input name="username" placeholder="Username"></p>
<p><input name="password" type="password" placeholder="Password"></p><p><button>Sign in</button></p></form></body></html>`

// HTTP answers every request with either a stock login page or a 404, and
// records the request: method, path, query, user agent and the first part of
// the body. A posted login form is also recorded as a login attempt.
type HTTP struct {
	Server string // value of the Server header
}

func (HTTP) Name() string { return "http" }

func (h HTTP) Serve(ctx context.Context, ln net.Listener, sink Sink) error {
	serverHeader := h.Server
	if serverHeader == "" {
		serverHeader = "nginx/1.24.0"
	}
	srv := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       connTimeout,
		WriteTimeout:      connTimeout,
		MaxHeaderBytes:    16 << 10,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Server", serverHeader)
			body, _ := io.ReadAll(io.LimitReader(r.Body, maxRead))
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			_, local := split(ln.Addr())

			e := event.Event{Sensor: "http", Port: local, Remote: host, Kind: event.KindRequest, Fields: map[string]string{
				"method": r.Method,
				"path":   r.URL.Path,
				"query":  r.URL.RawQuery,
				"ua":     r.UserAgent(),
				"body":   string(body),
				"host":   r.Host,
			}}
			e.Summary = r.Method + " " + r.URL.RequestURI()
			sink(e)

			if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "login") {
				if form, err := url.ParseQuery(string(body)); err == nil {
					login := e
					login.Kind = event.KindLogin
					login.Fields = map[string]string{"user": form.Get("username"), "password": form.Get("password"), "path": r.URL.Path, "ua": r.UserAgent()}
					login.Summary = "tried " + form.Get("username") + " / " + mask(form.Get("password"))
					sink(login)
				}
				http.Error(w, "Invalid username or password", http.StatusUnauthorized)
				return
			}
			switch r.URL.Path {
			case "/", "/login", "/admin", "/index.html":
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = io.WriteString(w, loginPage)
			default:
				http.NotFound(w, r)
			}
		}),
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
