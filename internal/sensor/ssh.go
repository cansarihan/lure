package sensor

import (
	"bufio"
	"context"
	"io"
	"net"
	"strings"

	"github.com/cansarihan/lure/internal/event"
)

// SSH sends an OpenSSH version banner and records the identification line the
// client sends back, which names its software. It never negotiates keys, so no
// session can be opened.
type SSH struct {
	Banner string
}

func (SSH) Name() string { return "ssh" }

func (s SSH) Serve(ctx context.Context, ln net.Listener, sink Sink) error {
	banner := s.Banner
	if banner == "" {
		banner = "SSH-2.0-OpenSSH_8.9p1 Ubuntu-3ubuntu0.6"
	}
	return acceptLoop(ctx, ln, func(conn net.Conn) {
		e := base("ssh", conn, event.KindConnect)
		if _, err := io.WriteString(conn, banner+"\r\n"); err != nil {
			return
		}
		line, _ := bufio.NewReaderSize(io.LimitReader(conn, maxRead), 512).ReadString('\n')
		client := strings.TrimSpace(line)
		e.Fields["client"] = client
		e.Summary = "connected"
		if client != "" {
			e.Summary = "client " + client
		}
		sink(e)
	})
}
