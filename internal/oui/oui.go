// Package oui names a device's manufacturer from its MAC address and guesses what kind of device it is.
package oui

import (
	"net"
	"strings"
	"sync"

	"github.com/lucasrafaldini/ouija"
)

type Kind string

const (
	Phone    Kind = "phone"
	Computer Kind = "computer"
	Network  Kind = "network"
	IoT      Kind = "iot"
	Unknown  Kind = "unknown"
)

type Info struct {
	Vendor string `json:"vendor"`
	Kind   Kind   `json:"kind"`
	// Randomized is true for locally administered addresses, which phones and laptops use
	// as private Wi-Fi addresses. They carry no manufacturer.
	Randomized bool `json:"randomized"`
}

// Lookup resolves a vendor name for a MAC. Swappable in tests.
type Lookup func(mac string) (string, error)

type Identifier struct {
	lookup Lookup
	mu     sync.RWMutex
	cache  map[string]Info
}

func New(lookup Lookup) *Identifier {
	return &Identifier{lookup: lookup, cache: map[string]Info{}}
}

// Default uses the IEEE registry through OUIja, which downloads and caches it on first use.
func Default() *Identifier {
	ouija.SetQuietMode(true)
	return New(ouija.GetVendor)
}

// IsRandomized reports a locally administered (private) address: bit 1 of the first octet.
func IsRandomized(mac net.HardwareAddr) bool {
	return len(mac) > 0 && mac[0]&0x02 != 0
}

func (id *Identifier) Identify(mac net.HardwareAddr) Info {
	if IsRandomized(mac) {
		return Info{Vendor: "Private address", Kind: Unknown, Randomized: true}
	}
	if len(mac) < 3 {
		return Info{Vendor: "Unknown", Kind: Unknown}
	}
	prefix := mac[:3].String()

	id.mu.RLock()
	info, ok := id.cache[prefix]
	id.mu.RUnlock()
	if ok {
		return info
	}

	info = Info{Vendor: "Unknown", Kind: Unknown}
	if name, err := id.lookup(mac.String()); err == nil && strings.TrimSpace(name) != "" {
		info = Info{Vendor: tidy(name), Kind: Classify(name)}
	}
	id.mu.Lock()
	id.cache[prefix] = info
	id.mu.Unlock()
	return info
}

var kindsByVendor = []struct {
	kind  Kind
	words []string
}{
	{Network, []string{"cisco", "tp-link", "tplink", "netgear", "d-link", "linksys", "ubiquiti", "mikrotik", "aruba", "juniper", "fortinet", "zyxel", "ruckus", "meraki", "arcadyan", "sagemcom", "technicolor"}},
	{IoT, []string{"espressif", "raspberry", "arduino", "nordic", "tuya", "sonoff", "shelly", "philips lighting", "signify", "ecobee", "nest", "ring", "wyze", "sonos", "roku", "amazon technologies", "chromecast"}},
	{Phone, []string{"oneplus", "oppo", "vivo", "realme", "xiaomi", "huawei", "honor", "motorola", "samsung", "nothing technology", "fairphone", "zte", "tcl"}},
	{Computer, []string{"intel", "dell", "hewlett", "hp inc", "lenovo", "acer", "asustek", "toshiba", "fujitsu", "micro-star", "gigabyte", "framework", "liteon", "azurewave", "realtek", "qualcomm atheros"}},
}

// Classify guesses a device kind from its manufacturer. Apple and Google make everything,
// so they stay Unknown rather than guessing wrong.
func Classify(vendor string) Kind {
	name := " " + strings.ToLower(vendor) + " "
	for _, entry := range kindsByVendor {
		for _, word := range entry.words {
			if containsWord(name, word) {
				return entry.kind
			}
		}
	}
	return Unknown
}

func containsWord(haystack, word string) bool {
	for i := strings.Index(haystack, word); i >= 0; {
		before := haystack[i-1]
		afterIndex := i + len(word)
		after := byte(' ')
		if afterIndex < len(haystack) {
			after = haystack[afterIndex]
		}
		if !isLetter(before) && !isLetter(after) {
			return true
		}
		next := strings.Index(haystack[i+1:], word)
		if next < 0 {
			return false
		}
		i += next + 1
	}
	return false
}

func isLetter(b byte) bool { return b >= 'a' && b <= 'z' }

// tidy drops legal suffixes the registry includes ("Apple, Inc." → "Apple").
func tidy(name string) string {
	name = strings.TrimSpace(name)
	for _, suffix := range []string{", Inc.", " Inc.", " Inc", ", Ltd.", " Co.,Ltd.", " Co., Ltd.", " Corporation", " Corp.", " GmbH", " LLC", " Limited"} {
		name = strings.TrimSuffix(name, suffix)
	}
	return strings.TrimRight(name, " ,")
}
