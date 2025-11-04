package deviceid

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/lucasrafaldini/ouija"
)

// VendorInfo contém informações sobre o fabricante
type VendorInfo struct {
	Name          string
	DeviceType    string
	PossibleTypes []string // Para fabricantes ambíguos
	IsAmbiguous   bool     // Indica se o fabricante faz múltiplos tipos
}

// Cache para armazenar consultas de API
type VendorCache struct {
	mu    sync.RWMutex
	cache map[string]VendorInfo
}

var vendorCache = &VendorCache{
	cache: make(map[string]VendorInfo),
}

// init inicializa o sistema de identificação com OUIja
func init() {
	log.Printf("✅ Sistema de identificação inicializado com OUIja (base IEEE oficial)")
	log.Printf("🌐 OUIja: Biblioteca de identificação de fabricantes via MAC address")
	log.Printf("📋 Fallback: Base de dados local para casos offline")
}

// ouiDatabase contém prefixos MAC (OUI) mais comuns - usado como fallback final
// NOTA: Agora usando OUIja como método principal - esta base serve apenas como fallback
// quando a biblioteca OUIja não conseguir identificar o dispositivo (casos offline)
var ouiDatabase = map[string]VendorInfo{
	// Apple - dispositivos principais
	"00:03:93": {Name: "Apple", DeviceType: "incerto", PossibleTypes: []string{"laptop", "smartphone", "tablet"}, IsAmbiguous: true},
	"28:cf:e9": {Name: "Apple", DeviceType: "incerto", PossibleTypes: []string{"laptop", "smartphone", "tablet"}, IsAmbiguous: true},
	"a4:5e:60": {Name: "Apple", DeviceType: "incerto", PossibleTypes: []string{"laptop", "smartphone", "tablet"}, IsAmbiguous: true},
	"ac:de:48": {Name: "Apple", DeviceType: "incerto", PossibleTypes: []string{"laptop", "smartphone", "tablet"}, IsAmbiguous: true},
	"f0:18:98": {Name: "Apple", DeviceType: "incerto", PossibleTypes: []string{"laptop", "smartphone", "tablet"}, IsAmbiguous: true},
	"dc:a6:32": {Name: "Apple", DeviceType: "incerto", PossibleTypes: []string{"laptop", "smartphone", "tablet"}, IsAmbiguous: true},

	// Samsung - múltiplos dispositivos
	"2c:44:01": {Name: "Samsung", DeviceType: "incerto", PossibleTypes: []string{"smartphone", "tablet", "laptop", "smarttv", "router"}, IsAmbiguous: true},
	"38:2d:d1": {Name: "Samsung", DeviceType: "incerto", PossibleTypes: []string{"smartphone", "tablet", "laptop", "smarttv", "router"}, IsAmbiguous: true},
	"78:d6:f0": {Name: "Samsung", DeviceType: "incerto", PossibleTypes: []string{"smartphone", "tablet", "laptop", "smarttv", "router"}, IsAmbiguous: true},
	"dc:ef:09": {Name: "Samsung", DeviceType: "incerto", PossibleTypes: []string{"smartphone", "tablet", "laptop", "smarttv", "router"}, IsAmbiguous: true},

	// Intel - principalmente laptops
	"a4:83:e7": {Name: "Intel", DeviceType: "laptop", IsAmbiguous: false},
	"cc:2f:71": {Name: "Intel", DeviceType: "laptop", IsAmbiguous: false},
	"80:86:f2": {Name: "Intel", DeviceType: "laptop", IsAmbiguous: false},

	// Qualcomm - principalmente smartphones
	"00:03:7f": {Name: "Qualcomm", DeviceType: "incerto", PossibleTypes: []string{"smartphone", "iot", "router"}, IsAmbiguous: true},
	"00:0a:f5": {Name: "Qualcomm", DeviceType: "incerto", PossibleTypes: []string{"smartphone", "iot", "router"}, IsAmbiguous: true},

	// Xiaomi - múltiplos dispositivos
	"34:ce:00": {Name: "Xiaomi", DeviceType: "incerto", PossibleTypes: []string{"smartphone", "laptop", "iot", "tablet"}, IsAmbiguous: true},
	"50:8f:4c": {Name: "Xiaomi", DeviceType: "incerto", PossibleTypes: []string{"smartphone", "laptop", "iot", "tablet"}, IsAmbiguous: true},

	// Huawei
	"00:e0:fc": {Name: "Huawei", DeviceType: "incerto", PossibleTypes: []string{"smartphone", "tablet", "laptop", "router"}, IsAmbiguous: true},
	"ac:5a:fc": {Name: "Huawei", DeviceType: "incerto", PossibleTypes: []string{"smartphone", "tablet", "laptop", "router"}, IsAmbiguous: true},

	// Broadcom - principalmente equipamentos de rede
	"84:0b:bb": {Name: "Broadcom", DeviceType: "incerto", PossibleTypes: []string{"router", "laptop", "smartphone"}, IsAmbiguous: true},
	"b8:27:eb": {Name: "Broadcom", DeviceType: "iot", IsAmbiguous: false}, // Raspberry Pi

	// TP-Link - equipamentos de rede
	"50:c7:bf": {Name: "TP-Link", DeviceType: "router", IsAmbiguous: false},
	"ec:08:6b": {Name: "TP-Link", DeviceType: "router", IsAmbiguous: false},

	// Motorola
	"cc:fb:65": {Name: "Motorola", DeviceType: "incerto", PossibleTypes: []string{"smartphone", "router"}, IsAmbiguous: true},

	// LG
	"10:68:3f": {Name: "LG Electronics", DeviceType: "incerto", PossibleTypes: []string{"smartphone", "smarttv", "laptop"}, IsAmbiguous: true},

	// ASUS
	"2c:56:dc": {Name: "ASUS", DeviceType: "incerto", PossibleTypes: []string{"laptop", "router", "smartphone"}, IsAmbiguous: true},
	"ac:9e:17": {Name: "ASUS", DeviceType: "incerto", PossibleTypes: []string{"laptop", "router", "smartphone"}, IsAmbiguous: true},

	// Espressif (ESP32/ESP8266) - IoT
	"30:ae:a4": {Name: "Espressif", DeviceType: "iot", IsAmbiguous: false},
	"24:6f:28": {Name: "Espressif", DeviceType: "iot", IsAmbiguous: false},

	// Shenzhen (fabricantes chineses)
	"d8:c6:78": {Name: "Shenzhen", DeviceType: "incerto", PossibleTypes: []string{"smartphone", "iot"}, IsAmbiguous: true},

	// Sony
	"08:00:46": {Name: "Sony", DeviceType: "incerto", PossibleTypes: []string{"smartphone", "smarttv", "console", "laptop"}, IsAmbiguous: true},
	"54:84:1b": {Name: "Sony", DeviceType: "incerto", PossibleTypes: []string{"smartphone", "smarttv", "console", "laptop"}, IsAmbiguous: true},

	// Microsoft
	"00:50:f2": {Name: "Microsoft", DeviceType: "incerto", PossibleTypes: []string{"laptop", "console", "iot"}, IsAmbiguous: true},
	"7c:ed:8d": {Name: "Microsoft", DeviceType: "incerto", PossibleTypes: []string{"laptop", "console", "iot"}, IsAmbiguous: true},

	// Nintendo
	"00:17:ab": {Name: "Nintendo", DeviceType: "console", IsAmbiguous: false},
	"a4:c0:e1": {Name: "Nintendo", DeviceType: "console", IsAmbiguous: false},

	// Dell
	"00:14:22": {Name: "Dell", DeviceType: "laptop", IsAmbiguous: false},
	"b8:ca:3a": {Name: "Dell", DeviceType: "laptop", IsAmbiguous: false},

	// HP/Hewlett-Packard
	"00:1b:78": {Name: "Hewlett Packard", DeviceType: "laptop", IsAmbiguous: false},
	"2c:27:d7": {Name: "Hewlett Packard", DeviceType: "laptop", IsAmbiguous: false},

	// Lenovo
	"00:1a:4b": {Name: "Lenovo", DeviceType: "laptop", IsAmbiguous: false},
	"54:ee:75": {Name: "Lenovo", DeviceType: "laptop", IsAmbiguous: false},

	// Cisco
	"00:0c:41": {Name: "Cisco", DeviceType: "router", IsAmbiguous: false},
	"00:23:ab": {Name: "Cisco", DeviceType: "router", IsAmbiguous: false},

	// Netgear
	"00:09:5b": {Name: "Netgear", DeviceType: "router", IsAmbiguous: false},
	"a0:40:a0": {Name: "Netgear", DeviceType: "router", IsAmbiguous: false},

	// D-Link
	"00:05:5d": {Name: "D-Link", DeviceType: "router", IsAmbiguous: false},
	"cc:b2:55": {Name: "D-Link", DeviceType: "router", IsAmbiguous: false},

	// Linksys
	"00:06:25": {Name: "Linksys", DeviceType: "router", IsAmbiguous: false},
	"48:f8:b3": {Name: "Linksys", DeviceType: "router", IsAmbiguous: false},

	// Amazon (Fire TV, Echo, etc.)
	"00:fc:8b": {Name: "Amazon", DeviceType: "smarttv", IsAmbiguous: false},
	"38:f7:3d": {Name: "Amazon", DeviceType: "iot", IsAmbiguous: false},

	// Google (Chromecast, Nest, etc.)
	"00:1a:11": {Name: "Google", DeviceType: "smarttv", IsAmbiguous: false},
	"64:16:66": {Name: "Google", DeviceType: "iot", IsAmbiguous: false},

	// Roku
	"dc:3a:5e": {Name: "Roku", DeviceType: "smarttv", IsAmbiguous: false},
	"b0:a7:37": {Name: "Roku", DeviceType: "smarttv", IsAmbiguous: false},

	// OnePlus
	"ac:37:43": {Name: "OnePlus", DeviceType: "smartphone", IsAmbiguous: false},
	"e8:b2:ac": {Name: "OnePlus", DeviceType: "smartphone", IsAmbiguous: false},

	// Oppo
	"20:6b:e7": {Name: "Oppo", DeviceType: "smartphone", IsAmbiguous: false},
	"94:e9:79": {Name: "Oppo", DeviceType: "smartphone", IsAmbiguous: false},

	// Vivo
	"8c:be:be": {Name: "Vivo", DeviceType: "smartphone", IsAmbiguous: false},
	"f8:e6:1a": {Name: "Vivo", DeviceType: "smartphone", IsAmbiguous: false},

	// Realme
	"02:69:6a": {Name: "Realme", DeviceType: "smartphone", IsAmbiguous: false},

	// Acer
	"00:02:e3": {Name: "Acer", DeviceType: "laptop", IsAmbiguous: false},
	"00:21:85": {Name: "Acer", DeviceType: "laptop", IsAmbiguous: false},

	// Toshiba
	"00:00:ba": {Name: "Toshiba", DeviceType: "laptop", IsAmbiguous: false},
	"00:80:d0": {Name: "Toshiba", DeviceType: "laptop", IsAmbiguous: false},

	// Marvell (chipsets)
	"00:50:43": {Name: "Marvell", DeviceType: "incerto", PossibleTypes: []string{"router", "laptop", "iot"}, IsAmbiguous: true},

	// Ralink/MediaTek
	"00:0c:43": {Name: "Ralink", DeviceType: "incerto", PossibleTypes: []string{"smartphone", "tablet", "iot"}, IsAmbiguous: true},
}

// lookupVendorLocal consulta apenas nossa base OUI local expandida
// TODO v2.0: Refatorar para usar biblioteca própria de consulta MAC address
// Sistema futuro incluirá cache, atualizações automáticas e múltiplas fontes
func lookupVendorLocal(mac string) (VendorInfo, error) {
	// Extrai os primeiros 3 octetos do MAC (OUI)
	parts := strings.Split(strings.ToLower(mac), ":")
	if len(parts) < 3 {
		return VendorInfo{}, fmt.Errorf("MAC inválido")
	}

	oui := strings.Join(parts[:3], ":")

	// Verifica na nossa base expandida
	if info, exists := ouiDatabase[oui]; exists {
		return info, nil
	}

	return VendorInfo{}, fmt.Errorf("vendor não encontrado na base local")
}

// lookupVendorOUIja consulta vendor usando a biblioteca OUIja
// Esta é a nova implementação que substitui a base de dados hardcoded
func lookupVendorOUIja(mac string) (VendorInfo, error) {
	// Utiliza a biblioteca OUIja para buscar o fabricante
	vendor, err := ouija.GetVendor(mac)
	if err != nil {
		return VendorInfo{}, fmt.Errorf("vendor não encontrado via OUIja: %v", err)
	}

	// Aplica a lógica de inferência de tipo de dispositivo
	deviceInfo := inferDeviceType(vendor)
	deviceInfo.Name = vendor // Garante que o nome do fabricante seja mantido

	return deviceInfo, nil
}

// inferDeviceType determina o tipo de dispositivo baseado no fabricante
// Retorna informações sobre ambiguidade quando fabricante faz múltiplos tipos
func inferDeviceType(company string) VendorInfo {
	company = strings.ToLower(company)

	// Fabricantes claramente de smartphones apenas
	if strings.Contains(company, "oneplus") ||
		strings.Contains(company, "oppo") ||
		strings.Contains(company, "vivo") ||
		strings.Contains(company, "realme") ||
		strings.Contains(company, "nokia") && strings.Contains(company, "mobile") {
		return VendorInfo{
			Name:        company,
			DeviceType:  "smartphone",
			IsAmbiguous: false,
		}
	}

	// Fabricantes claramente de laptops/desktops apenas
	if strings.Contains(company, "intel") ||
		strings.Contains(company, "dell") ||
		strings.Contains(company, "hp") ||
		strings.Contains(company, "hewlett") ||
		strings.Contains(company, "lenovo") && !strings.Contains(company, "mobile") ||
		strings.Contains(company, "acer") ||
		strings.Contains(company, "toshiba") ||
		strings.Contains(company, "fujitsu") ||
		strings.Contains(company, "msi") ||
		strings.Contains(company, "gigabyte") {
		return VendorInfo{
			Name:        company,
			DeviceType:  "laptop",
			IsAmbiguous: false,
		}
	}

	// Fabricantes claramente de equipamentos de rede apenas
	if strings.Contains(company, "cisco") ||
		strings.Contains(company, "tp-link") ||
		strings.Contains(company, "netgear") ||
		strings.Contains(company, "d-link") ||
		strings.Contains(company, "linksys") ||
		strings.Contains(company, "ubiquiti") ||
		strings.Contains(company, "mikrotik") ||
		strings.Contains(company, "aruba") ||
		strings.Contains(company, "juniper") ||
		strings.Contains(company, "fortinet") {
		return VendorInfo{
			Name:        company,
			DeviceType:  "router",
			IsAmbiguous: false,
		}
	}

	// Fabricantes claramente IoT apenas
	if strings.Contains(company, "espressif") ||
		strings.Contains(company, "raspberry") ||
		strings.Contains(company, "arduino") ||
		strings.Contains(company, "nordic") ||
		strings.Contains(company, "tuya smart") ||
		strings.Contains(company, "sonoff") {
		return VendorInfo{
			Name:        company,
			DeviceType:  "iot",
			IsAmbiguous: false,
		}
	}

	// FABRICANTES AMBÍGUOS (fazem múltiplos tipos)

	// Apple - principalmente laptops e smartphones
	if strings.Contains(company, "apple") {
		if strings.Contains(company, "iphone") || strings.Contains(company, "mobile") {
			return VendorInfo{
				Name:        company,
				DeviceType:  "smartphone",
				IsAmbiguous: false,
			}
		}
		if strings.Contains(company, "ipad") {
			return VendorInfo{
				Name:        company,
				DeviceType:  "tablet",
				IsAmbiguous: false,
			}
		}
		// Apple genérico - pode ser MacBook, iPhone, iPad, Apple TV, etc.
		return VendorInfo{
			Name:          company,
			DeviceType:    "incerto",
			PossibleTypes: []string{"laptop", "smartphone", "tablet", "smarttv"},
			IsAmbiguous:   true,
		}
	}

	// Samsung - faz de tudo: smartphones, TVs, laptops, tablets, routers
	if strings.Contains(company, "samsung") {
		return VendorInfo{
			Name:          company,
			DeviceType:    "incerto",
			PossibleTypes: []string{"smartphone", "tablet", "laptop", "smarttv", "router"},
			IsAmbiguous:   true,
		}
	}

	// LG - smartphones, TVs, laptops
	if strings.Contains(company, "lg") {
		return VendorInfo{
			Name:          company,
			DeviceType:    "incerto",
			PossibleTypes: []string{"smartphone", "smarttv", "laptop"},
			IsAmbiguous:   true,
		}
	}

	// Xiaomi - smartphones, laptops, IoT, tablets, TVs
	if strings.Contains(company, "xiaomi") {
		return VendorInfo{
			Name:          company,
			DeviceType:    "incerto",
			PossibleTypes: []string{"smartphone", "laptop", "iot", "tablet", "smarttv"},
			IsAmbiguous:   true,
		}
	}

	// Huawei - principalmente smartphones, mas também routers e laptops
	if strings.Contains(company, "huawei") {
		return VendorInfo{
			Name:          company,
			DeviceType:    "incerto",
			PossibleTypes: []string{"smartphone", "tablet", "laptop", "router"},
			IsAmbiguous:   true,
		}
	}

	// Qualcomm - principalmente smartphones, mas também IoT e routers
	if strings.Contains(company, "qualcomm") {
		return VendorInfo{
			Name:          company,
			DeviceType:    "incerto",
			PossibleTypes: []string{"smartphone", "iot", "router"},
			IsAmbiguous:   true,
		}
	}

	// ASUS - laptops, routers, smartphones
	if strings.Contains(company, "asus") {
		return VendorInfo{
			Name:          company,
			DeviceType:    "incerto",
			PossibleTypes: []string{"laptop", "router", "smartphone"},
			IsAmbiguous:   true,
		}
	}

	// Sony - smartphones, TVs, consoles, laptops
	if strings.Contains(company, "sony") {
		if strings.Contains(company, "mobile") {
			return VendorInfo{
				Name:        company,
				DeviceType:  "smartphone",
				IsAmbiguous: false,
			}
		}
		if strings.Contains(company, "computer") || strings.Contains(company, "playstation") {
			return VendorInfo{
				Name:        company,
				DeviceType:  "console",
				IsAmbiguous: false,
			}
		}
		return VendorInfo{
			Name:          company,
			DeviceType:    "incerto",
			PossibleTypes: []string{"smartphone", "smarttv", "console", "laptop"},
			IsAmbiguous:   true,
		}
	}

	// Motorola - principalmente smartphones, mas também routers
	if strings.Contains(company, "motorola") {
		return VendorInfo{
			Name:          company,
			DeviceType:    "incerto",
			PossibleTypes: []string{"smartphone", "router"},
			IsAmbiguous:   true,
		}
	}

	// Broadcom - principalmente routers, mas também encontrado em laptops/smartphones
	if strings.Contains(company, "broadcom") {
		return VendorInfo{
			Name:          company,
			DeviceType:    "incerto",
			PossibleTypes: []string{"router", "laptop", "smartphone"},
			IsAmbiguous:   true,
		}
	}

	// MediaTek - principalmente smartphones, mas também IoT e tablets
	if strings.Contains(company, "mediatek") {
		return VendorInfo{
			Name:          company,
			DeviceType:    "incerto",
			PossibleTypes: []string{"smartphone", "tablet", "iot"},
			IsAmbiguous:   true,
		}
	}

	// Microsoft - laptops (Surface), consoles (Xbox), IoT
	if strings.Contains(company, "microsoft") {
		if strings.Contains(company, "surface") {
			return VendorInfo{
				Name:        company,
				DeviceType:  "laptop",
				IsAmbiguous: false,
			}
		}
		return VendorInfo{
			Name:          company,
			DeviceType:    "incerto",
			PossibleTypes: []string{"laptop", "console", "iot"},
			IsAmbiguous:   true,
		}
	}

	// Consoles específicos
	if strings.Contains(company, "nintendo") {
		return VendorInfo{
			Name:        company,
			DeviceType:  "console",
			IsAmbiguous: false,
		}
	}

	// Smart TVs e streaming
	if strings.Contains(company, "amazon") ||
		strings.Contains(company, "roku") ||
		strings.Contains(company, "chromecast") {
		return VendorInfo{
			Name:        company,
			DeviceType:  "smarttv",
			IsAmbiguous: false,
		}
	}

	// Fabricantes chineses genéricos - geralmente smartphones ou IoT
	if strings.Contains(company, "shenzhen") ||
		strings.Contains(company, "guangzhou") ||
		strings.Contains(company, "dongguan") ||
		strings.Contains(company, "beijing") ||
		strings.Contains(company, "hangzhou") {
		return VendorInfo{
			Name:          company,
			DeviceType:    "incerto",
			PossibleTypes: []string{"smartphone", "iot"},
			IsAmbiguous:   true,
		}
	}

	// Fallback baseado em palavras-chave
	if strings.Contains(company, "mobile") || strings.Contains(company, "phone") {
		return VendorInfo{
			Name:        company,
			DeviceType:  "smartphone",
			IsAmbiguous: false,
		}
	}

	if strings.Contains(company, "computer") || strings.Contains(company, "laptop") {
		return VendorInfo{
			Name:        company,
			DeviceType:  "laptop",
			IsAmbiguous: false,
		}
	}

	if strings.Contains(company, "network") || strings.Contains(company, "router") {
		return VendorInfo{
			Name:        company,
			DeviceType:  "router",
			IsAmbiguous: false,
		}
	}

	if strings.Contains(company, "smart") || strings.Contains(company, "iot") {
		return VendorInfo{
			Name:        company,
			DeviceType:  "iot",
			IsAmbiguous: false,
		}
	}

	// Completamente desconhecido
	return VendorInfo{
		Name:        company,
		DeviceType:  "unknown",
		IsAmbiguous: false,
	}
}

// getDeviceEmoji retorna o emoji apropriado para o tipo de dispositivo
func getDeviceEmoji(deviceType string) string {
	switch deviceType {
	case "smartphone":
		return "📱"
	case "tablet":
		return "📲"
	case "laptop":
		return "💻"
	case "router":
		return "🌐"
	case "iot":
		return "🔗"
	case "smarttv":
		return "📺"
	case "console":
		return "🎮"
	case "incerto":
		return "❔"
	default:
		return "❓"
	}
}

// FormatDeviceInfo retorna uma string formatada com emoji para o dispositivo
func FormatDeviceInfo(vendor, deviceType string) string {
	emoji := getDeviceEmoji(deviceType)
	return fmt.Sprintf("%s %s - %s", emoji, vendor, deviceType)
}

// FormatDeviceInfoWithTooltip retorna informação formatada incluindo tipos possíveis
func FormatDeviceInfoWithTooltip(info VendorInfo) string {
	emoji := getDeviceEmoji(info.DeviceType)

	if info.IsAmbiguous && len(info.PossibleTypes) > 0 {
		possibleEmojis := make([]string, len(info.PossibleTypes))
		for i, deviceType := range info.PossibleTypes {
			possibleEmojis[i] = fmt.Sprintf("%s %s", getDeviceEmoji(deviceType), deviceType)
		}
		tooltip := strings.Join(possibleEmojis, ", ")
		return fmt.Sprintf("%s %s - %s (pode ser: %s)", emoji, info.Name, info.DeviceType, tooltip)
	}

	return fmt.Sprintf("%s %s - %s", emoji, info.Name, info.DeviceType)
}

// IdentifyDevice identifica o tipo e fabricante usando OSINT + cache + fallback
func IdentifyDevice(mac string) (vendor string, deviceType string) {
	info := IdentifyDeviceDetailed(mac)
	return info.Name, info.DeviceType
}

// IdentifyDeviceDetailed retorna informações completas incluindo ambiguidade
func IdentifyDeviceDetailed(mac string) VendorInfo {
	mac = strings.ToLower(mac)
	parts := strings.Split(mac, ":")
	if len(parts) < 3 {
		return VendorInfo{Name: "Unknown", DeviceType: "unknown", IsAmbiguous: false}
	}

	oui := strings.Join(parts[:3], ":")

	// 1. Verifica cache primeiro (performance)
	vendorCache.mu.RLock()
	if info, exists := vendorCache.cache[oui]; exists {
		vendorCache.mu.RUnlock()
		return info
	}
	vendorCache.mu.RUnlock()

	// 2. Tenta biblioteca OUIja (método principal)
	if info, err := lookupVendorOUIja(mac); err == nil {
		vendorCache.mu.Lock()
		vendorCache.cache[oui] = info
		vendorCache.mu.Unlock()

		log.Printf("🌐 OUIja: %s [%s] (base IEEE oficial)",
			mac, FormatDeviceInfoWithTooltip(info))
		return info
	}

	// 3. Fallback: Tenta biblioteca OUI local
	if info, err := lookupVendorLocal(mac); err == nil {
		vendorCache.mu.Lock()
		vendorCache.cache[oui] = info
		vendorCache.mu.Unlock()

		log.Printf("📋 LOCAL: %s [%s] (base OUI IEEE backup)",
			mac, FormatDeviceInfoWithTooltip(info))
		return info
	}

	// 4. Fallback para base local (OUIs mais comuns)
	if baseInfo, ok := ouiDatabase[oui]; ok {
		// Aplica a nova lógica de inferência à base local também
		detailedInfo := inferDeviceType(baseInfo.Name)
		detailedInfo.Name = baseInfo.Name // Mantém o nome da base local

		vendorCache.mu.Lock()
		vendorCache.cache[oui] = detailedInfo
		vendorCache.mu.Unlock()

		log.Printf("📋 FALLBACK: %s [%s] (base interna)",
			mac, FormatDeviceInfoWithTooltip(detailedInfo))
		return detailedInfo
	}

	// 4. Não encontrado - cacheia como desconhecido para evitar consultas repetidas
	unknown := VendorInfo{Name: "Unknown", DeviceType: "unknown", IsAmbiguous: false}
	vendorCache.mu.Lock()
	vendorCache.cache[oui] = unknown
	vendorCache.mu.Unlock()

	return unknown
}

// GetDetailedVendorInfo retorna informações detalhadas usando OUIja
// Inclui informações adicionais como MAC normalizado
func GetDetailedVendorInfo(mac string) (*VendorDetailedInfo, error) {
	// Busca informações detalhadas via OUIja
	result, err := ouija.LookupVendor(mac)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar informações detalhadas: %v", err)
	}

	// Aplica a lógica de inferência de tipo
	deviceInfo := inferDeviceType(result.Vendor)

	// Extrai OUI do MAC address
	parts := strings.Split(strings.ToLower(result.MAC), ":")
	oui := ""
	if len(parts) >= 3 {
		oui = strings.Join(parts[:3], ":")
	}

	detailedInfo := &VendorDetailedInfo{
		MAC:           result.MAC,
		Vendor:        result.Vendor,
		OUI:           oui,
		DeviceType:    deviceInfo.DeviceType,
		IsAmbiguous:   deviceInfo.IsAmbiguous,
		PossibleTypes: deviceInfo.PossibleTypes,
	}

	return detailedInfo, nil
}

// SearchDevicesByVendor busca dispositivos por fabricante usando OUIja
func SearchDevicesByVendor(vendorName string) ([]string, error) {
	macs, err := ouija.GetMACsForVendor(vendorName)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar MACs para vendor %s: %v", vendorName, err)
	}
	return macs, nil
}

// SearchVendorsByPattern busca fabricantes por padrão usando OUIja
func SearchVendorsByPattern(pattern string) ([]*VendorStats, error) {
	vendors := ouija.SearchVendors(pattern)

	stats := make([]*VendorStats, len(vendors))
	for i, vendor := range vendors {
		stats[i] = &VendorStats{
			Vendor: vendor.Vendor,
			Count:  vendor.Count,
		}
	}

	return stats, nil
}

// GetTopVendors retorna os principais fabricantes por número de OUIs
func GetTopVendors(limit int) []*VendorStats {
	vendors := ouija.GetTopVendors(limit)

	stats := make([]*VendorStats, len(vendors))
	for i, vendor := range vendors {
		stats[i] = &VendorStats{
			Vendor: vendor.Vendor,
			Count:  vendor.Count,
		}
	}

	return stats
}

// GetDatabaseStats retorna estatísticas da base de dados OUI
func GetDatabaseStats() (*DatabaseStats, error) {
	size, err := ouija.GetDatabaseInfo()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter informações da base de dados: %v", err)
	}

	return &DatabaseStats{
		TotalOUIs:  size,
		Source:     "Wireshark/IEEE Official Database",
		LastUpdate: "Auto-updated via OUIja",
	}, nil
}

// VendorDetailedInfo contém informações detalhadas sobre um vendor
type VendorDetailedInfo struct {
	MAC           string   `json:"mac"`
	Vendor        string   `json:"vendor"`
	OUI           string   `json:"oui"`
	DeviceType    string   `json:"device_type"`
	IsAmbiguous   bool     `json:"is_ambiguous"`
	PossibleTypes []string `json:"possible_types"`
}

// VendorStats contém estatísticas sobre um fabricante
type VendorStats struct {
	Vendor string `json:"vendor"`
	Count  int    `json:"count"`
}

// DatabaseStats contém estatísticas da base de dados
type DatabaseStats struct {
	TotalOUIs  int    `json:"total_ouis"`
	Source     string `json:"source"`
	LastUpdate string `json:"last_update"`
}
