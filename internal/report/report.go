// Package report derives timelines and daily totals from visits.
package report

import (
	"sort"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/store"
)

type Interval struct {
	Start   time.Time `json:"start"`
	End     time.Time `json:"end"`
	Ongoing bool      `json:"ongoing"`
}

type Day struct {
	Date    string     `json:"date"` // YYYY-MM-DD in the report's location
	Seconds int64      `json:"seconds"`
	First   *time.Time `json:"first,omitempty"`
	Last    *time.Time `json:"last,omitempty"`
}

// StartOfDay returns local midnight. Using time.Date keeps DST days at 23 or 25 hours.
func StartOfDay(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// end of a visit: open visits run until now.
func visitEnd(v store.Visit, now time.Time) (time.Time, bool) {
	if v.EndedAt != nil {
		return *v.EndedAt, false
	}
	return now, true
}

// Timeline returns each person's intervals on the day starting at dayStart, clipped to the
// day and merged where visits overlap.
func Timeline(visits []store.Visit, dayStart time.Time, now time.Time) map[int64][]Interval {
	dayEnd := dayStart.AddDate(0, 0, 1)
	byPerson := map[int64][]Interval{}
	for _, v := range visits {
		end, ongoing := visitEnd(v, now)
		start := maxTime(v.StartedAt, dayStart)
		end = minTime(end, dayEnd)
		if !end.After(start) {
			continue
		}
		byPerson[v.PersonID] = append(byPerson[v.PersonID], Interval{Start: start, End: end, Ongoing: ongoing && end.Equal(now)})
	}
	for id, intervals := range byPerson {
		byPerson[id] = merge(intervals)
	}
	return byPerson
}

func merge(intervals []Interval) []Interval {
	sort.Slice(intervals, func(i, j int) bool { return intervals[i].Start.Before(intervals[j].Start) })
	var merged []Interval
	for _, iv := range intervals {
		if n := len(merged); n > 0 && !iv.Start.After(merged[n-1].End) {
			if iv.End.After(merged[n-1].End) {
				merged[n-1].End = iv.End
				merged[n-1].Ongoing = iv.Ongoing
			}
			continue
		}
		merged = append(merged, iv)
	}
	return merged
}

// Daily totals per person for each of the days starting at the given local midnights.
func Daily(visits []store.Visit, dayStarts []time.Time, now time.Time) map[int64][]Day {
	result := map[int64][]Day{}
	people := map[int64]bool{}
	for _, v := range visits {
		people[v.PersonID] = true
	}
	for id := range people {
		result[id] = make([]Day, len(dayStarts))
	}
	for i, start := range dayStarts {
		for id, intervals := range Timeline(visits, start, now) {
			day := Day{Date: start.Format(time.DateOnly)}
			for _, iv := range intervals {
				day.Seconds += int64(iv.End.Sub(iv.Start).Seconds())
			}
			if len(intervals) > 0 {
				first, last := intervals[0].Start, intervals[len(intervals)-1].End
				day.First, day.Last = &first, &last
			}
			result[id][i] = day
		}
		for id := range result {
			if result[id][i].Date == "" {
				result[id][i].Date = start.Format(time.DateOnly)
			}
		}
	}
	return result
}

// LastDays returns the local midnights of the n days ending today, oldest first.
func LastDays(now time.Time, n int, loc *time.Location) []time.Time {
	today := StartOfDay(now, loc)
	days := make([]time.Time, n)
	for i := range days {
		days[i] = today.AddDate(0, 0, i-(n-1))
	}
	return days
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
