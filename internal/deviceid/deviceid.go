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

// VendorCache armazena consultas em memória para evitar lookups repetidos
type VendorCache struct {
	mu    sync.RWMutex
	cache map[string]VendorInfo
}

var vendorCache = &VendorCache{
	cache: make(map[string]VendorInfo),
}

func init() {
	log.Println("Sistema de identificação inicializado com OUIja (base IEEE oficial)")
}

// lookupVendorOUIja consulta vendor usando a biblioteca OUIja
func lookupVendorOUIja(mac string) (VendorInfo, error) {
	vendor, err := ouija.GetVendor(mac)
	if err != nil {
		return VendorInfo{}, fmt.Errorf("vendor não encontrado via OUIja: %v", err)
	}

	deviceInfo := inferDeviceType(vendor)
	deviceInfo.Name = vendor

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
			Name:       company,
			DeviceType: "smartphone",
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
			Name:       company,
			DeviceType: "laptop",
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
			Name:       company,
			DeviceType: "router",
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
			Name:       company,
			DeviceType: "iot",
			IsAmbiguous: false,
		}
	}

	// FABRICANTES AMBÍGUOS (fazem múltiplos tipos)

	// Apple - principalmente laptops e smartphones
	if strings.Contains(company, "apple") {
		if strings.Contains(company, "iphone") || strings.Contains(company, "mobile") {
			return VendorInfo{
				Name:       company,
				DeviceType: "smartphone",
				IsAmbiguous: false,
			}
		}
		if strings.Contains(company, "ipad") {
			return VendorInfo{
				Name:       company,
				DeviceType: "tablet",
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
				Name:       company,
				DeviceType: "smartphone",
				IsAmbiguous: false,
			}
		}
		if strings.Contains(company, "computer") || strings.Contains(company, "playstation") {
			return VendorInfo{
				Name:       company,
				DeviceType: "console",
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
				Name:       company,
				DeviceType: "laptop",
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
			Name:       company,
			DeviceType: "console",
			IsAmbiguous: false,
		}
	}

	// Smart TVs e streaming
	if strings.Contains(company, "amazon") ||
		strings.Contains(company, "roku") ||
		strings.Contains(company, "chromecast") {
		return VendorInfo{
			Name:       company,
			DeviceType: "smarttv",
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
			Name:       company,
			DeviceType: "smartphone",
			IsAmbiguous: false,
		}
	}

	if strings.Contains(company, "computer") || strings.Contains(company, "laptop") {
		return VendorInfo{
			Name:       company,
			DeviceType: "laptop",
			IsAmbiguous: false,
		}
	}

	if strings.Contains(company, "network") || strings.Contains(company, "router") {
		return VendorInfo{
			Name:       company,
			DeviceType: "router",
			IsAmbiguous: false,
		}
	}

	if strings.Contains(company, "smart") || strings.Contains(company, "iot") {
		return VendorInfo{
			Name:       company,
			DeviceType: "iot",
			IsAmbiguous: false,
		}
	}

	// Completamente desconhecido
	return VendorInfo{
		Name:       company,
		DeviceType: "unknown",
		IsAmbiguous: false,
	}
}

// FormatDeviceInfo retorna uma string formatada para o dispositivo
func FormatDeviceInfo(vendor, deviceType string) string {
	return fmt.Sprintf("%s - %s", vendor, deviceType)
}

// FormatDeviceInfoWithTooltip retorna informação formatada incluindo tipos possíveis
func FormatDeviceInfoWithTooltip(info VendorInfo) string {
	if info.IsAmbiguous && len(info.PossibleTypes) > 0 {
		tooltip := strings.Join(info.PossibleTypes, ", ")
		return fmt.Sprintf("%s - %s (pode ser: %s)", info.Name, info.DeviceType, tooltip)
	}
	return fmt.Sprintf("%s - %s", info.Name, info.DeviceType)
}

// IdentifyDevice identifica o tipo e fabricante usando OUIja + cache
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

	// 1. Verifica cache (performance)
	vendorCache.mu.RLock()
	if info, exists := vendorCache.cache[oui]; exists {
		vendorCache.mu.RUnlock()
		return info
	}
	vendorCache.mu.RUnlock()

	// 2. Consulta OUIja (base IEEE oficial via Wireshark)
	if info, err := lookupVendorOUIja(mac); err == nil {
		vendorCache.mu.Lock()
		vendorCache.cache[oui] = info
		vendorCache.mu.Unlock()

		log.Printf("OUIja: %s [%s]", mac, FormatDeviceInfoWithTooltip(info))
		return info
	}

	// 3. Desconhecido - cacheia para evitar consultas repetidas
	unknown := VendorInfo{Name: "Unknown", DeviceType: "unknown", IsAmbiguous: false}
	vendorCache.mu.Lock()
	vendorCache.cache[oui] = unknown
	vendorCache.mu.Unlock()

	return unknown
}

// GetDetailedVendorInfo retorna informações detalhadas usando OUIja
// Inclui informações adicionais como MAC normalizado
func GetDetailedVendorInfo(mac string) (*VendorDetailedInfo, error) {
	result, err := ouija.LookupVendor(mac)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar informações detalhadas: %v", err)
	}

	deviceInfo := inferDeviceType(result.Vendor)

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
