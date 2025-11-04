package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/config"
	"github.com/lucasrafaldini/wavetrack/internal/models"
	"github.com/lucasrafaldini/wavetrack/internal/storage"
	qrcode "github.com/skip2/go-qrcode"
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
	mux.HandleFunc("/api/cleanup/inactive", s.handleCleanupInactive)

	// Registration via QR Code
	mux.HandleFunc("/api/register/token", s.handleGenerateQRCode)
	mux.HandleFunc("/api/register/validate/", s.handleValidateToken)
	mux.HandleFunc("/api/register/submit", s.handleRegistrationSubmit)
	mux.HandleFunc("/register/", s.handleRegistrationPage)

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

// handleCleanupInactive remove dispositivos inativos sem cadastro de colaborador
func (s *Server) handleCleanupInactive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	log.Println("🧹 Limpeza manual iniciada via API...")

	// 1. Conta dispositivos não cadastrados antes da limpeza
	totalBefore, inactiveBefore, err := s.storage.GetUnregisteredDevicesCount()
	if err != nil {
		log.Printf("❌ Erro ao contar dispositivos: %v", err)
		http.Error(w, "Erro ao contar dispositivos", http.StatusInternalServerError)
		return
	}

	// 2. Remove dispositivos não cadastrados inativos IMEDIATAMENTE (sem espera)
	removedDevices, err := s.storage.CleanupUnregisteredDevices(0)
	if err != nil {
		log.Printf("❌ Erro na limpeza de dispositivos: %v", err)
		http.Error(w, "Erro na limpeza de dispositivos", http.StatusInternalServerError)
		return
	}

	// 3. Remove eventos antigos (mantém últimos 30 dias)
	removedEvents, err := s.storage.CleanupOldEvents(30)
	if err != nil {
		log.Printf("❌ Erro na limpeza de eventos: %v", err)
		http.Error(w, "Erro na limpeza de eventos", http.StatusInternalServerError)
		return
	}

	// 4. Relatório final
	totalAfter, inactiveAfter, err := s.storage.GetUnregisteredDevicesCount()
	if err != nil {
		log.Printf("❌ Erro ao contar dispositivos finais: %v", err)
		http.Error(w, "Erro ao contar dispositivos", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Limpeza manual concluída:")
	log.Printf("   📱 Dispositivos removidos: %d", removedDevices)
	log.Printf("   📋 Eventos removidos: %d", removedEvents)
	log.Printf("   📊 Dispositivos não cadastrados: %d → %d", totalBefore, totalAfter)
	log.Printf("   😴 Dispositivos inativos: %d → %d", inactiveBefore, inactiveAfter)

	// Retorna resposta
	response := map[string]interface{}{
		"success":         true,
		"removed_devices": removedDevices,
		"removed_events":  removedEvents,
		"total_before":    totalBefore,
		"total_after":     totalAfter,
		"inactive_before": inactiveBefore,
		"inactive_after":  inactiveAfter,
		"message":         fmt.Sprintf("Limpeza concluída: %d dispositivos e %d eventos removidos", removedDevices, removedEvents),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// === REGISTRATION VIA QR CODE ===

// handleGenerateQRCode gera um token e retorna o QR code
func (s *Server) handleGenerateQRCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Gera token único (UUID simples)
	token := fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())

	// Token expira em 24 horas
	expiresAt := time.Now().Add(24 * time.Hour)

	// Salva no banco
	regToken := &models.RegistrationToken{
		Token:     token,
		ExpiresAt: expiresAt,
		Used:      false,
		CreatedAt: time.Now(),
	}

	if err := s.storage.CreateRegistrationToken(regToken); err != nil {
		log.Printf("Erro ao criar token: %v", err)
		http.Error(w, "Erro ao gerar token", http.StatusInternalServerError)
		return
	}

	// Obtém IP local do servidor
	serverIP := s.getServerIP(r)

	// Monta URL de registro local
	registerURL := fmt.Sprintf("http://%s/register/%s", serverIP, token)

	// Gera QR code
	qrCode, err := qrcode.Encode(registerURL, qrcode.Medium, 256)
	if err != nil {
		log.Printf("Erro ao gerar QR code: %v", err)
		http.Error(w, "Erro ao gerar QR code", http.StatusInternalServerError)
		return
	}

	// Converte para base64
	qrBase64 := base64.StdEncoding.EncodeToString(qrCode)

	log.Printf("✓ QR Code gerado: %s (expira em 24h)", registerURL)

	// Retorna resposta
	response := map[string]interface{}{
		"token":      token,
		"url":        registerURL,
		"qr_code":    fmt.Sprintf("data:image/png;base64,%s", qrBase64),
		"expires_at": expiresAt.Format(time.RFC3339),
		"expires_in": "24 horas",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleValidateToken valida se um token ainda é válido
func (s *Server) handleValidateToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Extrai token da URL: /api/register/validate/{token}
	token := r.URL.Path[len("/api/register/validate/"):]
	if token == "" {
		http.Error(w, "Token não informado", http.StatusBadRequest)
		return
	}

	// Busca token no banco
	regToken, err := s.storage.GetRegistrationToken(token)
	if err != nil {
		log.Printf("Erro ao buscar token: %v", err)
		http.Error(w, "Erro ao validar token", http.StatusInternalServerError)
		return
	}

	if regToken == nil {
		http.Error(w, "Token inválido", http.StatusNotFound)
		return
	}

	// Verifica se já foi usado
	if regToken.Used {
		http.Error(w, "Token já foi utilizado", http.StatusGone)
		return
	}

	// Verifica se expirou
	if time.Now().After(regToken.ExpiresAt) {
		http.Error(w, "Token expirado", http.StatusGone)
		return
	}

	// Token válido
	response := map[string]interface{}{
		"valid":      true,
		"expires_at": regToken.ExpiresAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRegistrationSubmit processa o cadastro via QR code
func (s *Server) handleRegistrationSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var submission models.RegistrationSubmission
	if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}

	// Valida campos obrigatórios
	if submission.Token == "" || submission.Name == "" {
		http.Error(w, "Token e nome são obrigatórios", http.StatusBadRequest)
		return
	}

	// Busca e valida token
	regToken, err := s.storage.GetRegistrationToken(submission.Token)
	if err != nil {
		log.Printf("Erro ao buscar token: %v", err)
		http.Error(w, "Erro ao validar token", http.StatusInternalServerError)
		return
	}

	if regToken == nil {
		http.Error(w, "Token inválido", http.StatusNotFound)
		return
	}

	if regToken.Used {
		http.Error(w, "Token já foi utilizado", http.StatusGone)
		return
	}

	if time.Now().After(regToken.ExpiresAt) {
		http.Error(w, "Token expirado", http.StatusGone)
		return
	}

	// Captura MAC address do dispositivo que fez a requisição
	macAddress := s.getMACFromRequest(r)
	if macAddress == "" {
		// Se não conseguir detectar, tenta pegar do IP
		macAddress = s.getMACFromIP(r.RemoteAddr)
	}

	if macAddress == "" {
		log.Printf("⚠️  Não foi possível detectar MAC address para %s", submission.Name)
		http.Error(w, "Não foi possível detectar seu dispositivo. Tente conectar ao Wi-Fi primeiro.", http.StatusBadRequest)
		return
	}

	log.Printf("→ Detectado MAC: %s para %s", macAddress, submission.Name)

	// Verifica se o dispositivo já existe, senão cria
	device, err := s.storage.GetDevice(macAddress)
	if err != nil {
		log.Printf("Erro ao buscar dispositivo: %v", err)
	}

	if device == nil {
		// Cria dispositivo básico
		device = &models.Device{
			MACAddress: macAddress,
			Type:       "smartphone", // Assume smartphone por padrão
			Vendor:     "unknown",
			FirstSeen:  time.Now(),
			LastSeen:   time.Now(),
			IsActive:   true,
		}
		if err := s.storage.SaveDevice(device); err != nil {
			log.Printf("Erro ao criar dispositivo: %v", err)
		}
	}

	// Cria colaborador
	employee := &models.Employee{
		MACAddress:       macAddress,
		Name:             submission.Name,
		Department:       submission.Department,
		CustomDeviceType: submission.CustomDeviceType,
		CustomVendor:     submission.CustomVendor,
	}

	if err := s.storage.SaveEmployee(employee); err != nil {
		log.Printf("Erro ao salvar colaborador: %v", err)
		http.Error(w, "Erro ao cadastrar colaborador", http.StatusInternalServerError)
		return
	}

	// Marca token como usado
	if err := s.storage.MarkTokenAsUsed(submission.Token); err != nil {
		log.Printf("Erro ao marcar token como usado: %v", err)
	}

	log.Printf("✓ Colaborador cadastrado via QR Code: %s (%s) - MAC: %s",
		submission.Name, submission.Department, macAddress)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     "Cadastro realizado com sucesso!",
		"name":        submission.Name,
		"mac_address": macAddress,
	})
}

// handleRegistrationPage serve a página HTML de registro mobile
func (s *Server) handleRegistrationPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "web/register.html")
}

// getServerIP obtém o IP local do servidor da requisição
func (s *Server) getServerIP(r *http.Request) string {
	// Primeiro tenta pegar o IP real da interface de rede
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return fmt.Sprintf("%s:8080", ipnet.IP.String())
				}
			}
		}
	}

	// Se não conseguiu, tenta usar o Host da requisição
	host := r.Host
	if host != "" && !strings.HasPrefix(host, "localhost") && !strings.HasPrefix(host, "127.0.0.1") {
		return host
	}

	// Último fallback
	return "localhost:8080"
}

// getMACFromRequest tenta extrair MAC do header X-MAC-Address (se enviado pelo cliente)
func (s *Server) getMACFromRequest(r *http.Request) string {
	return r.Header.Get("X-MAC-Address")
}

// getMACFromIP tenta descobrir MAC a partir do IP (consulta ARP)
func (s *Server) getMACFromIP(remoteAddr string) string {
	// Remove porta do endereço
	ip := remoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	// Remove colchetes de IPv6
	ip = strings.Trim(ip, "[]")

	// Ignora localhost
	if ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return ""
	}

	log.Printf("→ Tentando detectar MAC do IP: %s", ip)

	// Tenta consultar a tabela ARP do sistema (macOS/Linux)
	// Executa: arp -n <ip>
	cmd := fmt.Sprintf("arp -n %s", ip)
	output, err := execCommand(cmd)
	if err == nil {
		// Parse da saída do ARP
		// Formato macOS: ? (192.168.1.100) at aa:bb:cc:dd:ee:ff on en0 ifscope [ethernet]
		// Formato Linux: 192.168.1.100 ether aa:bb:cc:dd:ee:ff C eth0
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			// Procura por padrão MAC (XX:XX:XX:XX:XX:XX)
			fields := strings.Fields(line)
			for _, field := range fields {
				if len(field) == 17 && strings.Count(field, ":") == 5 {
					// Valida se é um MAC válido
					if isValidMAC(field) {
						log.Printf("✓ MAC detectado via ARP: %s", field)
						return field
					}
				}
			}
		}
	}

	log.Printf("⚠️  Não foi possível detectar MAC via ARP para IP %s", ip)

	// Fallback: consulta dispositivos recentes (menos confiável)
	devices, err := s.storage.GetAllDevices()
	if err != nil {
		log.Printf("Erro ao buscar devices: %v", err)
		return ""
	}

	// Retorna o device mais recentemente visto e ativo
	var mostRecent *models.Device
	for _, d := range devices {
		if d.IsActive {
			if mostRecent == nil || d.LastSeen.After(mostRecent.LastSeen) {
				mostRecent = &d
			}
		}
	}

	if mostRecent != nil {
		log.Printf("→ Usando fallback: MAC do dispositivo ativo mais recente: %s", mostRecent.MACAddress)
		return mostRecent.MACAddress
	}

	return ""
}

// execCommand executa um comando shell e retorna a saída
func execCommand(cmd string) (string, error) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "", fmt.Errorf("comando vazio")
	}

	out, err := exec.Command(parts[0], parts[1:]...).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// isValidMAC verifica se uma string é um MAC address válido
func isValidMAC(mac string) bool {
	parts := strings.Split(mac, ":")
	if len(parts) != 6 {
		return false
	}
	for _, part := range parts {
		if len(part) != 2 {
			return false
		}
		// Verifica se é hexadecimal
		for _, c := range part {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}
