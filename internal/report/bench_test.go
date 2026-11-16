package report

import (
	"testing"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/store"
)

// Two visits per person per weekday, like testutil.Office, built in memory.
func benchVisits(people, days int, now time.Time) []store.Visit {
	var visits []store.Visit
	today := StartOfDay(now, time.Local)
	for d := days - 1; d >= 0; d-- {
		day := today.AddDate(0, 0, -d)
		if wd := day.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		for p := range people {
			jitter := time.Duration(p%37) * time.Minute
			lunch := day.Add(12*time.Hour + jitter)
			leave := day.Add(17*time.Hour + 30*time.Minute + jitter)
			visits = append(visits,
				store.Visit{PersonID: int64(p + 1), StartedAt: day.Add(8*time.Hour + 30*time.Minute + jitter), LastSeen: lunch, EndedAt: &lunch},
				store.Visit{PersonID: int64(p + 1), StartedAt: lunch.Add(time.Hour), LastSeen: leave, EndedAt: &leave})
		}
	}
	return visits
}

var benchNow = time.Date(2026, 11, 5, 14, 0, 0, 0, time.Local)

// The History page at its widest: 100 people over 31 days.
func BenchmarkDailyMonth(b *testing.B) {
	visits := benchVisits(100, 31, benchNow)
	days := LastDays(benchNow, 31, time.Local)
	b.ReportAllocs()
	for b.Loop() {
		Daily(visits, days, benchNow)
	}
}

// The Timeline page: one day, 100 people.
func BenchmarkTimelineDay(b *testing.B) {
	visits := benchVisits(100, 1, benchNow)
	day := StartOfDay(benchNow, time.Local)
	b.ReportAllocs()
	for b.Loop() {
		Timeline(visits, day, benchNow)
	}
}
