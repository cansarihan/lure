package sensor

import (
	"bufio"
	"context"
	"io"
	"net"
	"strings"

	"github.com/cansarihan/lure/internal/event"
)

const telnetAttempts = 3

// Telnet shows a device-style login prompt and records each username and
// password a client tries. Every attempt is refused.
type Telnet struct {
	Hostname string
}

func (Telnet) Name() string { return "telnet" }

func (t Telnet) Serve(ctx context.Context, ln net.Listener, sink Sink) error {
	host := t.Hostname
	if host == "" {
		host = "gateway"
	}
	return acceptLoop(ctx, ln, func(conn net.Conn) {
		r := bufio.NewReaderSize(io.LimitReader(conn, maxRead), 512)
		_, _ = io.WriteString(conn, "\r\n"+host+" login: ")
		for attempt := 1; attempt <= telnetAttempts; attempt++ {
			if attempt > 1 {
				_, _ = io.WriteString(conn, host+" login: ")
			}
			user, err := readTelnetLine(r)
			if err != nil {
				return
			}
			e := base("telnet", conn, event.KindLogin)
			if attempt == 1 {
				e.Fields["text"] = user
			}
			_, _ = io.WriteString(conn, "Password: ")
			password, _ := readTelnetLine(r)
			e.Fields["user"] = user
			e.Fields["password"] = password
			e.Fields["attempt"] = itoa(attempt)
			e.Summary = "tried " + user + " / " + mask(password)
			sink(e)
			_, _ = io.WriteString(conn, "\r\nLogin incorrect\r\n\r\n")
		}
	})
}

// readTelnetLine reads one line, dropping telnet option negotiation (IAC
// sequences) and trailing whitespace.
func readTelnetLine(r *bufio.Reader) (string, error) {
	var sb strings.Builder
	for {
		b, err := r.ReadByte()
		if err != nil {
			if sb.Len() > 0 {
				return strings.TrimSpace(sb.String()), nil
			}
			return "", err
		}
		switch {
		case b == 0xFF: // IAC: skip the command and its option byte
			_, _ = r.ReadByte()
			_, _ = r.ReadByte()
		case b == '\n':
			return strings.TrimSpace(sb.String()), nil
		case b == 0 || b == '\r':
		default:
			if sb.Len() < 256 {
				sb.WriteByte(b)
			}
		}
	}
}

// mask hides most of a password in the one-line summary; the full value stays
// in the event fields for analysis.
func mask(p string) string {
	if len(p) <= 2 {
		return strings.Repeat("*", len(p))
	}
	return p[:1] + strings.Repeat("*", len(p)-2) + p[len(p)-1:]
}
