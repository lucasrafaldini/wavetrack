// Package testutil holds helpers shared by tests.
package testutil

import (
	"net"
	"testing"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/oui"
	"github.com/lucasrafaldini/wavetrack/internal/store"
)

func Store(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func MAC(t *testing.T, s string) net.HardwareAddr {
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
