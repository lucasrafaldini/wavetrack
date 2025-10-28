package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/models"
)

// helper to create a temp storage and cleanup
func newTempStorage(t *testing.T) *Storage {
	t.Helper()
	dir := t.TempDir()

	// Ensure the data dir exists
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	st, err := NewStorage(dir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	t.Cleanup(func() {
		st.Close()
		// remove db file explicitly to avoid leftovers on some systems
		_ = os.Remove(filepath.Join(dir, "wavetrack.db"))
	})

	return st
}

func TestGetFirstArrivalToday_NoEvents(t *testing.T) {
	st := newTempStorage(t)

	ts, err := st.GetFirstArrivalToday("AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts != nil {
		t.Fatalf("expected nil, got %v", ts)
	}
}

func TestGetFirstArrivalToday_Earliest(t *testing.T) {
	st := newTempStorage(t)

	mac := "AA:BB:CC:DD:EE:01"
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Insert two arrivals today, at 09:00 and 08:30
	e1 := &models.PresenceEvent{Type: "arrival", MACAddress: mac, Timestamp: today.Add(9 * time.Hour)}
	e2 := &models.PresenceEvent{Type: "arrival", MACAddress: mac, Timestamp: today.Add(8*time.Hour + 30*time.Minute)}

	if err := st.SaveEvent(e1); err != nil {
		t.Fatalf("save e1: %v", err)
	}
	if err := st.SaveEvent(e2); err != nil {
		t.Fatalf("save e2: %v", err)
	}

	ts, err := st.GetFirstArrivalToday(mac)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts == nil {
		t.Fatalf("expected non-nil timestamp")
	}
	want := today.Add(8*time.Hour + 30*time.Minute)
	if !ts.Equal(want) {
		t.Fatalf("expected %v, got %v", want, *ts)
	}
}

func TestGetOnlineDurationToday_SessionsAndOngoing(t *testing.T) {
	st := newTempStorage(t)

	mac := "AA:BB:CC:DD:EE:02"
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Sessions: [08:00, 10:15] and [11:00, ongoing until 12:30]
	t0 := today.Add(8 * time.Hour)
	t1 := today.Add(10*time.Hour + 15*time.Minute)
	t2 := today.Add(11 * time.Hour)
	t3 := today.Add(12*time.Hour + 30*time.Minute)

	events := []*models.PresenceEvent{
		{Type: "arrival", MACAddress: mac, Timestamp: t0},
		{Type: "departure", MACAddress: mac, Timestamp: t1},
		{Type: "arrival", MACAddress: mac, Timestamp: t2},
	}

	for _, e := range events {
		if err := st.SaveEvent(e); err != nil {
			t.Fatalf("save event %v: %v", e, err)
		}
	}

	dur, err := st.GetOnlineDurationToday(mac, t3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := (t1.Sub(t0)) + (t3.Sub(t2))
	if dur != want {
		t.Fatalf("expected %v, got %v", want, dur)
	}
}
