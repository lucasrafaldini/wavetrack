// Package capture turns network traffic into sightings: "this MAC was here at this time".
package capture

import (
	"net"
	"sync"
	"time"
)

type Sighting struct {
	MAC net.HardwareAddr
	At  time.Time
	// Signal in dBm, or 0 when the capture mode can't measure it.
	Signal int
}

// Source produces sightings until it is closed.
type Source interface {
	Sightings() <-chan Sighting
	Describe() Description
	Close() error
}

type Description struct {
	Kind      string `json:"kind"` // "live" or "demo"
	Interface string `json:"interface,omitempty"`
	Mode      string `json:"mode"` // "monitor", "managed" or "simulated"
}

// throttle limits each MAC to one sighting per interval, so a chatty device
// doesn't flood the tracker with thousands of identical updates.
type throttle struct {
	interval time.Duration
	mu       sync.Mutex
	last     map[string]time.Time
}

func newThrottle(interval time.Duration) *throttle {
	return &throttle{interval: interval, last: map[string]time.Time{}}
}

func (t *throttle) allow(mac string, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if last, ok := t.last[mac]; ok && now.Sub(last) < t.interval {
		return false
	}
	t.last[mac] = now
	if len(t.last) > 50_000 {
		for key, seen := range t.last {
			if now.Sub(seen) > time.Hour {
				delete(t.last, key)
			}
		}
	}
	return true
}

// usable filters out addresses that can't belong to a nearby device.
func usable(mac net.HardwareAddr) bool {
	if len(mac) != 6 {
		return false
	}
	if mac[0]&0x01 != 0 { // multicast and broadcast
		return false
	}
	for _, b := range mac {
		if b != 0 {
			return true
		}
	}
	return false
}
