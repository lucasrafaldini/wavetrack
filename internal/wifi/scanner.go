package wifi

import (
	"fmt"
	"log"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/lucasrafaldini/wavetrack/internal/deviceid"
	"github.com/lucasrafaldini/wavetrack/internal/models"
)

// Scanner é responsável por monitorar dispositivos na rede Wi-Fi
type Scanner struct {
	iface       string
	handle      *pcap.Handle
	devices     map[string]*models.Device
	deviceChan  chan *models.Device
	stopChan    chan bool
	monitorMode bool // true se suporta modo monitor (802.11)
}

// NewScanner cria uma nova instância do scanner
func NewScanner(iface string) *Scanner {
	return &Scanner{
		iface:      iface,
		devices:    make(map[string]*models.Device),
		deviceChan: make(chan *models.Device, 100),
		stopChan:   make(chan bool),
	}
}

// Start inicia o monitoramento da interface de rede
func (s *Scanner) Start() error {
	var err error

	// Abre a interface em modo promíscuo para capturar todos os pacotes
	s.handle, err = pcap.OpenLive(s.iface, 65536, true, pcap.BlockForever)
	if err != nil {
		return fmt.Errorf("erro ao abrir interface %s: %v", s.iface, err)
	}

	// Detecta automaticamente se suporta modo monitor (802.11)
	filter := "type mgt"
	if err := s.handle.SetBPFFilter(filter); err != nil {
		// Se falhar, usa filtro ARP (modo compatibilidade para macOS/Windows)
		log.Printf("Modo 802.11 não disponível, usando modo ARP (compatibilidade)")
		log.Printf("⚠️  Dados limitados: sem RSSI, frequência ou canal")
		s.monitorMode = false
		filter = "arp or (udp and port 67) or (udp and port 68)"
		if err := s.handle.SetBPFFilter(filter); err != nil {
			log.Printf("Aviso: Não foi possível definir filtro BPF: %v", err)
			// Continua sem filtro (captura tudo)
		}
	} else {
		log.Printf("✓ Modo monitor 802.11 ativado - captura completa habilitada")
		log.Printf("✓ Dados disponíveis: RSSI, frequência, canal, taxa de transmissão")
		s.monitorMode = true
	}

	log.Printf("Scanner iniciado na interface %s", s.iface)

	// Inicia o processamento de pacotes em uma goroutine
	go s.capturePackets()

	return nil
}

// capturePackets processa os pacotes capturados
func (s *Scanner) capturePackets() {
	packetSource := gopacket.NewPacketSource(s.handle, s.handle.LinkType())

	for {
		select {
		case <-s.stopChan:
			log.Println("Parando captura de pacotes...")
			return
		case packet := <-packetSource.Packets():
			s.processPacket(packet)
		}
	}
}

// processPacket extrai informações de dispositivos dos pacotes capturados
func (s *Scanner) processPacket(packet gopacket.Packet) {
	var macAddr string
	signal := 0    // 0 = sinal não disponível (modo não-monitor)
	frequency := 0 // 0 = frequência não disponível
	channel := 0   // 0 = canal não disponível

	// Tenta primeiro extrair de pacotes 802.11 (modo monitor)
	dot11Layer := packet.Layer(layers.LayerTypeDot11)
	if dot11Layer != nil && s.monitorMode {
		dot11, _ := dot11Layer.(*layers.Dot11)
		macAddr = dot11.Address2.String()

		// Extrai informações do RadioTap (disponível em modo monitor)
		if radioTap := packet.Layer(layers.LayerTypeRadioTap); radioTap != nil {
			rt, _ := radioTap.(*layers.RadioTap)

			// Força do sinal em dBm
			signal = int(rt.DBMAntennaSignal)

			// Frequência em MHz
			frequency = int(rt.ChannelFrequency)

			// Calcula o canal a partir da frequência
			if frequency >= 2412 && frequency <= 2484 {
				// 2.4 GHz (canais 1-14)
				if frequency == 2484 {
					channel = 14
				} else {
					channel = (frequency - 2407) / 5
				}
			} else if frequency >= 5160 && frequency <= 5885 {
				// 5 GHz (canais 32-177)
				channel = (frequency - 5000) / 5
			} else if frequency >= 5955 && frequency <= 7115 {
				// 6 GHz (Wi-Fi 6E)
				channel = (frequency - 5950) / 5
			}
		}
	} else {
		// Modo compatibilidade: extrai de ARP ou Ethernet
		// Nota: Não há informação de RSSI, frequência ou canal em modo não-monitor
		if arpLayer := packet.Layer(layers.LayerTypeARP); arpLayer != nil {
			arp, _ := arpLayer.(*layers.ARP)
			macAddr = fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
				arp.SourceHwAddress[0], arp.SourceHwAddress[1],
				arp.SourceHwAddress[2], arp.SourceHwAddress[3],
				arp.SourceHwAddress[4], arp.SourceHwAddress[5])
		} else if ethLayer := packet.Layer(layers.LayerTypeEthernet); ethLayer != nil {
			eth, _ := ethLayer.(*layers.Ethernet)
			macAddr = eth.SrcMAC.String()
		}
	}

	if macAddr == "" || macAddr == "00:00:00:00:00:00" {
		return
	}

	// Atualiza ou cria registro do dispositivo
	now := time.Now()
	if device, exists := s.devices[macAddr]; exists {
		device.LastSeen = now
		device.SignalStrength = signal
		device.Frequency = frequency
		device.Channel = channel
		device.IsActive = true
	} else {
		// Identifica vendor e tipo do dispositivo
		vendor, deviceType := deviceid.IdentifyDevice(macAddr)
		deviceInfo := deviceid.IdentifyDeviceDetailed(macAddr)

		device := &models.Device{
			MACAddress:     macAddr,
			Type:           deviceType,
			Vendor:         vendor,
			SignalStrength: signal,
			Frequency:      frequency,
			Channel:        channel,
			FirstSeen:      now,
			LastSeen:       now,
			IsActive:       true,
			IsAmbiguous:    deviceInfo.IsAmbiguous,
			PossibleTypes:  deviceInfo.PossibleTypes,
		}
		s.devices[macAddr] = device
		s.deviceChan <- device

		if s.monitorMode {
			log.Printf("Novo dispositivo detectado: %s [%s] | %d dBm | %d MHz (Canal %d)",
				macAddr, deviceid.FormatDeviceInfoWithTooltip(deviceInfo), signal, frequency, channel)
		} else {
			log.Printf("Novo dispositivo detectado: %s [%s]", macAddr, deviceid.FormatDeviceInfoWithTooltip(deviceInfo))
		}
	}
}

// GetDevices retorna o canal de dispositivos detectados
func (s *Scanner) GetDevices() <-chan *models.Device {
	return s.deviceChan
}

// GetActiveDevices retorna dispositivos vistos recentemente
func (s *Scanner) GetActiveDevices(duration time.Duration) []*models.Device {
	var active []*models.Device
	threshold := time.Now().Add(-duration)

	for _, device := range s.devices {
		if device.LastSeen.After(threshold) {
			active = append(active, device)
		}
	}

	return active
}

// Stop para o scanner
func (s *Scanner) Stop() {
	close(s.stopChan)
	if s.handle != nil {
		s.handle.Close()
	}
	log.Println("Scanner parado")
}
