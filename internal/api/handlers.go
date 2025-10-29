package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/config"
	"github.com/lucasrafaldini/wavetrack/internal/deviceid"
	"github.com/lucasrafaldini/wavetrack/internal/models"
	"github.com/lucasrafaldini/wavetrack/internal/storage"
)

// Server gerencia a API web
type Server struct {
	storage *storage.Storage
	cfg     *config.Config
}

// NewServer cria uma nova instância do servidor API
func NewServer(storage *storage.Storage, cfg *config.Config) *Server {
	return &Server{
		storage: storage,
		cfg:     cfg,
	}
}

// DeviceResponse representa um dispositivo na resposta da API
type DeviceResponse struct {
	MACAddress     string     `json:"mac_address"`
	Type           string     `json:"type"`
	Vendor         string     `json:"vendor"`
	SignalStrength int        `json:"signal_strength"`
	Frequency      int        `json:"frequency"`
	Channel        int        `json:"channel"`
	FirstSeen      time.Time  `json:"first_seen"`
	LastSeen       time.Time  `json:"last_seen"`
	IsActive       bool       `json:"is_active"`
	IsAmbiguous    bool       `json:"is_ambiguous"`   // true se o fabricante faz múltiplos tipos
	PossibleTypes  []string   `json:"possible_types"` // tipos possíveis quando ambíguo
	EmployeeName   string     `json:"employee_name,omitempty"`
	Department     string     `json:"department,omitempty"`
	FirstSeenToday *time.Time `json:"first_seen_today,omitempty"`
	OnlineDuration int        `json:"online_duration_seconds"`
}

// AssociateDeviceRequest representa uma requisição para associar dispositivo a funcionário
type AssociateDeviceRequest struct {
	MACAddress       string `json:"mac_address"`
	OldMACAddress    string `json:"old_mac_address,omitempty"` // Para edição com mudança de MAC
	Name             string `json:"name"`
	Department       string `json:"department"`
	CustomDeviceType string `json:"custom_device_type,omitempty"`
	CustomVendor     string `json:"custom_vendor,omitempty"`
}

// DeviceVendorDetailRequest representa uma requisição para detalhes do vendor
type DeviceVendorDetailRequest struct {
	MACAddress string `json:"mac_address"`
}

// VendorSearchRequest representa uma requisição para buscar vendors
type VendorSearchRequest struct {
	Pattern string `json:"pattern"`
}

// SetupRoutes configura as rotas da API
func (s *Server) SetupRoutes() http.Handler {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/devices", s.handleDevices)
	mux.HandleFunc("/api/employees", s.handleEmployees)
	mux.HandleFunc("/api/employees/", s.handleEmployeeDelete) // DELETE específico
	mux.HandleFunc("/api/associate", s.handleAssociate)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/report/today", s.handleReportToday)
	mux.HandleFunc("/api/history/7days", s.handleHistory7Days)
	mux.HandleFunc("/api/vendor/details", s.GetDeviceVendorDetails)
	mux.HandleFunc("/api/vendor/search", s.SearchVendorsByPattern)
	mux.HandleFunc("/api/vendor/top", s.GetTopVendors)
	mux.HandleFunc("/api/vendor/stats", s.GetDatabaseStats)

	// Serve arquivos estáticos (interface web)
	mux.Handle("/", http.FileServer(http.Dir("web")))

	return s.enableCORS(mux)
}

// handleDevices retorna lista de dispositivos detectados
func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Busca todos os dispositivos do banco
	devices, err := s.storage.GetAllDevices()
	if err != nil {
		log.Printf("Erro ao buscar dispositivos: %v", err)
		http.Error(w, "Erro ao buscar dispositivos", http.StatusInternalServerError)
		return
	}

	// Converte para resposta
	response := make([]DeviceResponse, 0, len(devices))
	now := time.Now()

	// Usa o timeout configurado com piso mínimo de 20 minutos
	offlineThresholdMinutes := s.cfg.Presence.TimeoutMinutes
	if offlineThresholdMinutes < 20 {
		offlineThresholdMinutes = 20
	}
	offlineThreshold := time.Duration(offlineThresholdMinutes) * time.Minute
	for _, device := range devices {
		// Busca funcionário se existir
		employee, _ := s.storage.GetEmployeeByMAC(device.MACAddress)

		dr := DeviceResponse{
			MACAddress:     device.MACAddress,
			Type:           device.Type,
			Vendor:         device.Vendor,
			SignalStrength: device.SignalStrength,
			Frequency:      device.Frequency,
			Channel:        device.Channel,
			FirstSeen:      device.FirstSeen,
			LastSeen:       device.LastSeen,
			IsActive:       device.IsActive && now.Sub(device.LastSeen) < offlineThreshold,
			IsAmbiguous:    device.IsAmbiguous,
			PossibleTypes:  device.PossibleTypes,
		}

		if employee != nil {
			dr.EmployeeName = employee.Name
			dr.Department = employee.Department

			// Usa tipo/vendor customizado se disponível
			if employee.CustomDeviceType != "" {
				dr.Type = employee.CustomDeviceType
			}
			if employee.CustomVendor != "" {
				dr.Vendor = employee.CustomVendor
			}

			if ts, err := s.storage.GetFirstArrivalToday(device.MACAddress); err == nil {
				dr.FirstSeenToday = ts
			}
			if dur, err := s.storage.GetOnlineDurationToday(device.MACAddress, now); err == nil {
				dr.OnlineDuration = int(dur.Seconds())
			}
		}

		response = append(response, dr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleEmployees retorna lista de funcionários cadastrados
func (s *Server) handleEmployees(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	employees, err := s.storage.GetAllEmployees()
	if err != nil {
		log.Printf("Erro ao buscar funcionários: %v", err)
		http.Error(w, "Erro ao buscar funcionários", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(employees)
}

// handleAssociate associa um dispositivo a um funcionário
func (s *Server) handleAssociate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var req AssociateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}

	// Valida campos obrigatórios
	if req.MACAddress == "" || req.Name == "" {
		http.Error(w, "MAC address e nome são obrigatórios", http.StatusBadRequest)
		return
	}

	// Verifica se é uma edição (existe colaborador com o MAC antigo)
	var oldEmployee *models.Employee
	var err error

	if req.OldMACAddress != "" {
		// Está editando e possivelmente mudando o MAC
		oldEmployee, err = s.storage.GetEmployeeByMAC(req.OldMACAddress)
		if err != nil {
			log.Printf("Erro ao buscar colaborador antigo: %v", err)
		}
	} else {
		// Verifica se já existe colaborador com este MAC (edição sem mudar MAC)
		oldEmployee, err = s.storage.GetEmployeeByMAC(req.MACAddress)
		if err != nil {
			log.Printf("Erro ao buscar colaborador: %v", err)
		}
	}

	// Se está editando e o MAC mudou, atualiza todo o histórico
	if req.OldMACAddress != "" && req.OldMACAddress != req.MACAddress {
		log.Printf("→ MAC alterado de %s para %s - Atualizando histórico...", req.OldMACAddress, req.MACAddress)

		// Atualiza MAC em todos os eventos e devices históricos
		if err := s.storage.UpdateHistoryMAC(req.OldMACAddress, req.MACAddress); err != nil {
			log.Printf("Erro ao atualizar histórico de MAC: %v", err)
			http.Error(w, "Erro ao atualizar histórico", http.StatusInternalServerError)
			return
		}

		// Deleta o registro antigo do employee (o histórico já foi atualizado)
		if err := s.storage.DeleteEmployee(req.OldMACAddress); err != nil {
			log.Printf("Aviso ao deletar MAC antigo: %v", err)
		}

		log.Printf("✓ Histórico atualizado com sucesso")
	}

	// Se o nome mudou, atualiza em todos os eventos históricos
	if oldEmployee != nil && oldEmployee.Name != req.Name {
		log.Printf("→ Nome alterado de '%s' para '%s' - Atualizando histórico...", oldEmployee.Name, req.Name)
		if err := s.storage.UpdateHistoryEmployee(req.MACAddress, req.Name); err != nil {
			log.Printf("Erro ao atualizar nome no histórico: %v", err)
			// Não retorna erro, continua
		}
	}

	// Verifica se o dispositivo existe (apenas para novos cadastros ou quando muda MAC)
	device, err := s.storage.GetDevice(req.MACAddress)
	if err != nil {
		log.Printf("Erro ao buscar dispositivo: %v", err)
		http.Error(w, "Erro ao buscar dispositivo", http.StatusInternalServerError)
		return
	}

	// Se o dispositivo não existe, cria um registro básico
	if device == nil {
		log.Printf("⚠️  Dispositivo %s não encontrado, criando registro básico", req.MACAddress)
		newDevice := &models.Device{
			MACAddress: req.MACAddress,
			Type:       "unknown",
			Vendor:     "unknown",
			FirstSeen:  time.Now(),
			LastSeen:   time.Now(),
			IsActive:   false,
		}
		if err := s.storage.SaveDevice(newDevice); err != nil {
			log.Printf("Erro ao criar dispositivo: %v", err)
			http.Error(w, "Erro ao criar dispositivo", http.StatusInternalServerError)
			return
		}
	}

	// Cria ou atualiza funcionário
	employee := &models.Employee{
		MACAddress:       req.MACAddress,
		Name:             req.Name,
		Department:       req.Department,
		CustomDeviceType: req.CustomDeviceType,
		CustomVendor:     req.CustomVendor,
	}

	if err := s.storage.SaveEmployee(employee); err != nil {
		log.Printf("Erro ao salvar funcionário: %v", err)
		http.Error(w, "Erro ao cadastrar funcionário", http.StatusInternalServerError)
		return
	}

	log.Printf("✓ Funcionário cadastrado: %s (%s) - %s", req.Name, req.Department, req.MACAddress)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Funcionário cadastrado com sucesso",
	})
}

// handleEmployeeDelete deleta um funcionário
func (s *Server) handleEmployeeDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Extrai MAC address da URL: /api/employees/{mac}
	macAddress := r.URL.Path[len("/api/employees/"):]
	if macAddress == "" {
		http.Error(w, "MAC Address não informado", http.StatusBadRequest)
		return
	}

	// Verifica se funcionário existe
	employee, err := s.storage.GetEmployeeByMAC(macAddress)
	if err != nil || employee == nil {
		http.Error(w, "Funcionário não encontrado", http.StatusNotFound)
		return
	}

	// Deleta funcionário (histórico é preservado)
	if err := s.storage.DeleteEmployee(macAddress); err != nil {
		log.Printf("Erro ao deletar funcionário: %v", err)
		http.Error(w, "Erro ao deletar funcionário", http.StatusInternalServerError)
		return
	}

	log.Printf("✓ Funcionário deletado: %s (%s) - Histórico preservado para relatórios", employee.Name, macAddress)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Funcionário deletado com sucesso",
	})
}

// handleStats retorna estatísticas gerais
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	stats, err := s.storage.GetStats()
	if err != nil {
		log.Printf("Erro ao buscar estatísticas: %v", err)
		http.Error(w, "Erro ao buscar estatísticas", http.StatusInternalServerError)
		return
	}

	// Envia também o threshold de inatividade efetivo (min 20m)
	offlineThresholdMinutes := s.cfg.Presence.TimeoutMinutes
	if offlineThresholdMinutes < 20 {
		offlineThresholdMinutes = 20
	}
	stats["offline_threshold_minutes"] = offlineThresholdMinutes

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleEvents retorna eventos recentes
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Busca últimos 100 eventos
	events, err := s.storage.GetRecentEvents(100)
	if err != nil {
		log.Printf("Erro ao buscar eventos: %v", err)
		http.Error(w, "Erro ao buscar eventos", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// EmployeeReportItem representa um item do relatório diário de um funcionário
type EmployeeReportItem struct {
	Name              string     `json:"name"`
	Department        string     `json:"department"`
	MACAddress        string     `json:"mac_address"`
	FirstArrival      *time.Time `json:"first_arrival,omitempty"`
	LastDeparture     *time.Time `json:"last_departure,omitempty"`
	OnlineDuration    int        `json:"online_duration_seconds"`
	IsCurrentlyOnline bool       `json:"is_currently_online"`
}

// handleReportToday retorna relatório de presença do dia atual
func (s *Server) handleReportToday(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	employees, err := s.storage.GetAllEmployees()
	if err != nil {
		log.Printf("Erro ao buscar funcionários: %v", err)
		http.Error(w, "Erro ao buscar funcionários", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	offlineThresholdMinutes := s.cfg.Presence.TimeoutMinutes
	if offlineThresholdMinutes < 20 {
		offlineThresholdMinutes = 20
	}
	offlineThreshold := time.Duration(offlineThresholdMinutes) * time.Minute

	report := make([]EmployeeReportItem, 0, len(employees))

	for _, emp := range employees {
		item := EmployeeReportItem{
			Name:       emp.Name,
			Department: emp.Department,
			MACAddress: emp.MACAddress,
		}

		// Busca primeira chegada do dia
		if firstArr, err := s.storage.GetFirstArrivalToday(emp.MACAddress); err == nil {
			item.FirstArrival = firstArr
		}

		// Busca última saída do dia
		if lastDep, err := s.storage.GetLastDepartureToday(emp.MACAddress); err == nil {
			item.LastDeparture = lastDep
		}

		// Calcula duração online
		if dur, err := s.storage.GetOnlineDurationToday(emp.MACAddress, now); err == nil {
			item.OnlineDuration = int(dur.Seconds())
		}

		// Verifica se está online agora
		if device, err := s.storage.GetDevice(emp.MACAddress); err == nil && device != nil {
			item.IsCurrentlyOnline = device.IsActive && now.Sub(device.LastSeen) < offlineThreshold
		}

		report = append(report, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// HistoryDay representa um dia no histórico
type HistoryDay struct {
	Date           string            `json:"date"`
	TotalEmployees int               `json:"total_employees"`
	TotalDevices   int               `json:"total_devices"`
	TotalHours     float64           `json:"total_hours"`
	Employees      []HistoryEmployee `json:"employees"`
}

// HistoryEmployee representa um funcionário no histórico
type HistoryEmployee struct {
	Name               string `json:"name"`
	DeviceType         string `json:"device_type"`
	FirstArrival       string `json:"first_arrival"`
	LastDeparture      string `json:"last_departure"`
	DurationSeconds    int    `json:"duration_seconds"`
	PresencePercentage int    `json:"presence_percentage"`
}

// handleHistory7Days retorna histórico dos últimos 7 dias do banco de dados
func (s *Server) handleHistory7Days(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Busca histórico do banco de dados
	historyData, err := s.storage.GetHistory7Days()
	if err != nil {
		log.Printf("Erro ao buscar histórico: %v", err)
		http.Error(w, "Erro ao buscar histórico", http.StatusInternalServerError)
		return
	}

	// Converte para o formato da API
	history := make([]HistoryDay, 0, len(historyData))

	for _, day := range historyData {
		employees := make([]HistoryEmployee, 0, len(day.Employees))
		totalSeconds := 0

		for _, emp := range day.Employees {
			// Formata horários
			firstArrival := "—"
			if emp.FirstArrival != nil {
				firstArrival = emp.FirstArrival.Format("15:04")
			}

			lastDeparture := "—"
			if emp.LastDeparture != nil {
				lastDeparture = emp.LastDeparture.Format("15:04")
			}

			// Calcula percentual de presença (considerando 8h como 100%)
			presencePercent := 0
			if emp.DurationSeconds > 0 {
				presencePercent = (emp.DurationSeconds * 100) / (8 * 3600)
				// Não limita a 100% - mostra o percentual real (pode ser > 100%)
			}

			employees = append(employees, HistoryEmployee{
				Name:               emp.Name,
				DeviceType:         emp.DeviceType,
				FirstArrival:       firstArrival,
				LastDeparture:      lastDeparture,
				DurationSeconds:    emp.DurationSeconds,
				PresencePercentage: presencePercent,
			})

			totalSeconds += emp.DurationSeconds
		}

		history = append(history, HistoryDay{
			Date:           day.Date.Format("2006-01-02"),
			TotalEmployees: day.TotalEmployees,
			TotalDevices:   day.TotalDevices,
			TotalHours:     float64(totalSeconds) / 3600.0,
			Employees:      employees,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// GetDeviceVendorDetails retorna informações detalhadas do vendor via OUIja
func (s *Server) GetDeviceVendorDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeviceVendorDetailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.MACAddress == "" {
		http.Error(w, "MAC address is required", http.StatusBadRequest)
		return
	}

	// Utiliza a nova função que integra com OUIja
	details, err := deviceid.GetDetailedVendorInfo(req.MACAddress)
	if err != nil {
		log.Printf("Erro ao buscar detalhes do vendor para %s: %v", req.MACAddress, err)
		http.Error(w, "Vendor not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(details)
}

// SearchVendorsByPattern busca vendors por padrão usando OUIja
func (s *Server) SearchVendorsByPattern(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VendorSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Pattern == "" {
		http.Error(w, "Search pattern is required", http.StatusBadRequest)
		return
	}

	// Utiliza a nova função que integra com OUIja
	vendors, err := deviceid.SearchVendorsByPattern(req.Pattern)
	if err != nil {
		log.Printf("Erro ao buscar vendors com padrão %s: %v", req.Pattern, err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pattern": req.Pattern,
		"vendors": vendors,
		"count":   len(vendors),
	})
}

// GetTopVendors retorna os principais vendors por número de OUIs
func (s *Server) GetTopVendors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Por padrão retorna top 20, pode ser parametrizado futuramente
	limit := 20
	vendors := deviceid.GetTopVendors(limit)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"top_vendors": vendors,
		"limit":       limit,
		"count":       len(vendors),
	})
}

// GetDatabaseStats retorna estatísticas da base de dados OUI
func (s *Server) GetDatabaseStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := deviceid.GetDatabaseStats()
	if err != nil {
		log.Printf("Erro ao obter estatísticas da base: %v", err)
		http.Error(w, "Failed to get database stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// enableCORS adiciona headers CORS
func (s *Server) enableCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler.ServeHTTP(w, r)
	})
}
