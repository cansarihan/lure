package sensor

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cansarihan/lure/internal/event"
)

type collector struct {
	mu     sync.Mutex
	events []event.Event
}

func (c *collector) sink(e event.Event) { c.mu.Lock(); c.events = append(c.events, e); c.mu.Unlock() }

func (c *collector) wait(t *testing.T, n int) []event.Event {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		if len(c.events) >= n {
			out := append([]event.Event(nil), c.events...)
			c.mu.Unlock()
			return out
		}
		c.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("waited for %d events", n)
	return nil
}

func serve(t *testing.T, s Sensor) (string, *collector) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := &collector{}
	done := make(chan struct{})
	go func() { _ = s.Serve(ctx, ln, c.sink); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	return ln.Addr().String(), c
}

func TestSSHRecordsClient(t *testing.T) {
	addr, c := serve(t, SSH{})
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	banner, _ := bufio.NewReader(conn).ReadString('\n')
	if !strings.HasPrefix(banner, "SSH-2.0-") {
		t.Errorf("banner = %q", banner)
	}
	_, _ = io.WriteString(conn, "SSH-2.0-libssh_0.9.6\r\n")
	_ = conn.Close()
	e := c.wait(t, 1)[0]
	if e.Fields["client"] != "SSH-2.0-libssh_0.9.6" || e.Remote != "127.0.0.1" || e.Port == 0 {
		t.Errorf("event = %+v", e)
	}
}

func TestRawRecordsBytes(t *testing.T) {
	addr, c := serve(t, Raw{Label: "redis"})
	conn, _ := net.Dial("tcp", addr)
	_, _ = io.WriteString(conn, "*1\r\n$4\r\nINFO\r\n")
	_ = conn.Close()
	e := c.wait(t, 1)[0]
	if e.Kind != event.KindPayload || !strings.HasPrefix(e.Fields["hex"], "2a310d0a") || e.Fields["dressed_as"] != "redis" {
		t.Errorf("event = %+v", e)
	}
}

func TestTelnetRecordsAttempts(t *testing.T) {
	addr, c := serve(t, Telnet{})
	conn, _ := net.Dial("tcp", addr)
	// a client opens with option negotiation before typing
	_, _ = conn.Write([]byte{0xFF, 0xFD, 0x01})
	_, _ = io.WriteString(conn, "root\r\nroot\r\nadmin\r\nadmin\r\n")
	events := c.wait(t, 2)
	_ = conn.Close()
	if events[0].Fields["user"] != "root" || events[0].Fields["password"] != "root" {
		t.Errorf("first attempt = %+v", events[0].Fields)
	}
	if events[1].Fields["user"] != "admin" || events[1].Fields["attempt"] != "2" {
		t.Errorf("second attempt = %+v", events[1].Fields)
	}
	if strings.Contains(events[0].Summary, "root / root") {
		t.Error("summary should mask the password")
	}
}

func TestHTTPRecordsRequestAndLogin(t *testing.T) {
	addr, c := serve(t, HTTP{})
	res, err := http.Get("http://" + addr + "/.env?x=1")
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusNotFound || res.Header.Get("Server") == "" {
		t.Errorf("probe answered %d server=%q", res.StatusCode, res.Header.Get("Server"))
	}
	res, err = http.PostForm("http://"+addr+"/login", url.Values{"username": {"admin"}, "password": {"admin"}})
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("login answered %d", res.StatusCode)
	}
	events := c.wait(t, 3)
	if events[0].Fields["path"] != "/.env" || events[0].Fields["query"] != "x=1" {
		t.Errorf("request = %+v", events[0].Fields)
	}
	if events[2].Kind != event.KindLogin || events[2].Fields["user"] != "admin" {
		t.Errorf("login = %+v", events[2])
	}
}
