// Package sensor holds the decoy services. Each one answers just enough of its
// protocol to make a client show what it wanted — a path, a password, its first
// bytes — records that, and hangs up. Nothing a client sends is ever executed,
// and every connection is bounded in time and in bytes read.
package sensor

import (
	"context"
	"errors"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/cansarihan/lure/internal/event"
)

const (
	// connTimeout bounds how long any client may hold a connection.
	connTimeout = 20 * time.Second
	// maxRead bounds how much of a client's input a sensor will read.
	maxRead = 4096
)

// Sink receives what a sensor observed.
type Sink func(event.Event)

// Sensor serves one decoy protocol on a listener until ctx ends.
type Sensor interface {
	Name() string
	Serve(ctx context.Context, ln net.Listener, sink Sink) error
}

// acceptLoop runs handle for every connection until the listener closes,
// waiting for in-flight connections before it returns.
func acceptLoop(ctx context.Context, ln net.Listener, handle func(net.Conn)) error {
	go func() { <-ctx.Done(); _ = ln.Close() }()
	var wg sync.WaitGroup
	defer wg.Wait()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { _ = conn.Close() }()
			_ = conn.SetDeadline(time.Now().Add(connTimeout))
			handle(conn)
		}()
	}
}

// base fills the fields every event shares.
func base(sensor string, conn net.Conn, kind event.Kind) event.Event {
	host, port := split(conn.RemoteAddr())
	_, local := split(conn.LocalAddr())
	_ = port
	return event.Event{Sensor: sensor, Port: local, Remote: host, Kind: kind, Fields: map[string]string{}}
}

func split(addr net.Addr) (string, int) {
	host, portText, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String(), 0
	}
	port, _ := strconv.Atoi(portText)
	return host, port
}
