// Package testutil holds helpers shared by tests.
package testutil

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/oui"
	"github.com/lucasrafaldini/wavetrack/internal/store"
)

func Store(t testing.TB) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func MAC(t testing.TB, s string) net.HardwareAddr {
	t.Helper()
	mac, err := net.ParseMAC(s)
	if err != nil {
		t.Fatal(err)
	}
	return mac
}

// Identifier names every device "Acme" without touching the network.
type Identifier struct{}

func (Identifier) Identify(mac net.HardwareAddr) oui.Info {
	if oui.IsRandomized(mac) {
		return oui.Info{Vendor: "Private address", Kind: oui.Unknown, Randomized: true}
	}
	return oui.Info{Vendor: "Acme", Kind: oui.Phone}
}

// Day returns 2026-11-05 at hh:mm local time.
func Day(hour, minute int) time.Time {
	return time.Date(2026, 11, 5, hour, minute, 0, 0, time.Local)
}

// Office fills st with people (each with one phone) and their history: on every
// weekday of the last days, a morning and an afternoon visit around a lunch break.
// The last day's afternoon visits are left open, as if everyone were still in.
// It returns the people's IDs and their phones' MAC addresses.
func Office(t testing.TB, st *store.Store, people, days int, now time.Time) ([]int64, []string) {
	t.Helper()
	ids := make([]int64, people)
	macs := make([]string, people)
	for i := range people {
		p, err := st.CreatePerson(fmt.Sprintf("Person %03d", i), []string{"Engineering", "Design", "Operations", "Sales"}[i%4], nil, now)
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = p.ID
		macs[i] = fmt.Sprintf("a4:83:e7:%02x:%02x:%02x", i>>16&0xff, i>>8&0xff, i&0xff)
		if err := st.AddDevice(store.Device{MAC: macs[i], PersonID: p.ID, Label: "Phone", AddedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	var visits []store.Visit
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for d := days - 1; d >= 0; d-- {
		day := today.AddDate(0, 0, -d)
		if wd := day.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		for i, id := range ids {
			jitter := time.Duration(i%37) * time.Minute
			morning := day.Add(8*time.Hour + 30*time.Minute + jitter)
			lunch := day.Add(12*time.Hour + jitter)
			back := lunch.Add(time.Hour)
			leave := day.Add(17*time.Hour + 30*time.Minute + jitter)
			visits = append(visits, store.Visit{PersonID: id, StartedAt: morning, LastSeen: lunch, EndedAt: &lunch})
			if d == 0 && leave.After(now) {
				if back.Before(now) {
					visits = append(visits, store.Visit{PersonID: id, StartedAt: back, LastSeen: now})
				}
				continue
			}
			end := leave
			visits = append(visits, store.Visit{PersonID: id, StartedAt: back, LastSeen: end, EndedAt: &end})
		}
	}
	if err := st.InsertVisits(visits); err != nil {
		t.Fatal(err)
	}
	return ids, macs
}
