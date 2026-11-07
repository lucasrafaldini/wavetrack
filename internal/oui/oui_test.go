package oui

import (
	"errors"
	"net"
	"testing"
)

func TestRandomizedAddressesSkipTheLookup(t *testing.T) {
	calls := 0
	id := New(func(string) (string, error) { calls++; return "Apple, Inc.", nil })
	mac, _ := net.ParseMAC("da:a1:19:6b:3f:f6")
	if info := id.Identify(mac); !info.Randomized || calls != 0 {
		t.Fatalf("info = %+v, lookups = %d", info, calls)
	}
}

func TestLookupsAreCachedPerPrefix(t *testing.T) {
	calls := 0
	id := New(func(string) (string, error) { calls++; return "Samsung Electronics Co.,Ltd.", nil })
	a, _ := net.ParseMAC("8c:f5:a3:00:00:01")
	b, _ := net.ParseMAC("8c:f5:a3:00:00:02")
	first, second := id.Identify(a), id.Identify(b)
	if calls != 1 || first != second || first.Kind != Phone || first.Vendor != "Samsung Electronics" {
		t.Fatalf("calls = %d, info = %+v", calls, first)
	}
}

func TestFailedLookupIsUnknown(t *testing.T) {
	id := New(func(string) (string, error) { return "", errors.New("offline") })
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	if info := id.Identify(mac); info.Vendor != "Unknown" || info.Kind != Unknown {
		t.Fatalf("info = %+v", info)
	}
}

func TestClassifyMatchesWholeWords(t *testing.T) {
	cases := map[string]Kind{
		"Hewlett Packard":            Computer,
		"HP Inc.":                    Computer,
		"Ubiquiti Inc":               Network,
		"Espressif Inc.":             IoT,
		"Apple, Inc.":                Unknown, // makes phones and computers alike
		"Shenzhen Photon Technology": Unknown, // contains "hp" inside a word
		"Vivotek Inc":                Unknown, // contains "vivo" inside a word
	}
	for vendor, want := range cases {
		if got := Classify(vendor); got != want {
			t.Errorf("Classify(%q) = %s, want %s", vendor, got, want)
		}
	}
}
