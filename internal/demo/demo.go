// Package demo simulates an office so WaveTrack can be tried, tested and documented
// without capture hardware or root privileges.
package demo

import (
	"hash/fnv"
	"math/rand/v2"
	"net"
	"sync"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/capture"
	"github.com/lucasrafaldini/wavetrack/internal/oui"
	"github.com/lucasrafaldini/wavetrack/internal/store"
)

type Member struct {
	Name    string
	Team    string
	Devices []Device
	// HomeBias is the chance of working remotely on a weekday.
	HomeBias float64
}

type Device struct {
	MAC    string
	Label  string
	Vendor string
	Kind   oui.Kind
}

var Office = []Member{
	{"Ana Ribeiro", "Design", []Device{{"a4:83:e7:21:5c:10", "iPhone", "Apple", oui.Phone}, {"f0:18:98:4a:02:11", "MacBook Pro", "Apple", oui.Computer}}, 0.10},
	{"Bruno Costa", "Engineering", []Device{{"8c:f5:a3:77:19:2e", "Galaxy S24", "Samsung", oui.Phone}}, 0.25},
	{"Chiara Rossi", "Research", []Device{{"3c:22:fb:90:0d:41", "iPhone", "Apple", oui.Phone}}, 0.15},
	{"Daniel Okafor", "Engineering", []Device{{"f4:f5:d8:12:6b:03", "Pixel 9", "Google", oui.Phone}, {"a4:34:d9:5e:71:b2", "ThinkPad", "Intel", oui.Computer}}, 0.20},
	{"Elena Petrova", "Product", []Device{{"ac:bc:32:61:9f:24", "iPhone", "Apple", oui.Phone}}, 0.10},
	{"Felipe Andrade", "Operations", []Device{{"64:09:80:3d:aa:57", "Redmi Note", "Xiaomi", oui.Phone}}, 0.05},
	{"Hana Sato", "Design", []Device{{"f0:18:98:2b:c4:68", "iPhone", "Apple", oui.Phone}}, 0.30},
	{"Isaac Mendes", "Engineering", []Device{{"8c:f5:a3:0a:3f:79", "Galaxy A55", "Samsung", oui.Phone}}, 0.15},
	{"Julia Fischer", "Research", []Device{{"3c:22:fb:4d:e1:8a", "iPhone", "Apple", oui.Phone}}, 0.20},
	{"Kofi Mensah", "Operations", []Device{{"f4:f5:d8:6e:20:9b", "Pixel 8a", "Google", oui.Phone}}, 0.05},
	{"Lea Martin", "Product", []Device{{"ac:bc:32:18:77:ac", "iPhone", "Apple", oui.Phone}}, 0.35},
	{"Marco Silva", "Engineering", []Device{{"a4:83:e7:9c:51:bd", "iPhone", "Apple", oui.Phone}}, 0.10},
}

// Devices nobody has registered: a printer, a speaker, a guest's phone with a private address.
var visitors = []Device{
	{"3c:d9:2b:44:10:c3", "", "HP", oui.Computer},
	{"48:a6:b8:20:7e:d4", "", "Sonos", oui.IoT},
	{"8c:f5:a3:51:c2:e5", "", "Samsung", oui.Phone},
	{"da:a1:19:6b:3f:f6", "", "Private address", oui.Unknown},
	{"7e:12:9a:c0:44:07", "", "Private address", oui.Unknown},
}

type window struct{ start, end time.Time }

func seed(parts ...string) uint64 {
	h := fnv.New64a()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return h.Sum64()
}

func at(day time.Time, minutes float64) time.Time {
	return day.Add(time.Duration(minutes * float64(time.Minute)))
}

// schedule is deterministic per person and day, so the live simulation and the seeded
// history always agree.
func schedule(m Member, day time.Time) []window {
	r := rand.New(rand.NewPCG(seed(m.Name, day.Format(time.DateOnly)), 7))
	weekend := day.Weekday() == time.Saturday || day.Weekday() == time.Sunday
	if weekend {
		if r.Float64() > 0.06 {
			return nil
		}
		start := 10*60 + r.Float64()*120
		return []window{{at(day, start), at(day, start+90+r.Float64()*120)}}
	}
	if r.Float64() < m.HomeBias {
		return nil
	}
	arrive := 8*60 + 15 + r.NormFloat64()*25 + 30
	leave := 17*60 + 45 + r.NormFloat64()*35
	if r.Float64() < 0.08 { // half day
		leave = 13*60 + r.Float64()*60
	}
	arrive = clamp(arrive, 7*60+30, 11*60)
	leave = clamp(leave, arrive+120, 21*60)

	if r.Float64() < 0.55 && leave > 14*60 { // out for lunch
		lunch := 12*60 + r.Float64()*80
		back := lunch + 40 + r.Float64()*35
		if lunch > arrive+30 && back < leave-30 {
			return []window{{at(day, arrive), at(day, lunch)}, {at(day, back), at(day, leave)}}
		}
	}
	return []window{{at(day, arrive), at(day, leave)}}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Clock returns a clock for the demo. Outside office hours it is shifted to 14:20, and on
// weekends to the Friday before, so the dashboard always shows a lively office.
func Clock(real func() time.Time) func() time.Time {
	now := real()
	weekend := 0
	switch now.Weekday() {
	case time.Saturday:
		weekend = 1
	case time.Sunday:
		weekend = 2
	}
	if h := now.Hour(); weekend == 0 && h >= 9 && h < 18 {
		return real
	}
	target := time.Date(now.Year(), now.Month(), now.Day()-weekend, 14, 20, 0, 0, now.Location())
	offset := target.Sub(now)
	return func() time.Time { return real().Add(offset) }
}

// Seed fills an empty store with the office and `days` days of history up to now.
// Visits in progress are left open so the tracker resumes them.
func Seed(st *store.Store, now time.Time, days int) error {
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	created := today.AddDate(0, 0, -days)

	for _, m := range Office {
		person, err := st.CreatePerson(m.Name, m.Team, nil, created)
		if err != nil {
			return err
		}
		for _, d := range m.Devices {
			if err := st.AddDevice(store.Device{MAC: d.MAC, PersonID: person.ID, Label: d.Label, Vendor: d.Vendor, Kind: string(d.Kind), AddedAt: created}); err != nil {
				return err
			}
		}
		for i := days; i >= 0; i-- {
			day := today.AddDate(0, 0, -i)
			for _, w := range schedule(m, day) {
				switch {
				case !w.start.Before(now):
					// later today: the live simulation will produce it
				case w.end.Before(now):
					if err := st.InsertVisit(person.ID, w.start, w.end); err != nil {
						return err
					}
				default:
					id, err := st.StartVisit(person.ID, w.start)
					if err != nil {
						return err
					}
					if err := st.TouchVisit(id, now.Add(-30*time.Second)); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// Source emits sightings for whoever the schedule says is in the office right now.
type Source struct {
	clock     func() time.Time
	out       chan capture.Sighting
	done      chan struct{}
	closeOnce sync.Once
	rng       *rand.Rand
}

func NewSource(clock func() time.Time) *Source {
	s := &Source{
		clock: clock,
		out:   make(chan capture.Sighting, 64),
		done:  make(chan struct{}),
		rng:   rand.New(rand.NewPCG(uint64(clock().UnixNano()), 3)),
	}
	go s.run()
	return s
}

func (s *Source) run() {
	defer close(s.out)
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()
	s.emitAll()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.emitAll()
		}
	}
}

func (s *Source) emitAll() {
	now := s.clock()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for _, m := range Office {
		if !inside(schedule(m, day), now) {
			continue
		}
		for _, d := range m.Devices {
			if s.rng.Float64() < 0.35 { // devices are chatty, but not every few seconds
				s.emit(d.MAC, now, -48-s.rng.IntN(25))
			}
		}
	}
	if h := now.Hour(); h >= 8 && h < 20 {
		for i, d := range visitors {
			if (i < 2 || s.rng.Float64() < 0.5) && s.rng.Float64() < 0.4 {
				s.emit(d.MAC, now, -60-s.rng.IntN(25))
			}
		}
	}
}

func inside(windows []window, now time.Time) bool {
	for _, w := range windows {
		if !now.Before(w.start) && now.Before(w.end) {
			return true
		}
	}
	return false
}

func (s *Source) emit(mac string, now time.Time, signal int) {
	hw, _ := net.ParseMAC(mac)
	select {
	case s.out <- capture.Sighting{MAC: hw, At: now, Signal: signal}:
	case <-s.done:
	}
}

func (s *Source) Sightings() <-chan capture.Sighting { return s.out }

func (s *Source) Describe() capture.Description {
	return capture.Description{Kind: "demo", Mode: "simulated"}
}

func (s *Source) Close() error {
	s.closeOnce.Do(func() { close(s.done) })
	return nil
}

// Identifier names the simulated devices without a network lookup and defers to fallback otherwise.
type Identifier struct {
	Fallback interface {
		Identify(net.HardwareAddr) oui.Info
	}
}

func (i Identifier) Identify(mac net.HardwareAddr) oui.Info {
	for _, d := range visitors {
		if d.MAC == mac.String() {
			return oui.Info{Vendor: d.Vendor, Kind: d.Kind, Randomized: oui.IsRandomized(mac)}
		}
	}
	return i.Fallback.Identify(mac)
}
