package capture

import (
	"net"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func packet(t *testing.T, first gopacket.LayerType, ls ...gopacket.SerializableLayer) gopacket.Packet {
	t.Helper()
	buf := gopacket.NewSerializeBuffer()
	if err := gopacket.SerializeLayers(buf, gopacket.SerializeOptions{FixLengths: true}, ls...); err != nil {
		t.Fatal(err)
	}
	p := gopacket.NewPacket(buf.Bytes(), first, gopacket.Default)
	p.Metadata().Timestamp = time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	return p
}

func TestManagedModeReadsTheEthernetSender(t *testing.T) {
	phone, _ := net.ParseMAC("a4:83:e7:21:5c:10")
	l := &Live{throttle: newThrottle(15 * time.Second)}
	arp := packet(t, layers.LayerTypeEthernet,
		&layers.Ethernet{SrcMAC: phone, DstMAC: net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, EthernetType: layers.EthernetTypeARP},
		&layers.ARP{AddrType: layers.LinkTypeEthernet, Protocol: layers.EthernetTypeIPv4, HwAddressSize: 6, ProtAddressSize: 4,
			Operation: layers.ARPRequest, SourceHwAddress: phone, SourceProtAddress: []byte{192, 168, 1, 20},
			DstHwAddress: make([]byte, 6), DstProtAddress: []byte{192, 168, 1, 1}})

	s, ok := l.sighting(arp)
	if !ok || s.MAC.String() != phone.String() || s.Signal != 0 || !s.At.Equal(arp.Metadata().Timestamp) {
		t.Fatalf("sighting = %+v, %v", s, ok)
	}
	if _, ok := l.sighting(arp); ok {
		t.Fatal("the same device was reported twice within the throttle interval")
	}
}

func TestMonitorModeReadsTheTransmitterAndSignal(t *testing.T) {
	phone, _ := net.ParseMAC("8c:f5:a3:77:19:2e")
	l := &Live{monitor: true, throttle: newThrottle(15 * time.Second)}
	frame := packet(t, layers.LayerTypeRadioTap,
		&layers.RadioTap{Present: layers.RadioTapPresentDBMAntennaSignal, DBMAntennaSignal: -61},
		&layers.Dot11{Type: layers.Dot11TypeMgmtProbeReq, Address1: net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, Address2: phone, Address3: net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		gopacket.Payload([]byte{0, 0}))

	s, ok := l.sighting(frame)
	if !ok || s.MAC.String() != phone.String() || s.Signal != -61 {
		t.Fatalf("sighting = %+v, %v", s, ok)
	}
}

func TestBroadcastSendersAreIgnored(t *testing.T) {
	l := &Live{throttle: newThrottle(15 * time.Second)}
	frame := packet(t, layers.LayerTypeEthernet,
		&layers.Ethernet{SrcMAC: net.HardwareAddr{0x01, 0, 0x5e, 0, 0, 0xfb}, DstMAC: net.HardwareAddr{0x01, 0, 0x5e, 0, 0, 0xfb}, EthernetType: layers.EthernetTypeIPv4},
		gopacket.Payload([]byte{1, 2, 3}))
	if _, ok := l.sighting(frame); ok {
		t.Fatal("multicast sender produced a sighting")
	}
}
