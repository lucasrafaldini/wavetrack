package server

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/capture"
	"github.com/lucasrafaldini/wavetrack/internal/presence"
	"github.com/lucasrafaldini/wavetrack/internal/testutil"
	"github.com/lucasrafaldini/wavetrack/web"
)

// benchServer is a signed-in dashboard over an office of 100 people with a year of
// history, everyone present right now. Requests go straight to the handler, so the
// numbers are WaveTrack's own cost without the network.
func benchServer(b *testing.B) (http.Handler, *http.Cookie) {
	b.Helper()
	st := testutil.Store(b)
	now := testutil.Day(14, 0)
	_, macs := testutil.Office(b, st, 100, 365, now)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clock := func() time.Time { return now }
	tracker := presence.New(st, testutil.Identifier{}, presence.Options{AwayAfter: 20 * time.Minute, MinSignal: -85, Logger: logger, Clock: clock})
	if err := tracker.Start(now); err != nil {
		b.Fatal(err)
	}
	for _, m := range macs {
		tracker.Observe(capture.Sighting{MAC: testutil.MAC(b, m), At: now.Add(-time.Minute), Signal: -60})
	}
	s := New(Options{
		Store: st, Tracker: tracker, Ident: testutil.Identifier{}, Capture: capture.Description{Kind: "demo", Mode: "simulated"},
		Assets: web.FS, Clock: clock, AwayAfter: 20 * time.Minute, Logger: logger, Version: "bench",
		LookupMAC:  func(string) (net.HardwareAddr, error) { return nil, nil },
		InviteBase: func(*http.Request) string { return "http://127.0.0.1:8080" },
	})
	token := NewToken("")
	if err := st.CreateSession(Hash(token), now, now.Add(time.Hour)); err != nil {
		b.Fatal(err)
	}
	return s.Handler(), &http.Cookie{Name: sessionCookie, Value: token}
}

func benchGet(b *testing.B, h http.Handler, cookie *http.Cookie, path string) {
	b.Helper()
	b.ReportAllocs()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("%s: %d %s", path, rec.Code, rec.Body)
		}
	}
}

func BenchmarkAPI(b *testing.B) {
	h, cookie := benchServer(b)
	for _, path := range []string{"/api/now", "/api/timeline", "/api/history?days=7", "/api/history?days=31", "/api/export.csv?days=366"} {
		b.Run(path, func(b *testing.B) { benchGet(b, h, cookie, path) })
	}
}

// Many dashboards refreshing at once, e.g. a wall display plus every manager's laptop.
func BenchmarkAPINowParallel(b *testing.B) {
	h, cookie := benchServer(b)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/api/now", nil)
			req.AddCookie(cookie)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				b.Errorf("%d", rec.Code)
				return
			}
		}
	})
}
