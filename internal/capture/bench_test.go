package capture

import (
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

func benchPacket(b *testing.B, first gopacket.LayerType, ls ...gopacket.SerializableLayer) []byte {
	b.Helper()
	buf := gopacket.NewSerializeBuffer()
	if err := gopacket.SerializeLayers(buf, gopacket.SerializeOptions{FixLengths: true}, ls...); err != nil {
		b.Fatal(err)
	}
	return buf.Bytes()
}

// Decoding a frame and extracting its sender: the per-packet cost of live capture.
// Each iteration decodes the raw bytes, as the capture loop does.
func BenchmarkSightingEthernet(b *testing.B) {
	phone, _ := net.ParseMAC("a4:83:e7:21:5c:10")
	raw := benchPacket(b, layers.LayerTypeEthernet,
		&layers.Ethernet{SrcMAC: phone, DstMAC: net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, EthernetType: layers.EthernetTypeARP},
		&layers.ARP{AddrType: layers.LinkTypeEthernet, Protocol: layers.EthernetTypeIPv4, HwAddressSize: 6, ProtAddressSize: 4,
			Operation: layers.ARPRequest, SourceHwAddress: phone, SourceProtAddress: []byte{192, 168, 1, 20},
			DstHwAddress: make([]byte, 6), DstProtAddress: []byte{192, 168, 1, 1}})
	l := &Live{throttle: newThrottle(15 * time.Second)}
	at := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	b.SetBytes(int64(len(raw)))
	b.ReportAllocs()
	for b.Loop() {
		p := gopacket.NewPacket(raw, layers.LayerTypeEthernet, gopacket.Default)
		p.Metadata().Timestamp = at
		at = at.Add(time.Millisecond)
		l.sighting(p)
	}
}

func BenchmarkSightingMonitor(b *testing.B) {
	phone, _ := net.ParseMAC("8c:f5:a3:77:19:2e")
	bcast := net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	raw := benchPacket(b, layers.LayerTypeRadioTap,
		&layers.RadioTap{Present: []layers.RadioTapPresent{layers.RadioTapPresentDBMAntennaSignal}, RadioTapValues: []layers.RadioTapNamespace{{DBMAntennaSignal: -61}}},
		&layers.Dot11{Type: layers.Dot11TypeMgmtProbeReq, Address1: bcast, Address2: phone, Address3: bcast},
		gopacket.Payload([]byte{0, 0}))
	l := &Live{monitor: true, throttle: newThrottle(15 * time.Second)}
	at := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	b.SetBytes(int64(len(raw)))
	b.ReportAllocs()
	for b.Loop() {
		p := gopacket.NewPacket(raw, layers.LayerTypeRadioTap, gopacket.Default)
		p.Metadata().Timestamp = at
		at = at.Add(time.Millisecond)
		l.sighting(p)
	}
}
