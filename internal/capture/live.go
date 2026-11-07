package capture

import (
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// In managed mode a Wi-Fi card only hands over traffic that reaches the host, so listen
// for what phones and laptops broadcast while idle: ARP, DHCP, IPv6 neighbor discovery and mDNS.
const managedFilter = "arp or icmp6 or (udp and (port 67 or port 68 or port 5353))"

// In monitor mode every 802.11 frame is visible; management and data frames carry the sender.
const monitorFilter = "type mgt or type data"

type Live struct {
	handle    *pcap.Handle
	iface     string
	monitor   bool
	out       chan Sighting
	throttle  *throttle
	closeOnce sync.Once
	done      chan struct{}
}

// OpenLive starts capturing on iface. Monitor mode needs an adapter and driver that support it.
func OpenLive(iface string, monitor bool) (*Live, error) {
	device, err := pcapDevice(iface)
	if err != nil {
		return nil, err
	}
	inactive, err := pcap.NewInactiveHandle(device)
	if err != nil {
		return nil, explain(iface, err)
	}
	defer inactive.CleanUp()

	_ = inactive.SetSnapLen(512)
	_ = inactive.SetPromisc(true)
	_ = inactive.SetTimeout(time.Second) // lets Close stop the read loop promptly
	if monitor {
		if err := inactive.SetRFMon(true); err != nil {
			return nil, fmt.Errorf("monitor mode isn't available on %s: %w", iface, err)
		}
	}
	handle, err := inactive.Activate()
	if err != nil {
		return nil, explain(iface, err)
	}

	filter := managedFilter
	if monitor {
		filter = monitorFilter
	}
	if err := handle.SetBPFFilter(filter); err != nil {
		handle.Close()
		return nil, fmt.Errorf("couldn't apply capture filter on %s: %w", iface, err)
	}

	l := &Live{
		handle:   handle,
		iface:    iface,
		monitor:  monitor,
		out:      make(chan Sighting, 256),
		throttle: newThrottle(15 * time.Second),
		done:     make(chan struct{}),
	}
	go l.run()
	return l, nil
}

// pcapDevice maps an interface name to the device libpcap expects. On Windows, Npcap names
// devices \Device\NPF_{GUID} while the system calls them "Wi-Fi"; match them by address.
func pcapDevice(iface string) (string, error) {
	if runtime.GOOS != "windows" || strings.HasPrefix(iface, `\Device\`) {
		return iface, nil
	}
	netIface, err := net.InterfaceByName(iface)
	if err != nil {
		return "", fmt.Errorf("no interface named %q: %w", iface, err)
	}
	addrs, _ := netIface.Addrs()
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return "", explain(iface, err)
	}
	for _, d := range devices {
		for _, da := range d.Addresses {
			for _, a := range addrs {
				if ipnet, ok := a.(*net.IPNet); ok && ipnet.IP.Equal(da.IP) {
					return d.Name, nil
				}
			}
		}
	}
	return "", fmt.Errorf("Npcap doesn't list %q; is the interface connected?", iface)
}

func explain(iface string, err error) error {
	msg := strings.ToLower(err.Error())
	if runtime.GOOS == "windows" && (strings.Contains(msg, "wpcap") || strings.Contains(msg, "dll")) {
		return fmt.Errorf("packet capture needs Npcap: install it from https://npcap.com with \"WinPcap API-compatible mode\" checked: %w", err)
	}
	if strings.Contains(msg, "permission denied") || strings.Contains(msg, "operation not permitted") {
		return fmt.Errorf("no permission to capture on %s; run `wavetrack doctor` for the one-time setup: %w", iface, err)
	}
	return fmt.Errorf("couldn't open %s: %w", iface, err)
}

func (l *Live) run() {
	defer close(l.out)
	source := gopacket.NewPacketSource(l.handle, l.handle.LinkType())
	source.NoCopy = true
	for {
		select {
		case <-l.done:
			return
		case packet, ok := <-source.Packets():
			if !ok {
				return
			}
			if s, ok := l.sighting(packet); ok {
				select {
				case l.out <- s:
				default: // never block capture on a slow consumer
				}
			}
		}
	}
}

func (l *Live) sighting(packet gopacket.Packet) (Sighting, bool) {
	at := packet.Metadata().Timestamp
	if at.IsZero() {
		at = time.Now()
	}
	var mac net.HardwareAddr
	signal := 0

	if l.monitor {
		dot11, ok := packet.Layer(layers.LayerTypeDot11).(*layers.Dot11)
		if !ok {
			return Sighting{}, false
		}
		mac = dot11.Address2
		if radio, ok := packet.Layer(layers.LayerTypeRadioTap).(*layers.RadioTap); ok {
			signal = int(radio.DBMAntennaSignal)
		}
	} else if eth, ok := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet); ok {
		mac = eth.SrcMAC
	}

	if !usable(mac) || !l.throttle.allow(mac.String(), at) {
		return Sighting{}, false
	}
	return Sighting{MAC: append(net.HardwareAddr(nil), mac...), At: at, Signal: signal}, true
}

func (l *Live) Sightings() <-chan Sighting { return l.out }

func (l *Live) Describe() Description {
	mode := "managed"
	if l.monitor {
		mode = "monitor"
	}
	return Description{Kind: "live", Interface: l.iface, Mode: mode}
}

func (l *Live) Close() error {
	l.closeOnce.Do(func() {
		close(l.done)
		l.handle.Close()
	})
	return nil
}
