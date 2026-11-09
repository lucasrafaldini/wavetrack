package demo

import (
	"testing"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/testutil"
)

func TestSchedulesAreDeterministicAndPlausible(t *testing.T) {
	day := time.Date(2026, 11, 4, 0, 0, 0, 0, time.Local) // a Wednesday
	for _, m := range Office {
		a, b := schedule(m, day), schedule(m, day)
		if len(a) != len(b) {
			t.Fatalf("%s: schedule changed between calls", m.Name)
		}
		for i, w := range a {
			if !w.start.Equal(b[i].start) || !w.end.After(w.start) {
				t.Fatalf("%s: bad window %+v", m.Name, w)
			}
			if w.start.Hour() < 7 || w.end.Hour() > 21 {
				t.Fatalf("%s: outside office hours %+v", m.Name, w)
			}
		}
	}
}

func TestSeedFillsHistoryAndLeavesCurrentVisitsOpen(t *testing.T) {
	st := testutil.Store(t)
	now := time.Date(2026, 11, 5, 14, 20, 0, 0, time.Local)
	if err := Seed(st, now, 14); err != nil {
		t.Fatal(err)
	}
	people, _ := st.People(false)
	if len(people) != len(Office) {
		t.Fatalf("people = %d", len(people))
	}
	visits, _ := st.VisitsBetween(now.AddDate(0, 0, -14), now)
	if len(visits) < 50 {
		t.Fatalf("only %d visits in two weeks", len(visits))
	}
	open, _ := st.OpenVisits()
	if len(open) == 0 {
		t.Fatal("nobody is in the office at 14:20 on a Thursday")
	}
	for _, v := range open {
		if v.StartedAt.After(now) {
			t.Fatalf("visit starts in the future: %+v", v)
		}
	}
}

func TestClockShiftsOnlyOutsideOfficeHours(t *testing.T) {
	night := func() time.Time { return time.Date(2026, 11, 5, 2, 0, 0, 0, time.Local) }
	if got := Clock(night)(); got.Hour() != 14 || got.Minute() != 20 {
		t.Fatalf("night clock = %v", got)
	}
	day := func() time.Time { return time.Date(2026, 11, 5, 11, 0, 0, 0, time.Local) }
	if got := Clock(day)(); !got.Equal(day()) {
		t.Fatalf("daytime clock shifted to %v", got)
	}
}

func TestClockMovesWeekendsToFridayAfternoon(t *testing.T) {
	sunday := time.Date(2026, 11, 8, 2, 20, 0, 0, time.Local)
	got := Clock(func() time.Time { return sunday })()
	if got.Weekday() != time.Friday || got.Hour() != 14 || got.Minute() != 20 || got.Day() != 6 {
		t.Fatalf("Sunday became %v", got)
	}
	tuesday := time.Date(2026, 11, 10, 11, 0, 0, 0, time.Local)
	if got := Clock(func() time.Time { return tuesday })(); !got.Equal(tuesday) {
		t.Fatalf("office hours were shifted to %v", got)
	}
}
