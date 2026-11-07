package presence

import (
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/capture"
	"github.com/lucasrafaldini/wavetrack/internal/store"
	"github.com/lucasrafaldini/wavetrack/internal/testutil"
)

const phone = "a4:83:e7:21:5c:10"

func setup(t *testing.T) (*Tracker, *store.Store, int64) {
	t.Helper()
	st := testutil.Store(t)
	p, err := st.CreatePerson("Ana", "Design", nil, testutil.Day(8, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AddDevice(store.Device{MAC: phone, PersonID: p.ID, AddedAt: testutil.Day(8, 0)}); err != nil {
		t.Fatal(err)
	}
	tr := New(st, testutil.Identifier{}, Options{AwayAfter: 20 * time.Minute, MinSignal: -80, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err := tr.Start(testutil.Day(8, 0)); err != nil {
		t.Fatal(err)
	}
	return tr, st, p.ID
}

func see(t *testing.T, tr *Tracker, mac string, at time.Time, signal int) {
	t.Helper()
	tr.Observe(capture.Sighting{MAC: testutil.MAC(t, mac), At: at, Signal: signal})
}

func TestArrivalStartsAVisit(t *testing.T) {
	tr, st, id := setup(t)
	see(t, tr, phone, testutil.Day(9, 0), 0)

	snap := tr.Snapshot(testutil.Day(9, 1))
	if len(snap.Present) != 1 || snap.Present[0].PersonID != id || !snap.Present[0].Since.Equal(testutil.Day(9, 0)) {
		t.Fatalf("present = %+v", snap.Present)
	}
	open, _ := st.OpenVisits()
	if len(open) != 1 {
		t.Fatalf("open visits = %d, want 1", len(open))
	}
}

// Regression: 2.x only reported a device the first time it was seen, so everyone was
// marked as gone after the timeout even while they were still in the office.
func TestRepeatedSightingsKeepSomeonePresentAllDay(t *testing.T) {
	tr, st, _ := setup(t)
	for at := testutil.Day(9, 0); at.Before(testutil.Day(17, 0)); at = at.Add(5 * time.Minute) {
		see(t, tr, phone, at, 0)
		tr.Sweep(at.Add(time.Minute))
	}
	if n := len(tr.Snapshot(testutil.Day(17, 0)).Present); n != 1 {
		t.Fatalf("present = %d after a full day of sightings, want 1", n)
	}
	visits, _ := st.VisitsBetween(testutil.Day(0, 0), testutil.Day(23, 59))
	if len(visits) != 1 {
		t.Fatalf("visits = %d, want a single continuous visit", len(visits))
	}
}

func TestDepartureIsRecordedAtTheLastSighting(t *testing.T) {
	tr, st, _ := setup(t)
	see(t, tr, phone, testutil.Day(9, 0), 0)
	see(t, tr, phone, testutil.Day(12, 30), 0)

	tr.Sweep(testutil.Day(12, 45)) // within AwayAfter: still here
	if len(tr.Snapshot(testutil.Day(12, 45)).Present) != 1 {
		t.Fatal("left too early")
	}
	tr.Sweep(testutil.Day(12, 51))
	if len(tr.Snapshot(testutil.Day(12, 51)).Present) != 0 {
		t.Fatal("still present after AwayAfter")
	}
	visits, _ := st.VisitsBetween(testutil.Day(0, 0), testutil.Day(23, 59))
	if len(visits) != 1 || visits[0].EndedAt == nil || !visits[0].EndedAt.Equal(testutil.Day(12, 30)) {
		t.Fatalf("visit = %+v, want it to end at 12:30", visits)
	}

	see(t, tr, phone, testutil.Day(13, 40), 0)
	visits, _ = st.VisitsBetween(testutil.Day(0, 0), testutil.Day(23, 59))
	if len(visits) != 2 {
		t.Fatalf("visits = %d, want a second visit after lunch", len(visits))
	}
}

func TestWeakSignalIsIgnored(t *testing.T) {
	tr, _, _ := setup(t)
	see(t, tr, phone, testutil.Day(9, 0), -92)
	if len(tr.Snapshot(testutil.Day(9, 0)).Present) != 0 {
		t.Fatal("a sighting below MinSignal counted")
	}
	see(t, tr, phone, testutil.Day(9, 1), -60)
	if len(tr.Snapshot(testutil.Day(9, 1)).Present) != 1 {
		t.Fatal("a strong sighting didn't count")
	}
}

func TestUnknownDevicesStayInMemory(t *testing.T) {
	tr, st, _ := setup(t)
	see(t, tr, "da:a1:19:6b:3f:f6", testutil.Day(9, 0), 0)
	see(t, tr, "8c:f5:a3:51:c2:e5", testutil.Day(9, 2), 0)

	nearby := tr.Snapshot(testutil.Day(9, 5)).Nearby
	if len(nearby) != 2 || nearby[0].MAC != "8c:f5:a3:51:c2:e5" || !nearby[1].Private {
		t.Fatalf("nearby = %+v", nearby)
	}
	devices, _ := st.Devices()
	if len(devices) != 1 {
		t.Fatalf("unknown devices were written to the database: %+v", devices)
	}
	if len(tr.Snapshot(testutil.Day(9, 40)).Nearby) != 0 {
		t.Fatal("stale nearby devices should drop out of the snapshot")
	}
}

func TestAssigningANearbyDeviceMakesItCount(t *testing.T) {
	tr, st, id := setup(t)
	laptop := "f0:18:98:4a:02:11"
	see(t, tr, laptop, testutil.Day(9, 0), 0)
	if err := st.AddDevice(store.Device{MAC: laptop, PersonID: id, AddedAt: testutil.Day(9, 1)}); err != nil {
		t.Fatal(err)
	}
	if err := tr.ReloadDevices(); err != nil {
		t.Fatal(err)
	}
	if n := len(tr.Snapshot(testutil.Day(9, 1)).Nearby); n != 0 {
		t.Fatalf("assigned device still listed as nearby (%d)", n)
	}
	see(t, tr, laptop, testutil.Day(9, 2), 0)
	if len(tr.Snapshot(testutil.Day(9, 2)).Present) != 1 {
		t.Fatal("assigned device didn't mark its owner present")
	}
}

func TestStartResumesRecentVisitsAndClosesStaleOnes(t *testing.T) {
	st := testutil.Store(t)
	ana, _ := st.CreatePerson("Ana", "", nil, testutil.Day(8, 0))
	bruno, _ := st.CreatePerson("Bruno", "", nil, testutil.Day(8, 0))
	recent, _ := st.StartVisit(ana.ID, testutil.Day(9, 0))
	_ = st.TouchVisit(recent, testutil.Day(11, 55))
	stale, _ := st.StartVisit(bruno.ID, testutil.Day(9, 0))
	_ = st.TouchVisit(stale, testutil.Day(10, 0))

	tr := New(st, testutil.Identifier{}, Options{AwayAfter: 20 * time.Minute, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err := tr.Start(testutil.Day(12, 0)); err != nil {
		t.Fatal(err)
	}
	present := tr.Snapshot(testutil.Day(12, 0)).Present
	if len(present) != 1 || present[0].PersonID != ana.ID {
		t.Fatalf("present after restart = %+v, want only Ana", present)
	}
	open, _ := st.OpenVisits()
	if len(open) != 1 || open[0].ID != recent {
		t.Fatalf("open visits = %+v", open)
	}
	visits, _ := st.VisitsBetween(testutil.Day(0, 0), testutil.Day(23, 0))
	for _, v := range visits {
		if v.ID == stale && (v.EndedAt == nil || !v.EndedAt.Equal(testutil.Day(10, 0))) {
			t.Fatalf("stale visit should end at its last sighting, got %+v", v)
		}
	}
}

// Run with -race: sightings, sweeps and snapshots happen on different goroutines.
func TestConcurrentUse(t *testing.T) {
	tr, _, _ := setup(t)
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				at := testutil.Day(9, 0).Add(time.Duration(i) * time.Second)
				switch g {
				case 0:
					see(t, tr, phone, at, 0)
				case 1:
					see(t, tr, "8c:f5:a3:51:c2:e5", at, 0)
				case 2:
					tr.Sweep(at)
				default:
					tr.Snapshot(at)
				}
			}
		}(g)
	}
	wg.Wait()
}
