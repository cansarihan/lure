package sensor

import (
	"context"
	"net"
	"testing"

	"github.com/cansarihan/lure/internal/event"
)

func TestParseSpec(t *testing.T) {
	s, err := ParseSpec("raw:redis=:6379")
	if err != nil || s.Sensor != "raw" || s.Label != "redis" || s.Addr != ":6379" {
		t.Fatalf("parsed %+v, %v", s, err)
	}
	for _, bad := range []string{"http", "ftp=:21", "ssh="} {
		if _, err := ParseSpec(bad); err == nil {
			t.Errorf("%q should not parse", bad)
		}
	}
}

func TestStartAndStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	bound, wait, err := Start(ctx, []Spec{{Sensor: "ssh", Addr: "127.0.0.1:0"}, {Sensor: "http", Addr: "127.0.0.1:0"}}, func(event.Event) {})
	if err != nil {
		t.Fatal(err)
	}
	if len(bound) != 2 || bound[0].Addr == "127.0.0.1:0" {
		t.Fatalf("bound = %+v", bound)
	}
	cancel()
	wait()
	if _, err := net.Dial("tcp", bound[0].Addr); err == nil {
		t.Error("listener still open after stop")
	}
}

func TestStartRollsBackOnBindFailure(t *testing.T) {
	busy, _ := net.Listen("tcp", "127.0.0.1:0")
	defer func() { _ = busy.Close() }()
	_, _, err := Start(context.Background(), []Spec{{Sensor: "ssh", Addr: "127.0.0.1:0"}, {Sensor: "http", Addr: busy.Addr().String()}}, func(event.Event) {})
	if err == nil {
		t.Fatal("binding a busy port should fail")
	}
}
