package sensor

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
)

// Spec describes one listener: which decoy, where, and (for raw ports) what it
// is dressed as.
type Spec struct {
	Sensor string `json:"sensor"`
	Addr   string `json:"addr"`
	Label  string `json:"label,omitempty"`
}

// Bound is a listener that started, with the address it actually holds.
type Bound struct {
	Spec Spec   `json:"spec"`
	Addr string `json:"addr"`
}

// ParseSpec reads "sensor=addr" or "raw:label=addr", e.g. "http=:8080" or
// "raw:redis=:6379".
func ParseSpec(s string) (Spec, error) {
	name, addr, ok := strings.Cut(s, "=")
	if !ok || addr == "" {
		return Spec{}, fmt.Errorf("listener %q: want sensor=addr", s)
	}
	name, label, _ := strings.Cut(name, ":")
	if _, err := build(Spec{Sensor: name}); err != nil {
		return Spec{}, err
	}
	return Spec{Sensor: name, Addr: addr, Label: label}, nil
}

func build(s Spec) (Sensor, error) {
	switch s.Sensor {
	case "http":
		return HTTP{}, nil
	case "ssh":
		return SSH{}, nil
	case "telnet":
		return Telnet{}, nil
	case "raw":
		return Raw{Label: s.Label}, nil
	default:
		return nil, fmt.Errorf("unknown sensor %q (want http, ssh, telnet or raw)", s.Sensor)
	}
}

// Start opens every listener and serves it until ctx ends. If any listener
// fails to bind, the ones already opened are closed and the error returned.
// wait blocks until every sensor has stopped.
func Start(ctx context.Context, specs []Spec, sink Sink) (bound []Bound, wait func(), err error) {
	type opened struct {
		sensor Sensor
		ln     net.Listener
	}
	var all []opened
	for _, s := range specs {
		sensor, err := build(s)
		if err == nil {
			var ln net.Listener
			ln, err = net.Listen("tcp", s.Addr)
			if err == nil {
				all = append(all, opened{sensor, ln})
				bound = append(bound, Bound{Spec: s, Addr: ln.Addr().String()})
				continue
			}
		}
		for _, o := range all {
			_ = o.ln.Close()
		}
		return nil, nil, fmt.Errorf("%s on %s: %w", s.Sensor, s.Addr, err)
	}
	var wg sync.WaitGroup
	for _, o := range all {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = o.sensor.Serve(ctx, o.ln, sink)
		}()
	}
	return bound, wg.Wait, nil
}
