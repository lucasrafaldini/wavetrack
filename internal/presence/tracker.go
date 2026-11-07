// Package presence turns sightings into visits: a person arrives when one of their
// devices is first seen and leaves once none has been seen for a while.
package presence

import (
	"context"
	"log/slog"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/capture"
	"github.com/lucasrafaldini/wavetrack/internal/oui"
	"github.com/lucasrafaldini/wavetrack/internal/store"
)

// Store is the slice of persistence the tracker needs.
type Store interface {
	Devices() ([]store.Device, error)
	OpenVisits() ([]store.Visit, error)
	StartVisit(personID int64, at time.Time) (int64, error)
	TouchVisit(id int64, at time.Time) error
	EndVisit(id int64, at time.Time) error
	TouchDevice(mac string, at time.Time) error
}

type Identifier interface {
	Identify(mac net.HardwareAddr) oui.Info
}

type EventKind string

const (
	Arrived EventKind = "arrived"
	Left    EventKind = "left"
)

type Event struct {
	Kind     EventKind `json:"kind"`
	PersonID int64     `json:"person_id"`
	At       time.Time `json:"at"`
}

// Stay is a person currently on site.
type Stay struct {
	PersonID int64     `json:"person_id"`
	Since    time.Time `json:"since"`
	LastSeen time.Time `json:"last_seen"`
}

// Nearby is a device seen recently that belongs to nobody. Kept in memory only.
type Nearby struct {
	MAC       string    `json:"mac"`
	Vendor    string    `json:"vendor"`
	Kind      oui.Kind  `json:"kind"`
	Private   bool      `json:"private"`
	Signal    int       `json:"signal"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

type Options struct {
	AwayAfter time.Duration
	MinSignal int
	Logger    *slog.Logger
	// Clock drives periodic sweeps. The demo shifts it to office hours.
	Clock func() time.Time
}

// Persisting every sighting would mean a write every few seconds per device.
const persistEvery = time.Minute

type stay struct {
	visitID     int64
	since       time.Time
	lastSeen    time.Time
	persistedAt time.Time
}

type Tracker struct {
	store Store
	ident Identifier
	opts  Options

	mu          sync.Mutex
	owners      map[string]int64
	present     map[int64]*stay
	nearby      map[string]*Nearby
	deviceTouch map[string]time.Time
}

func New(st Store, ident Identifier, opts Options) *Tracker {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	return &Tracker{
		store:       st,
		ident:       ident,
		opts:        opts,
		owners:      map[string]int64{},
		present:     map[int64]*stay{},
		nearby:      map[string]*Nearby{},
		deviceTouch: map[string]time.Time{},
	}
}

// Start loads devices and resumes visits left open by a previous run.
// Visits whose person has been gone longer than AwayAfter are closed at their last sighting.
func (t *Tracker) Start(now time.Time) error {
	if err := t.ReloadDevices(); err != nil {
		return err
	}
	open, err := t.store.OpenVisits()
	if err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, v := range open {
		if now.Sub(v.LastSeen) > t.opts.AwayAfter {
			if err := t.store.EndVisit(v.ID, v.LastSeen); err != nil {
				return err
			}
			continue
		}
		t.present[v.PersonID] = &stay{visitID: v.ID, since: v.StartedAt, lastSeen: v.LastSeen, persistedAt: v.LastSeen}
	}
	return nil
}

// ReloadDevices refreshes which MAC belongs to whom. Call after people or devices change.
func (t *Tracker) ReloadDevices() error {
	devices, err := t.store.Devices()
	if err != nil {
		return err
	}
	owners := make(map[string]int64, len(devices))
	for _, d := range devices {
		owners[d.MAC] = d.PersonID
	}
	t.mu.Lock()
	t.owners = owners
	for mac := range owners {
		delete(t.nearby, mac)
	}
	t.mu.Unlock()
	return nil
}

// Observe records one sighting.
func (t *Tracker) Observe(s capture.Sighting) {
	mac := s.MAC.String()

	t.mu.Lock()
	personID, owned := t.owners[mac]
	_, known := t.nearby[mac]
	t.mu.Unlock()

	if !owned {
		var info oui.Info
		if !known {
			info = t.ident.Identify(s.MAC) // may hit the network the first time, so outside the lock
		}
		t.mu.Lock()
		t.observeUnknown(mac, info, s)
		t.mu.Unlock()
		return
	}

	if s.Signal != 0 && s.Signal < t.opts.MinSignal {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if last := t.deviceTouch[mac]; s.At.Sub(last) >= persistEvery {
		t.deviceTouch[mac] = s.At
		if err := t.store.TouchDevice(mac, s.At); err != nil {
			t.opts.Logger.Warn("couldn't update device", "mac", mac, "err", err)
		}
	}

	current := t.present[personID]
	if current == nil {
		id, err := t.store.StartVisit(personID, s.At)
		if err != nil {
			t.opts.Logger.Error("couldn't start visit", "person", personID, "err", err)
			return
		}
		t.present[personID] = &stay{visitID: id, since: s.At, lastSeen: s.At, persistedAt: s.At}
		t.record(Event{Kind: Arrived, PersonID: personID, At: s.At})
		return
	}

	if s.At.After(current.lastSeen) {
		current.lastSeen = s.At
	}
	if current.lastSeen.Sub(current.persistedAt) >= persistEvery {
		if err := t.store.TouchVisit(current.visitID, current.lastSeen); err != nil {
			t.opts.Logger.Warn("couldn't update visit", "person", personID, "err", err)
			return
		}
		current.persistedAt = current.lastSeen
	}
}

func (t *Tracker) observeUnknown(mac string, info oui.Info, s capture.Sighting) {
	n := t.nearby[mac]
	if n == nil {
		n = &Nearby{MAC: mac, Vendor: info.Vendor, Kind: info.Kind, Private: info.Randomized, FirstSeen: s.At}
		t.nearby[mac] = n
	}
	if s.At.After(n.LastSeen) {
		n.LastSeen = s.At
	}
	if s.Signal != 0 {
		n.Signal = s.Signal
	}
}

// Sweep ends the visits of everyone unseen for AwayAfter. Their departure is recorded at
// the last sighting, which is when they actually left, not when the timeout noticed.
func (t *Tracker) Sweep(now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for personID, current := range t.present {
		if now.Sub(current.lastSeen) <= t.opts.AwayAfter {
			continue
		}
		if err := t.store.EndVisit(current.visitID, current.lastSeen); err != nil {
			t.opts.Logger.Error("couldn't end visit", "person", personID, "err", err)
			continue
		}
		delete(t.present, personID)
		t.record(Event{Kind: Left, PersonID: personID, At: current.lastSeen})
	}

	for mac, n := range t.nearby {
		if now.Sub(n.LastSeen) > 2*t.opts.AwayAfter {
			delete(t.nearby, mac)
		}
	}
}

func (t *Tracker) record(e Event) {
	t.opts.Logger.Info("presence", "event", e.Kind, "person", e.PersonID, "at", e.At.Format(time.TimeOnly))
}

type Snapshot struct {
	Present []Stay   `json:"present"`
	Nearby  []Nearby `json:"nearby"`
}

// Snapshot returns who is here and which unknown devices were seen within AwayAfter.
func (t *Tracker) Snapshot(now time.Time) Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()

	snap := Snapshot{Present: []Stay{}, Nearby: []Nearby{}}
	for personID, s := range t.present {
		snap.Present = append(snap.Present, Stay{PersonID: personID, Since: s.since, LastSeen: s.lastSeen})
	}
	sort.Slice(snap.Present, func(i, j int) bool { return snap.Present[i].Since.Before(snap.Present[j].Since) })

	for _, n := range t.nearby {
		if now.Sub(n.LastSeen) <= t.opts.AwayAfter {
			snap.Nearby = append(snap.Nearby, *n)
		}
	}
	sort.Slice(snap.Nearby, func(i, j int) bool { return snap.Nearby[i].LastSeen.After(snap.Nearby[j].LastSeen) })
	return snap
}

// Flush writes the latest sighting of everyone present, so a restart resumes accurately.
func (t *Tracker) Flush() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for personID, s := range t.present {
		if err := t.store.TouchVisit(s.visitID, s.lastSeen); err != nil {
			t.opts.Logger.Warn("couldn't save visit", "person", personID, "err", err)
		}
		s.persistedAt = s.lastSeen
	}
}

// Forget drops a person's live state, used when they are archived or erased.
func (t *Tracker) Forget(personID int64) {
	t.mu.Lock()
	delete(t.present, personID)
	t.mu.Unlock()
}

// Run consumes sightings and sweeps periodically until ctx ends or the source closes.
func (t *Tracker) Run(ctx context.Context, source capture.Source) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	sightings := source.Sightings()
	for {
		select {
		case <-ctx.Done():
			return
		case s, ok := <-sightings:
			if !ok {
				return
			}
			t.Observe(s)
		case <-ticker.C:
			t.Sweep(t.opts.Clock())
		}
	}
}
