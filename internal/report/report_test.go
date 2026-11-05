package report

import (
	"testing"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/store"
)

func at(loc *time.Location, day, hour, minute int) time.Time {
	return time.Date(2026, 11, day, hour, minute, 0, 0, loc)
}

func ptr(t time.Time) *time.Time { return &t }

func TestTimelineClipsToTheDayAndMergesOverlaps(t *testing.T) {
	loc := time.UTC
	visits := []store.Visit{
		{PersonID: 1, StartedAt: at(loc, 4, 22, 0), EndedAt: ptr(at(loc, 5, 1, 0))}, // across midnight
		{PersonID: 1, StartedAt: at(loc, 5, 9, 0), EndedAt: ptr(at(loc, 5, 12, 0))},
		{PersonID: 1, StartedAt: at(loc, 5, 11, 30), EndedAt: ptr(at(loc, 5, 13, 0))}, // overlaps
		{PersonID: 2, StartedAt: at(loc, 5, 14, 0)},                                   // ongoing
	}
	now := at(loc, 5, 15, 0)
	tl := Timeline(visits, at(loc, 5, 0, 0), now)

	one := tl[1]
	if len(one) != 2 {
		t.Fatalf("person 1 intervals = %+v", one)
	}
	if !one[0].Start.Equal(at(loc, 5, 0, 0)) || !one[0].End.Equal(at(loc, 5, 1, 0)) {
		t.Errorf("midnight clip = %+v", one[0])
	}
	if !one[1].Start.Equal(at(loc, 5, 9, 0)) || !one[1].End.Equal(at(loc, 5, 13, 0)) {
		t.Errorf("merge = %+v", one[1])
	}
	two := tl[2]
	if len(two) != 1 || !two[0].Ongoing || !two[0].End.Equal(now) {
		t.Errorf("ongoing = %+v", two)
	}
}

func TestDailyTotals(t *testing.T) {
	loc := time.UTC
	visits := []store.Visit{
		{PersonID: 7, StartedAt: at(loc, 4, 9, 0), EndedAt: ptr(at(loc, 4, 17, 30))},
		{PersonID: 7, StartedAt: at(loc, 5, 8, 45), EndedAt: ptr(at(loc, 5, 12, 0))},
		{PersonID: 7, StartedAt: at(loc, 5, 13, 0), EndedAt: ptr(at(loc, 5, 18, 0))},
	}
	days := LastDays(at(loc, 5, 20, 0), 3, loc)
	totals := Daily(visits, days, at(loc, 5, 20, 0))[7]

	if len(totals) != 3 || totals[0].Seconds != 0 || totals[0].Date != "2026-11-03" {
		t.Fatalf("first day = %+v", totals)
	}
	if totals[1].Seconds != int64(8.5*3600) {
		t.Errorf("Nov 4 = %v", totals[1].Seconds)
	}
	if totals[2].Seconds != int64((3*60+15+5*60)*60) || !totals[2].First.Equal(at(loc, 5, 8, 45)) || !totals[2].Last.Equal(at(loc, 5, 18, 0)) {
		t.Errorf("Nov 5 = %+v", totals[2])
	}
}

func TestDaysFollowLocalMidnightAcrossDST(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no tz database")
	}
	// Clocks spring forward on 2026-03-08, a 23-hour day.
	days := LastDays(time.Date(2026, 3, 9, 12, 0, 0, 0, ny), 2, ny)
	if got := days[1].Sub(days[0]); got != 23*time.Hour {
		t.Fatalf("DST day length = %v", got)
	}
	if days[1].Hour() != 0 {
		t.Fatalf("day doesn't start at local midnight: %v", days[1])
	}
}
