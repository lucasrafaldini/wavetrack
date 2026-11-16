package presence

import (
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/capture"
	"github.com/lucasrafaldini/wavetrack/internal/testutil"
)

func benchTracker(b *testing.B, people int) (*Tracker, []net.HardwareAddr, time.Time) {
	b.Helper()
	st := testutil.Store(b)
	now := testutil.Day(9, 0)
	_, macs := testutil.Office(b, st, people, 0, now)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tr := New(st, testutil.Identifier{}, Options{AwayAfter: 20 * time.Minute, MinSignal: -85, Logger: logger, Clock: func() time.Time { return now }})
	if err := tr.Start(now); err != nil {
		b.Fatal(err)
	}
	hw := make([]net.HardwareAddr, len(macs))
	for i, m := range macs {
		hw[i] = testutil.MAC(b, m)
	}
	return tr, hw, now
}

// Registered phones seen over and over during the day: 100 people, one sighting
// per simulated second, round robin. Includes the once-a-minute SQLite writes.
func BenchmarkObserveRegistered(b *testing.B) {
	tr, macs, now := benchTracker(b, 100)
	at := now
	i := 0
	b.ReportAllocs()
	for b.Loop() {
		at = at.Add(time.Second)
		tr.Observe(capture.Sighting{MAC: macs[i%len(macs)], At: at, Signal: -60})
		i++
	}
}

// Strangers' devices going past: kept in memory only, for the Nearby page.
func BenchmarkObserveNearby(b *testing.B) {
	tr, _, now := benchTracker(b, 1)
	macs := make([]net.HardwareAddr, 500)
	for i := range macs {
		macs[i] = net.HardwareAddr{0x3c, 0x22, 0xfb, byte(i >> 8), byte(i), 0x01}
	}
	at := now
	i := 0
	b.ReportAllocs()
	for b.Loop() {
		at = at.Add(100 * time.Millisecond)
		tr.Observe(capture.Sighting{MAC: macs[i%len(macs)], At: at, Signal: -70})
		i++
	}
}

// What the dashboard reads on every refresh.
func BenchmarkSnapshot(b *testing.B) {
	tr, macs, now := benchTracker(b, 100)
	for i, mac := range macs {
		tr.Observe(capture.Sighting{MAC: mac, At: now.Add(time.Duration(i) * time.Second), Signal: -60})
	}
	b.ReportAllocs()
	for b.Loop() {
		tr.Snapshot(now.Add(5 * time.Minute))
	}
}
