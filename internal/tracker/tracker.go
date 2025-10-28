package tracker

import (
	"log"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/config"
	"github.com/lucasrafaldini/wavetrack/internal/logger"
	"github.com/lucasrafaldini/wavetrack/internal/models"
	"github.com/lucasrafaldini/wavetrack/internal/storage"
	"github.com/lucasrafaldini/wavetrack/internal/wifi"
)

// PresenceTracker gerencia o rastreamento de presença de funcionários
type PresenceTracker struct {
	config      *config.Config
	scanner     *wifi.Scanner
	logger      *logger.EventLogger
	storage     *storage.Storage
	lastSeen    map[string]time.Time // MAC -> LastSeen
	presenceMap map[string]bool      // MAC -> IsPresent
}

// NewPresenceTracker cria uma nova instância do tracker
func NewPresenceTracker(cfg *config.Config, scanner *wifi.Scanner, logger *logger.EventLogger, storage *storage.Storage) *PresenceTracker {
	tracker := &PresenceTracker{
		config:      cfg,
		scanner:     scanner,
		logger:      logger,
		storage:     storage,
		lastSeen:    make(map[string]time.Time),
		presenceMap: make(map[string]bool),
	}

	return tracker
}

// Start inicia o monitoramento de presença
func (t *PresenceTracker) Start() {
	log.Println("Iniciando rastreamento de presença...")

	// Carrega funcionários cadastrados
	employees, err := t.storage.GetAllEmployees()
	if err != nil {
		log.Printf("Erro ao carregar funcionários: %v", err)
	} else {
		log.Printf("Monitorando %d funcionários cadastrados", len(employees))
	}

	// Processa dispositivos detectados
	go t.processDevices()

	// Verifica timeouts periodicamente
	go t.checkTimeouts()
}

// processDevices processa novos dispositivos detectados
func (t *PresenceTracker) processDevices() {
	for device := range t.scanner.GetDevices() {
		t.handleDevice(device)
	}
}

// handleDevice processa um dispositivo detectado
func (t *PresenceTracker) handleDevice(device *models.Device) {
	// Salva/atualiza dispositivo no storage
	if err := t.storage.SaveDevice(device); err != nil {
		log.Printf("Erro ao salvar dispositivo %s: %v", device.MACAddress, err)
	}

	// Verifica se o sinal é forte o suficiente
	if device.SignalStrength < t.config.Presence.SignalThreshold && device.SignalStrength != 0 {
		return
	}

	// Busca funcionário no storage usando o MAC
	employee, err := t.storage.GetEmployeeByMAC(device.MACAddress)
	if err != nil {
		log.Printf("Erro ao buscar funcionário: %v", err)
		return
	}

	if employee != nil {
		// Checa se é uma nova chegada
		wasPresent := t.presenceMap[device.MACAddress]

		if !wasPresent {
			// Funcionário chegou
			event := &models.PresenceEvent{
				Timestamp:      time.Now(),
				Type:           "arrival",
				EmployeeName:   employee.Name,
				MACAddress:     device.MACAddress,
				SignalStrength: device.SignalStrength,
			}
			t.logger.LogEvent(event)

			// Salva evento no banco
			if err := t.storage.SaveEvent(event); err != nil {
				log.Printf("Erro ao salvar evento: %v", err)
			}

			t.presenceMap[device.MACAddress] = true
			log.Printf("✓ %s chegou (Departamento: %s)", employee.Name, employee.Department)
		}

		// Atualiza último visto
		t.lastSeen[device.MACAddress] = time.Now()
	} else {
		// Dispositivo desconhecido - registra evento
		event := &models.PresenceEvent{
			Timestamp:      time.Now(),
			Type:           "unknown_device",
			MACAddress:     device.MACAddress,
			SignalStrength: device.SignalStrength,
		}
		t.logger.LogEvent(event)

		// Salva evento no banco
		if err := t.storage.SaveEvent(event); err != nil {
			log.Printf("Erro ao salvar evento: %v", err)
		}
	}
}

// checkTimeouts verifica se algum funcionário saiu
func (t *PresenceTracker) checkTimeouts() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		// Usa o timeout configurado com piso mínimo de 20 minutos
		timeoutMinutes := t.config.Presence.TimeoutMinutes
		if timeoutMinutes < 20 {
			timeoutMinutes = 20
		}
		timeout := time.Duration(timeoutMinutes) * time.Minute

		// Busca todos os funcionários do storage
		employees, err := t.storage.GetAllEmployees()
		if err != nil {
			log.Printf("Erro ao buscar funcionários: %v", err)
			continue
		}

		for _, employee := range employees {
			mac := employee.MACAddress
			lastSeen, exists := t.lastSeen[mac]

			if !exists {
				continue
			}

			// Verifica se passou do timeout
			if now.Sub(lastSeen) > timeout && t.presenceMap[mac] {
				// Funcionário saiu
				event := &models.PresenceEvent{
					Timestamp:      now,
					Type:           "departure",
					EmployeeName:   employee.Name,
					MACAddress:     mac,
					SignalStrength: 0,
				}
				t.logger.LogEvent(event)

				// Salva evento no banco
				if err := t.storage.SaveEvent(event); err != nil {
					log.Printf("Erro ao salvar evento: %v", err)
				}

				// Atualiza status do dispositivo
				if err := t.storage.UpdateDeviceStatus(mac, false); err != nil {
					log.Printf("Erro ao atualizar status: %v", err)
				}

				t.presenceMap[mac] = false
				log.Printf("✗ %s saiu", employee.Name)
			}
		}
	}
}

// GetPresentEmployees retorna lista de funcionários presentes
func (t *PresenceTracker) GetPresentEmployees() []string {
	var present []string
	employees, err := t.storage.GetAllEmployees()
	if err != nil {
		log.Printf("Erro ao buscar funcionários: %v", err)
		return present
	}

	for _, employee := range employees {
		if t.presenceMap[employee.MACAddress] {
			present = append(present, employee.Name)
		}
	}
	return present
}
