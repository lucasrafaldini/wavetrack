package models

import "time"

// Device representa um dispositivo detectado na rede Wi-Fi
type Device struct {
	MACAddress     string    `json:"mac_address"`
	Type           string    `json:"type"` // laptop, smartphone, tablet, etc
	Vendor         string    `json:"vendor"`
	SignalStrength int       `json:"signal_strength"` // em dBm (0 = não disponível)
	Frequency      int       `json:"frequency"`       // Frequência em MHz (0 = não disponível)
	Channel        int       `json:"channel"`         // Canal Wi-Fi (0 = não disponível)
	FirstSeen      time.Time `json:"first_seen"`
	LastSeen       time.Time `json:"last_seen"`
	IsActive       bool      `json:"is_active"`      // true se visto recentemente
	IsAmbiguous    bool      `json:"is_ambiguous"`   // true se o fabricante faz múltiplos tipos
	PossibleTypes  []string  `json:"possible_types"` // tipos possíveis quando ambíguo
}

// Employee representa um funcionário cadastrado no sistema
type Employee struct {
	ID               int       `json:"id"`
	MACAddress       string    `json:"mac_address"` // MAC address principal do dispositivo
	Name             string    `json:"name"`
	Department       string    `json:"department"`
	CustomDeviceType string    `json:"custom_device_type,omitempty"` // Tipo customizado pelo usuário
	CustomVendor     string    `json:"custom_vendor,omitempty"`      // Fabricante customizado pelo usuário
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// PresenceEvent representa um evento de presença detectado
type PresenceEvent struct {
	Type           string    `json:"type"` // "arrival", "departure", "unknown_device"
	MACAddress     string    `json:"mac_address"`
	EmployeeName   string    `json:"employee_name,omitempty"`
	SignalStrength int       `json:"signal_strength"`
	Timestamp      time.Time `json:"timestamp"`
	Metadata       string    `json:"metadata,omitempty"` // JSON adicional
}

// RouterConfig armazena as configurações do roteador
type RouterConfig struct {
	Interface    string `json:"interface"`     // Interface de rede (ex: wlan0, en0)
	Channel      int    `json:"channel"`       // Canal Wi-Fi para monitorar
	ScanInterval int    `json:"scan_interval"` // Intervalo de scan em segundos
}
