package netinfo

import "testing"

func TestChoosePrefersConfigThenConnectedWiFi(t *testing.T) {
	list := []Interface{
		{Name: "en7", Up: true, IPv4: "10.0.0.2"}, // wired
		{Name: "en0", Wireless: true, Up: true, IPv4: "192.168.1.10"},
	}
	if got, _ := Choose("wlx1234", list); got != "wlx1234" {
		t.Errorf("configured interface ignored: %q", got)
	}
	if got, _ := Choose("", list); got != "en0" {
		t.Errorf("chose %q, want the Wi-Fi interface", got)
	}
	if _, err := Choose("", list[:1]); err == nil {
		t.Error("guessed a wired interface instead of asking")
	}
}

func TestParseARP(t *testing.T) {
	if got := parseBSDARP("? (192.168.1.20) at 3a:1:c:4f:0:2 on en0 ifscope [ethernet]\n"); got != "3a:1:c:4f:0:2" {
		t.Errorf("bsd = %q", got)
	}
	if got := parseBSDARP("? (192.168.1.21) at (incomplete) on en0 ifscope [ethernet]\n"); got != "" {
		t.Errorf("incomplete entry = %q", got)
	}
	table := "IP address       HW type     Flags       HW address            Mask     Device\n" +
		"192.168.1.20     0x1         0x2         aa:bb:cc:dd:ee:ff     *        wlan0\n" +
		"192.168.1.21     0x1         0x0         00:00:00:00:00:00     *        wlan0\n"
	if got := parseProcARP(table, "192.168.1.20"); got != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("proc = %q", got)
	}
	if got := parseProcARP(table, "192.168.1.21"); got != "" {
		t.Errorf("unresolved proc entry = %q", got)
	}
	mac, err := ParseLooseMAC("3a:1:c:4f:0:2")
	if err != nil || mac.String() != "3a:01:0c:4f:00:02" {
		t.Errorf("loose = %v, %v", mac, err)
	}
}

func TestWindowsParsers(t *testing.T) {
	arp := "\r\nInterface: 192.168.1.10 --- 0x12\r\n  Internet Address      Physical Address      Type\r\n  192.168.1.20          3a-01-0c-4f-00-02     dynamic\r\n"
	if got := parseWindowsARP(arp, "192.168.1.20"); got != "3a:01:0c:4f:00:02" {
		t.Errorf("arp = %q", got)
	}
	if got := parseWindowsARP(arp, "192.168.1.99"); got != "" {
		t.Errorf("missing entry = %q", got)
	}
	netsh := "\r\nThere is 1 interface on the system:\r\n\r\n    Name                   : Wi-Fi\r\n    Description            : Intel(R) Wi-Fi 6 AX201\r\n    State                  : connected\r\n"
	if got := parseNetshInterfaces(netsh); len(got) != 1 || got[0] != "Wi-Fi" {
		t.Errorf("netsh = %q", got)
	}
	if !looksWireless("Wi-Fi") || !looksWireless("WLAN Wireless") || looksWireless("Ethernet") {
		t.Error("looksWireless misjudged a Windows name")
	}
}

func TestMACForIPRejectsLoopbackAndGarbage(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "not-an-ip", "::1"} {
		if _, err := MACForIP(ip); err == nil {
			t.Errorf("MACForIP(%q) succeeded", ip)
		}
	}
}
