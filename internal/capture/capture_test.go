package capture

import (
	"net"
	"testing"
	"time"
)

func TestThrottleAllowsOneSightingPerInterval(t *testing.T) {
	th := newThrottle(15 * time.Second)
	t0 := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	if !th.allow("a", t0) || th.allow("a", t0.Add(5*time.Second)) || !th.allow("b", t0.Add(5*time.Second)) || !th.allow("a", t0.Add(16*time.Second)) {
		t.Fatal("throttle misbehaved")
	}
}

func TestUsableRejectsBroadcastAndZero(t *testing.T) {
	for raw, want := range map[string]bool{
		"a4:83:e7:21:5c:10": true,
		"ff:ff:ff:ff:ff:ff": false,
		"01:00:5e:00:00:fb": false, // multicast (mDNS)
		"00:00:00:00:00:00": false,
	} {
		mac, _ := net.ParseMAC(raw)
		if got := usable(mac); got != want {
			t.Errorf("usable(%s) = %v", raw, got)
		}
	}
}
