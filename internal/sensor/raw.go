package sensor

import (
	"context"
	"encoding/hex"
	"io"
	"net"
	"strings"
	"time"
	"unicode"

	"github.com/cansarihan/lure/internal/event"
)

// Raw listens on a port that pretends to be an unknown service. It reads the
// first bytes a client sends — the opening of whatever protocol it expected —
// and records them.
type Raw struct {
	Label string // service the port is dressed as, e.g. "redis"
}

func (Raw) Name() string { return "raw" }

func (r Raw) Serve(ctx context.Context, ln net.Listener, sink Sink) error {
	return acceptLoop(ctx, ln, func(conn net.Conn) {
		e := base("raw", conn, event.KindConnect)
		if r.Label != "" {
			e.Fields["dressed_as"] = r.Label
		}
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		buf := make([]byte, 1024)
		n, _ := io.ReadAtLeast(conn, buf, 1)
		if n == 0 {
			e.Summary = "connected and sent nothing"
			sink(e)
			return
		}
		e.Kind = event.KindPayload
		head := buf[:min(n, 32)]
		e.Fields["hex"] = hex.EncodeToString(head)
		e.Fields["text"] = printable(buf[:n])
		e.Fields["bytes"] = itoa(n)
		e.Summary = "sent " + itoa(n) + " bytes"
		sink(e)
	})
}

// printable keeps the text a human can read and turns the rest into dots.
func printable(b []byte) string {
	var sb strings.Builder
	for _, c := range string(b) {
		switch {
		case c == '\r' || c == '\n' || c == '\t':
			sb.WriteRune(c)
		case unicode.IsPrint(c) && c < unicode.MaxASCII:
			sb.WriteRune(c)
		default:
			sb.WriteByte('.')
		}
	}
	return sb.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
