// Package importer brings data over from WaveTrack 2.x databases.
package importer

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/store"
)

type Result struct {
	People int
	Visits int
}

var layouts = []string{
	time.RFC3339Nano,
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
}

func parseTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	// go-sqlite3 appends a monotonic clock reading ("m=+1.23") to some 2.x rows
	if i := strings.Index(raw, " m="); i >= 0 {
		raw = raw[:i]
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized time %q", raw)
}

// V2 copies employees and their arrival/departure history from a 2.x database into dst.
// Arrivals are paired with the next departure of the same device; an arrival without one
// can't be trusted and is skipped.
func V2(oldPath string, dst *store.Store, now time.Time) (Result, error) {
	old, err := sql.Open("sqlite3", "file:"+oldPath+"?mode=ro")
	if err != nil {
		return Result{}, err
	}
	defer old.Close()

	rows, err := old.Query(`SELECT mac_address, name, COALESCE(department, '') FROM employees`)
	if err != nil {
		return Result{}, fmt.Errorf("%s doesn't look like a WaveTrack 2 database: %w", oldPath, err)
	}
	type employee struct{ mac, name, team string }
	var employees []employee
	for rows.Next() {
		var e employee
		if err := rows.Scan(&e.mac, &e.name, &e.team); err != nil {
			rows.Close()
			return Result{}, err
		}
		employees = append(employees, e)
	}
	rows.Close()

	var result Result
	personFor := map[string]int64{}
	for _, e := range employees {
		mac, err := store.NormalizeMAC(e.mac)
		if err != nil {
			continue
		}
		p, err := dst.CreatePerson(e.name, e.team, nil, now)
		if err != nil {
			return result, err
		}
		if err := dst.AddDevice(store.Device{MAC: mac, PersonID: p.ID, Label: "Imported device", AddedAt: now}); err != nil {
			return result, err
		}
		personFor[mac] = p.ID
		result.People++
	}

	events, err := old.Query(`SELECT mac_address, event_type, timestamp FROM events
		WHERE event_type IN ('arrival', 'departure') ORDER BY mac_address, timestamp`)
	if err != nil {
		return result, err
	}
	defer events.Close()

	type event struct {
		kind string
		at   time.Time
	}
	byMAC := map[string][]event{}
	for events.Next() {
		var mac, kind, raw string
		if err := events.Scan(&mac, &kind, &raw); err != nil {
			return result, err
		}
		at, err := parseTime(raw)
		if err != nil {
			continue
		}
		if normalized, err := store.NormalizeMAC(mac); err == nil {
			byMAC[normalized] = append(byMAC[normalized], event{kind, at})
		}
	}

	for mac, list := range byMAC {
		personID, ok := personFor[mac]
		if !ok {
			continue
		}
		sort.Slice(list, func(i, j int) bool { return list[i].at.Before(list[j].at) })
		var open *time.Time
		for _, e := range list {
			switch {
			case e.kind == "arrival" && open == nil:
				at := e.at
				open = &at
			case e.kind == "departure" && open != nil:
				if e.at.After(*open) {
					if err := dst.InsertVisit(personID, *open, e.at); err != nil {
						return result, err
					}
					result.Visits++
				}
				open = nil
			}
		}
	}
	return result, nil
}
