// Package netinfo finds the capture interface, the address phones should use to reach
// WaveTrack, and the MAC behind an IP on the local network.
package netinfo

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
)

type Interface struct {
	Name     string `json:"name"`
	Wireless bool   `json:"wireless"`
	Up       bool   `json:"up"`
	IPv4     string `json:"ipv4,omitempty"`
}

// Interfaces lists non-loopback interfaces, wireless and connected ones first.
func Interfaces() ([]Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	wireless := wirelessNames()
	var result []Interface
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || len(iface.HardwareAddr) == 0 {
			continue
		}
		result = append(result, Interface{
			Name:     iface.Name,
			Wireless: wireless[iface.Name] || looksWireless(iface.Name),
			Up:       iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagRunning != 0,
			IPv4:     ipv4(iface),
		})
	}
	sortCandidates(result)
	return result, nil
}

func sortCandidates(list []Interface) {
	score := func(i Interface) int {
		s := 0
		if i.Up {
			s += 4
		}
		if i.IPv4 != "" {
			s += 2
		}
		if i.Wireless {
			s++
		}
		return s
	}
	sort.SliceStable(list, func(a, b int) bool { return score(list[a]) > score(list[b]) })
}

// Choose picks the capture interface: an explicit choice always wins; otherwise the
// connected Wi-Fi interface; otherwise nothing, because guessing a wired port would
// silently capture the wrong network.
func Choose(configured string, candidates []Interface) (string, error) {
	if configured != "" {
		return configured, nil
	}
	for _, c := range candidates {
		if c.Wireless && c.Up && c.IPv4 != "" {
			return c.Name, nil
		}
	}
	return "", errors.New("no connected Wi-Fi interface found; set capture.interface or pass --interface (see `wavetrack interfaces`)")
}

func looksWireless(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, "wl") || strings.HasPrefix(lower, "wifi") || strings.HasPrefix(lower, "wi-fi") || strings.Contains(lower, "wireless")
}

func ipv4(iface net.Interface) string {
	addrs, _ := iface.Addrs()
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if v4 := ipnet.IP.To4(); v4 != nil && !v4.IsLinkLocalUnicast() {
				return v4.String()
			}
		}
	}
	return ""
}

// IPv4 returns the interface's address, used to build invite links phones can open.
func IPv4(name string) string {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return ""
	}
	return ipv4(*iface)
}

func wirelessNames() map[string]bool {
	names := map[string]bool{}
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("/usr/sbin/networksetup", "-listallhardwareports").Output()
		if err != nil {
			return names
		}
		var port string
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := scanner.Text()
			if v, ok := strings.CutPrefix(line, "Hardware Port: "); ok {
				port = v
			} else if v, ok := strings.CutPrefix(line, "Device: "); ok && (port == "Wi-Fi" || port == "AirPort") {
				names[v] = true
			}
		}
	case "linux":
		entries, _ := os.ReadDir("/sys/class/net")
		for _, e := range entries {
			if _, err := os.Stat("/sys/class/net/" + e.Name() + "/wireless"); err == nil {
				names[e.Name()] = true
			}
		}
	case "windows":
		out, err := exec.Command("netsh", "wlan", "show", "interfaces").Output()
		if err == nil {
			for _, name := range parseNetshInterfaces(string(out)) {
				names[name] = true
			}
		}
	}
	return names
}

// parseNetshInterfaces reads interface names from `netsh wlan show interfaces`:
//
//	Name                   : Wi-Fi
func parseNetshInterfaces(out string) []string {
	var names []string
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if ok && strings.TrimSpace(key) == "Name" {
			names = append(names, strings.TrimSpace(value))
		}
	}
	return names
}

// MACForIP looks up the hardware address of a host on the local network in the ARP cache.
// The host just sent us an HTTP request, so its entry is fresh.
func MACForIP(ip string) (net.HardwareAddr, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() == nil {
		return nil, fmt.Errorf("%q isn't an IPv4 address on this network", ip)
	}
	if parsed.IsLoopback() {
		return nil, errors.New("this request came from WaveTrack's own machine")
	}
	var raw string
	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile("/proc/net/arp")
		if err != nil {
			return nil, err
		}
		raw = parseProcARP(string(data), parsed.String())
	case "windows":
		out, err := exec.Command("arp", "-a", parsed.String()).Output()
		if err != nil {
			return nil, fmt.Errorf("arp lookup failed: %w", err)
		}
		raw = parseWindowsARP(string(out), parsed.String())
	default:
		out, err := exec.Command("/usr/sbin/arp", "-n", parsed.String()).Output()
		if err != nil {
			return nil, fmt.Errorf("arp lookup failed: %w", err)
		}
		raw = parseBSDARP(string(out))
	}
	if raw == "" {
		return nil, fmt.Errorf("no ARP entry for %s", ip)
	}
	return ParseLooseMAC(raw)
}

func parseProcARP(table, ip string) string {
	for _, line := range strings.Split(table, "\n")[1:] {
		fields := strings.Fields(line)
		if len(fields) >= 4 && fields[0] == ip && fields[3] != "00:00:00:00:00:00" {
			return fields[3]
		}
	}
	return ""
}

// 192.168.1.20          3a-01-0c-4f-00-02     dynamic
func parseWindowsARP(out, ip string) string {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == ip {
			return strings.ReplaceAll(fields[1], "-", ":")
		}
	}
	return ""
}

// "? (192.168.1.20) at 3a:1:c:4f:0:2 on en0 ifscope [ethernet]"
func parseBSDARP(out string) string {
	fields := strings.Fields(out)
	for i, f := range fields {
		if f == "at" && i+1 < len(fields) && fields[i+1] != "(incomplete)" {
			return fields[i+1]
		}
	}
	return ""
}

// ParseLooseMAC accepts BSD's unpadded octets ("3a:1:c:4f:0:2").
func ParseLooseMAC(raw string) (net.HardwareAddr, error) {
	parts := strings.Split(raw, ":")
	if len(parts) != 6 {
		return nil, fmt.Errorf("unexpected MAC %q", raw)
	}
	for i, p := range parts {
		if len(p) == 1 {
			parts[i] = "0" + p
		}
	}
	return net.ParseMAC(strings.Join(parts, ":"))
}
