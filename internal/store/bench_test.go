package store_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/store"
	"github.com/lucasrafaldini/wavetrack/internal/testutil"
)

// An office of 100 people with a year of history: the default retention.
const benchPeople = 100

func benchStore(b *testing.B, days int) (*store.Store, []int64, time.Time) {
	b.Helper()
	st := testutil.Store(b)
	now := testutil.Day(14, 0)
	ids, _ := testutil.Office(b, st, benchPeople, days, now)
	return st, ids, now
}

// The query behind every dashboard refresh: today's visits.
func BenchmarkVisitsToday(b *testing.B) {
	for _, days := range []int{30, 365} {
		b.Run(fmt.Sprintf("history=%dd", days), func(b *testing.B) {
			st, _, now := benchStore(b, days)
			from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			b.ReportAllocs()
			for b.Loop() {
				if _, err := st.VisitsBetween(from, from.AddDate(0, 0, 1)); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// The History page's widest window.
func BenchmarkVisitsMonth(b *testing.B) {
	st, _, now := benchStore(b, 365)
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -30)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := st.VisitsBetween(from, from.AddDate(0, 0, 31)); err != nil {
			b.Fatal(err)
		}
	}
}

// The tracker's steady write: each present person's visit is extended once a minute.
func BenchmarkTouchVisit(b *testing.B) {
	st, ids, now := benchStore(b, 1)
	id, err := st.StartVisit(ids[0], now)
	if err != nil {
		b.Fatal(err)
	}
	at := now
	b.ReportAllocs()
	for b.Loop() {
		at = at.Add(time.Minute)
		if err := st.TouchVisit(id, at); err != nil {
			b.Fatal(err)
		}
	}
}
