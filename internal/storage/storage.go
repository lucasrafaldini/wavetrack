package storage

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/lucasrafaldini/wavetrack/internal/models"
)

//go:embed schema.sql
var schemaSQL string

// Storage gerencia o armazenamento persistente usando SQLite
type Storage struct {
	db *sql.DB
}

// NewStorage cria uma nova instância do storage com SQLite
func NewStorage(dataDir string) (*Storage, error) {
	dbPath := filepath.Join(dataDir, "wavetrack.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir banco de dados: %v", err)
	}

	// Configura pool de conexões
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	storage := &Storage{db: db}

	// Inicializa schema
	if err := storage.initSchema(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("erro ao criar schema: %v (erro ao fechar: %v)", err, closeErr)
		}
		return nil, err
	}

	return storage, nil
}

// initSchema cria as tabelas se não existirem
func (s *Storage) initSchema() error {
	_, err := s.db.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("erro ao criar schema: %v", err)
	}
	return nil
}

// Close fecha a conexão com o banco
func (s *Storage) Close() error {
	return s.db.Close()
}

// GetFirstArrivalToday retorna o primeiro horário de chegada (arrival) do dia para um MAC
func (s *Storage) GetFirstArrivalToday(macAddress string) (*time.Time, error) {
	// Define o intervalo do dia local [início do dia, início do próximo dia)
	// Usamos funções do SQLite para considerar o timezone local.
	query := `
		SELECT MIN(timestamp)
		FROM events
		WHERE mac_address = ?
		 AND event_type = 'arrival'
		 AND timestamp >= datetime(date('now','localtime'))
		 AND timestamp < datetime(date('now','localtime'), '+1 day')
	`

	var ts sql.NullString
	if err := s.db.QueryRow(query, macAddress).Scan(&ts); err != nil {
		return nil, err
	}
	if !ts.Valid || ts.String == "" {
		return nil, nil
	}
	// Parse com suporte a offset, se presente
	t, err := time.Parse(time.RFC3339Nano, ts.String)
	if err != nil {
		// Tenta formato padrão SQLite sem offset
		// Ex.: 2025-10-28 16:28:30.029489
		layouts := []string{
			"2006-01-02 15:04:05.999999-07:00",
			"2006-01-02 15:04:05.999999",
			"2006-01-02 15:04:05",
		}
		for _, layout := range layouts {
			if tt, e := time.Parse(layout, ts.String); e == nil {
				return &tt, nil
			}
		}
		return nil, err
	}
	return &t, nil
}

// GetOnlineDurationToday retorna a soma de períodos online (arrival->departure) no dia atual para o MAC.
// Se estiver online agora (sem departure após o último arrival), soma até "now".
func (s *Storage) GetOnlineDurationToday(macAddress string, now time.Time) (time.Duration, error) {
	query := `
		SELECT event_type, timestamp
		FROM events
		WHERE mac_address = ?
		 AND timestamp >= datetime(date('now','localtime'))
		 AND timestamp < datetime(date('now','localtime'), '+1 day')
		ORDER BY timestamp ASC
	`

	rows, err := s.db.Query(query, macAddress)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var (
		total time.Duration
		openSess *time.Time
	)

	parseTS := func(ts string) (time.Time, error) {
		// Tenta vários formatos comuns do SQLite
		layouts := []string{
			time.RFC3339Nano,
			"2006-01-02 15:04:05.999999-07:00",
			"2006-01-02 15:04:05.999999",
			"2006-01-02 15:04:05",
		}
		for _, layout := range layouts {
			if t, e := time.Parse(layout, ts); e == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("formato de timestamp desconhecido: %s", ts)
	}

	for rows.Next() {
		var et, ts string
		if err := rows.Scan(&et, &ts); err != nil {
			return 0, err
		}
		t, err := parseTS(ts)
		if err != nil {
			return 0, err
		}

		switch et {
		case "arrival":
			if openSess == nil { // inicia sessão
				tt := t
				openSess = &tt
			}
		case "departure":
			if openSess != nil && t.After(*openSess) {
				total += t.Sub(*openSess)
				openSess = nil
			}
		}
	}

	// Se há sessão aberta até "now"
	if openSess != nil && now.After(*openSess) {
		total += now.Sub(*openSess)
	}

	return total, rows.Err()
}

// === DEVICES ===

// SaveDevice salva ou atualiza um dispositivo
func (s *Storage) SaveDevice(device *models.Device) error {
	// Converte PossibleTypes para JSON
	var possibleTypesJSON string
	if len(device.PossibleTypes) > 0 {
		bytes, _ := json.Marshal(device.PossibleTypes)
		possibleTypesJSON = string(bytes)
	}

	query := `
		INSERT INTO devices (mac_address, vendor, type, first_seen, last_seen, signal_strength, frequency, channel, is_active, is_ambiguous, possible_types)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(mac_address) DO UPDATE SET
			vendor = excluded.vendor,
			type = excluded.type,
			last_seen = excluded.last_seen,
			signal_strength = excluded.signal_strength,
			frequency = excluded.frequency,
			channel = excluded.channel,
			is_active = excluded.is_active,
			is_ambiguous = excluded.is_ambiguous,
			possible_types = excluded.possible_types
	`

	_, err := s.db.Exec(query,
		device.MACAddress,
		device.Vendor,
		device.Type,
		device.FirstSeen,
		device.LastSeen,
		device.SignalStrength,
		device.Frequency,
		device.Channel,
		device.IsActive,
		device.IsAmbiguous,
		possibleTypesJSON,
	)

	return err
}

// GetDevice retorna um dispositivo pelo MAC
func (s *Storage) GetDevice(macAddress string) (*models.Device, error) {
	query := `
		SELECT mac_address, vendor, type, first_seen, last_seen, signal_strength, frequency, channel, is_active
		FROM devices
		WHERE mac_address = ?
	`

	var device models.Device
	err := s.db.QueryRow(query, macAddress).Scan(
		&device.MACAddress,
		&device.Vendor,
		&device.Type,
		&device.FirstSeen,
		&device.LastSeen,
		&device.SignalStrength,
		&device.Frequency,
		&device.Channel,
		&device.IsActive,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &device, nil
}

// GetAllDevices retorna todos os dispositivos
func (s *Storage) GetAllDevices() ([]models.Device, error) {
	query := `
		SELECT mac_address, vendor, type, first_seen, last_seen, signal_strength, frequency, channel, is_active, is_ambiguous, possible_types
		FROM devices
		ORDER BY last_seen DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var device models.Device
		var possibleTypesJSON sql.NullString

		err := rows.Scan(
			&device.MACAddress,
			&device.Vendor,
			&device.Type,
			&device.FirstSeen,
			&device.LastSeen,
			&device.SignalStrength,
			&device.Frequency,
			&device.Channel,
			&device.IsActive,
			&device.IsAmbiguous,
			&possibleTypesJSON,
		)
		if err != nil {
			return nil, err
		}

		// Deserializa PossibleTypes do JSON
		if possibleTypesJSON.Valid && possibleTypesJSON.String != "" {
			if err := json.Unmarshal([]byte(possibleTypesJSON.String), &device.PossibleTypes); err != nil {
				log.Printf("Aviso: erro ao deserializar possible_types: %v", err)
			}
		}

		devices = append(devices, device)
	}

	return devices, rows.Err()
}

// UpdateDeviceStatus atualiza o status de ativo/inativo
func (s *Storage) UpdateDeviceStatus(macAddress string, isActive bool) error {
	query := `UPDATE devices SET is_active = ? WHERE mac_address = ?`
	_, err := s.db.Exec(query, isActive, macAddress)
	return err
}

// === EMPLOYEES ===

// SaveEmployee salva ou atualiza um funcionário
func (s *Storage) SaveEmployee(employee *models.Employee) error {
	query := `
		INSERT INTO employees (mac_address, name, department, custom_device_type, custom_vendor, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(mac_address) DO UPDATE SET
			name = excluded.name,
			department = excluded.department,
			custom_device_type = excluded.custom_device_type,
			custom_vendor = excluded.custom_vendor,
			updated_at = excluded.updated_at
	`

	now := time.Now()
	if employee.CreatedAt.IsZero() {
		employee.CreatedAt = now
	}
	employee.UpdatedAt = now

	_, err := s.db.Exec(query,
		employee.MACAddress,
		employee.Name,
		employee.Department,
		employee.CustomDeviceType,
		employee.CustomVendor,
		employee.CreatedAt,
		employee.UpdatedAt,
	)

	return err
}

// GetEmployeeByMAC busca um funcionário pelo MAC address
func (s *Storage) GetEmployeeByMAC(macAddress string) (*models.Employee, error) {
	query := `
		SELECT id, mac_address, name, department,
		 COALESCE(custom_device_type, ''), COALESCE(custom_vendor, ''),
		 created_at, updated_at
		FROM employees
		WHERE mac_address = ?
	`

	var employee models.Employee
	err := s.db.QueryRow(query, macAddress).Scan(
		&employee.ID,
		&employee.MACAddress,
		&employee.Name,
		&employee.Department,
		&employee.CustomDeviceType,
		&employee.CustomVendor,
		&employee.CreatedAt,
		&employee.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &employee, nil
}

// GetAllEmployees retorna todos os funcionários
func (s *Storage) GetAllEmployees() ([]models.Employee, error) {
	query := `
		SELECT id, mac_address, name, department,
		 COALESCE(custom_device_type, ''), COALESCE(custom_vendor, ''),
		 created_at, updated_at
		FROM employees
		ORDER BY name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []models.Employee
	for rows.Next() {
		var employee models.Employee
		err := rows.Scan(
			&employee.ID,
			&employee.MACAddress,
			&employee.Name,
			&employee.Department,
			&employee.CustomDeviceType,
			&employee.CustomVendor,
			&employee.CreatedAt,
			&employee.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		employees = append(employees, employee)
	}

	return employees, rows.Err()
}

// DeleteEmployee remove um funcionário (mas preserva seu histórico de eventos)
func (s *Storage) DeleteEmployee(macAddress string) error {
	query := `DELETE FROM employees WHERE mac_address = ?`
	_, err := s.db.Exec(query, macAddress)
	// NOTA: Os eventos (arrivals/departures) são preservados para relatórios
	// Apenas o vínculo na tabela employees é removido
	return err
}

// UpdateHistoryMAC atualiza o MAC address em todos os eventos históricos
func (s *Storage) UpdateHistoryMAC(oldMAC, newMAC string) error {
	// Atualiza eventos primeiro
	query := `UPDATE events SET mac_address = ? WHERE mac_address = ?`
	_, err := s.db.Exec(query, newMAC, oldMAC)
	if err != nil {
		return fmt.Errorf("erro ao atualizar eventos: %v", err)
	}

	// Verifica se o novo MAC já existe em devices
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM devices WHERE mac_address = ?)`
	err = s.db.QueryRow(checkQuery, newMAC).Scan(&exists)
	if err != nil {
		return fmt.Errorf("erro ao verificar device: %v", err)
	}

	if exists {
		// Se já existe, apenas deleta o antigo
		deleteQuery := `DELETE FROM devices WHERE mac_address = ?`
		_, err = s.db.Exec(deleteQuery, oldMAC)
		if err != nil {
			return fmt.Errorf("erro ao deletar device antigo: %v", err)
		}
	} else {
		// Se não existe, faz o UPDATE do MAC
		updateQuery := `UPDATE devices SET mac_address = ? WHERE mac_address = ?`
		_, err = s.db.Exec(updateQuery, newMAC, oldMAC)
		if err != nil {
			return fmt.Errorf("erro ao atualizar devices: %v", err)
		}
	}

	return nil
}

// UpdateHistoryEmployee atualiza nome do colaborador em todos os eventos históricos
func (s *Storage) UpdateHistoryEmployee(macAddress, newName string) error {
	query := `UPDATE events SET employee_name = ? WHERE mac_address = ?`
	_, err := s.db.Exec(query, newName, macAddress)
	if err != nil {
		return fmt.Errorf("erro ao atualizar nome nos eventos: %v", err)
	}
	return nil
}

// === EVENTS ===

// SaveEvent salva um evento de presença
func (s *Storage) SaveEvent(event *models.PresenceEvent) error {
	query := `
		INSERT INTO events (event_type, mac_address, employee_name, signal_strength, timestamp, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		event.Type,
		event.MACAddress,
		event.EmployeeName,
		event.SignalStrength,
		event.Timestamp,
		event.Metadata,
	)

	return err
}

// GetEvents retorna eventos filtrados por data
func (s *Storage) GetEvents(startDate, endDate time.Time, eventType string) ([]models.PresenceEvent, error) {
	query := `
		SELECT id, event_type, mac_address, employee_name, signal_strength, timestamp, metadata
		FROM events
		WHERE timestamp BETWEEN ? AND ?
	`

	args := []interface{}{startDate, endDate}

	if eventType != "" {
		query += " AND event_type = ?"
		args = append(args, eventType)
	}

	query += " ORDER BY timestamp DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.PresenceEvent
	for rows.Next() {
		var event models.PresenceEvent
		var id int
		err := rows.Scan(
			&id,
			&event.Type,
			&event.MACAddress,
			&event.EmployeeName,
			&event.SignalStrength,
			&event.Timestamp,
			&event.Metadata,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// GetRecentEvents retorna os últimos N eventos
func (s *Storage) GetRecentEvents(limit int) ([]models.PresenceEvent, error) {
	query := `
		SELECT id, event_type, mac_address, employee_name, signal_strength, timestamp, metadata
		FROM events
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.PresenceEvent
	for rows.Next() {
		var event models.PresenceEvent
		var id int
		err := rows.Scan(
			&id,
			&event.Type,
			&event.MACAddress,
			&event.EmployeeName,
			&event.SignalStrength,
			&event.Timestamp,
			&event.Metadata,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// === STATISTICS ===

// GetStats retorna estatísticas gerais
func (s *Storage) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total de dispositivos
	var totalDevices int
	err := s.db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&totalDevices)
	if err != nil {
		return nil, err
	}
	stats["total_devices"] = totalDevices

	// Dispositivos ativos
	var activeDevices int
	err = s.db.QueryRow("SELECT COUNT(*) FROM devices WHERE is_active = 1").Scan(&activeDevices)
	if err != nil {
		return nil, err
	}
	stats["active_devices"] = activeDevices

	// Total de funcionários
	var totalEmployees int
	err = s.db.QueryRow("SELECT COUNT(*) FROM employees").Scan(&totalEmployees)
	if err != nil {
		return nil, err
	}
	stats["total_employees"] = totalEmployees

	// Funcionários presentes (employees com dispositivos ativos)
	var presentEmployees int
	err = s.db.QueryRow(`
		SELECT COUNT(DISTINCT e.id)
		FROM employees e
		INNER JOIN devices d ON e.mac_address = d.mac_address
		WHERE d.is_active = 1
	`).Scan(&presentEmployees)
	if err != nil {
		return nil, err
	}
	stats["present_employees"] = presentEmployees

	// Eventos de hoje
	today := time.Now().Truncate(24 * time.Hour)
	var eventsToday int
	err = s.db.QueryRow("SELECT COUNT(*) FROM events WHERE timestamp >= ?", today).Scan(&eventsToday)
	if err != nil {
		return nil, err
	}
	stats["events_today"] = eventsToday

	return stats, nil
}

// GetDeviceDetails retorna device com dados do employee (usando a view)
func (s *Storage) GetDeviceDetails(macAddress string) (map[string]interface{}, error) {
	query := `
		SELECT mac_address, vendor, type, first_seen, last_seen, signal_strength, is_active,
		 employee_name, employee_department
		FROM device_details
		WHERE mac_address = ?
	`

	var (
		mac, vendor, devType string
		firstSeen, lastSeen time.Time
		signal int
		isActive bool
		employeeName, employeeDept sql.NullString
	)

	err := s.db.QueryRow(query, macAddress).Scan(
		&mac, &vendor, &devType, &firstSeen, &lastSeen, &signal, &isActive,
		&employeeName, &employeeDept,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	details := map[string]interface{}{
		"mac_address": mac,
		"vendor": vendor,
		"type": devType,
		"first_seen": firstSeen,
		"last_seen": lastSeen,
		"signal_strength": signal,
		"is_active": isActive,
	}

	if employeeName.Valid {
		details["employee_name"] = employeeName.String
	}
	if employeeDept.Valid {
		details["employee_department"] = employeeDept.String
	}

	return details, nil
}

// GetLastDepartureToday retorna o último horário de saída (departure) do dia para um MAC
func (s *Storage) GetLastDepartureToday(macAddress string) (*time.Time, error) {
	query := `
		SELECT MAX(timestamp)
		FROM events
		WHERE mac_address = ?
		 AND event_type = 'departure'
		 AND timestamp >= datetime(date('now','localtime'))
		 AND timestamp < datetime(date('now','localtime'), '+1 day')
	`

	var ts sql.NullString
	if err := s.db.QueryRow(query, macAddress).Scan(&ts); err != nil {
		return nil, err
	}
	if !ts.Valid || ts.String == "" {
		return nil, nil
	}

	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999-07:00",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, e := time.Parse(layout, ts.String); e == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("formato de timestamp desconhecido: %s", ts.String)
}

// HistoryDayData representa dados de um dia no histórico
type HistoryDayData struct {
	Date time.Time
	TotalEmployees int
	TotalDevices int
	Employees []HistoryEmployeeData
}

// HistoryEmployeeData representa dados de um funcionário em um dia
type HistoryEmployeeData struct {
	MACAddress string
	Name string
	DeviceType string
	Vendor string
	FirstArrival *time.Time
	LastDeparture *time.Time
	DurationSeconds int
}

// GetHistory7Days busca o histórico dos últimos 7 dias do banco de dados
func (s *Storage) GetHistory7Days() ([]HistoryDayData, error) {
	history := make([]HistoryDayData, 0, 7)

	for dayOffset := 0; dayOffset < 7; dayOffset++ {
		date := time.Now().AddDate(0, 0, -dayOffset)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
		endOfDay := startOfDay.AddDate(0, 0, 1)

		// Busca funcionários que tiveram eventos neste dia
		query := `
			SELECT DISTINCT e.mac_address, emp.name, emp.custom_device_type, emp.custom_vendor
			FROM events e
			INNER JOIN employees emp ON e.mac_address = emp.mac_address
			WHERE e.timestamp >= ? AND e.timestamp < ?
			ORDER BY emp.name
		`

		rows, err := s.db.Query(query, startOfDay.Format("2006-01-02 15:04:05"), endOfDay.Format("2006-01-02 15:04:05"))
		if err != nil {
			return nil, fmt.Errorf("erro ao buscar funcionários do dia %s: %v", date.Format("2006-01-02"), err)
		}

		employees := make([]HistoryEmployeeData, 0)

		for rows.Next() {
			var empData HistoryEmployeeData
			var deviceType, vendor sql.NullString

			if err := rows.Scan(&empData.MACAddress, &empData.Name, &deviceType, &vendor); err != nil {
				rows.Close()
				return nil, err
			}

			if deviceType.Valid {
				empData.DeviceType = deviceType.String
			}
			if vendor.Valid {
				empData.Vendor = vendor.String
			}

			// Busca primeiro arrival do dia
			firstArrival, _ := s.getFirstArrivalForDate(empData.MACAddress, startOfDay, endOfDay)
			empData.FirstArrival = firstArrival

			// Busca último departure do dia
			lastDeparture, _ := s.getLastDepartureForDate(empData.MACAddress, startOfDay, endOfDay)
			empData.LastDeparture = lastDeparture

			// Calcula duração total do dia
			duration, _ := s.getOnlineDurationForDate(empData.MACAddress, startOfDay, endOfDay)
			empData.DurationSeconds = int(duration.Seconds())

			employees = append(employees, empData)
		}
		rows.Close()

		dayData := HistoryDayData{
			Date: date,
			TotalEmployees: len(employees),
			TotalDevices: len(employees),
			Employees: employees,
		}

		history = append(history, dayData)
	}

	return history, nil
}

// getFirstArrivalForDate retorna o primeiro arrival de um MAC em uma data específica
func (s *Storage) getFirstArrivalForDate(macAddress string, startOfDay, endOfDay time.Time) (*time.Time, error) {
	query := `
		SELECT MIN(timestamp)
		FROM events
		WHERE mac_address = ?
		 AND event_type = 'arrival'
		 AND timestamp >= ?
		 AND timestamp < ?
	`

	var ts sql.NullString
	if err := s.db.QueryRow(query, macAddress, startOfDay.Format("2006-01-02 15:04:05"), endOfDay.Format("2006-01-02 15:04:05")).Scan(&ts); err != nil {
		return nil, err
	}
	if !ts.Valid || ts.String == "" {
		return nil, nil
	}

	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999-07:00",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, e := time.Parse(layout, ts.String); e == nil {
			return &t, nil
		}
	}
	return nil, nil
}

// getLastDepartureForDate retorna o último departure de um MAC em uma data específica
func (s *Storage) getLastDepartureForDate(macAddress string, startOfDay, endOfDay time.Time) (*time.Time, error) {
	query := `
		SELECT MAX(timestamp)
		FROM events
		WHERE mac_address = ?
		 AND event_type = 'departure'
		 AND timestamp >= ?
		 AND timestamp < ?
	`

	var ts sql.NullString
	if err := s.db.QueryRow(query, macAddress, startOfDay.Format("2006-01-02 15:04:05"), endOfDay.Format("2006-01-02 15:04:05")).Scan(&ts); err != nil {
		return nil, err
	}
	if !ts.Valid || ts.String == "" {
		return nil, nil
	}

	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999-07:00",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, e := time.Parse(layout, ts.String); e == nil {
			return &t, nil
		}
	}
	return nil, nil
}

// getOnlineDurationForDate calcula a duração online de um MAC em uma data específica
func (s *Storage) getOnlineDurationForDate(macAddress string, startOfDay, endOfDay time.Time) (time.Duration, error) {
	// Busca todos os eventos do dia ordenados por timestamp
	query := `
		SELECT event_type, timestamp
		FROM events
		WHERE mac_address = ?
		 AND timestamp >= ?
		 AND timestamp < ?
		ORDER BY timestamp ASC
	`

	rows, err := s.db.Query(query, macAddress, startOfDay.Format("2006-01-02 15:04:05"), endOfDay.Format("2006-01-02 15:04:05"))
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var totalDuration time.Duration
	var lastArrival *time.Time

	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999-07:00",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
	}

	for rows.Next() {
		var eventType, tsStr string
		if err := rows.Scan(&eventType, &tsStr); err != nil {
			continue
		}

		var ts time.Time
		parsed := false
		for _, layout := range layouts {
			if t, e := time.Parse(layout, tsStr); e == nil {
				ts = t
				parsed = true
				break
			}
		}
		if !parsed {
			continue
		}

		if eventType == "arrival" {
			lastArrival = &ts
		} else if eventType == "departure" && lastArrival != nil {
			duration := ts.Sub(*lastArrival)
			if duration > 0 {
				totalDuration += duration
			}
			lastArrival = nil
		}
	}

	return totalDuration, nil
}

// === REGISTRATION TOKENS ===

// CreateRegistrationToken cria um novo token de registro
func (s *Storage) CreateRegistrationToken(token *models.RegistrationToken) error {
	query := `
		INSERT INTO registration_tokens (token, expires_at, used, created_at)
		VALUES (?, ?, ?, ?)
	`
	_, err := s.db.Exec(query, token.Token, token.ExpiresAt, token.Used, token.CreatedAt)
	return err
}

// GetRegistrationToken busca um token pelo seu valor
func (s *Storage) GetRegistrationToken(token string) (*models.RegistrationToken, error) {
	query := `
		SELECT token, expires_at, used, created_at, used_at
		FROM registration_tokens
		WHERE token = ?
	`

	var rt models.RegistrationToken
	var usedAt sql.NullTime

	err := s.db.QueryRow(query, token).Scan(
		&rt.Token,
		&rt.ExpiresAt,
		&rt.Used,
		&rt.CreatedAt,
		&usedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if usedAt.Valid {
		rt.UsedAt = &usedAt.Time
	}

	return &rt, nil
}

// MarkTokenAsUsed marca um token como usado
func (s *Storage) MarkTokenAsUsed(token string) error {
	query := `
		UPDATE registration_tokens
		SET used = 1, used_at = CURRENT_TIMESTAMP
		WHERE token = ?
	`
	_, err := s.db.Exec(query, token)
	return err
}

// CleanExpiredTokens remove tokens expirados (rotina de limpeza)
func (s *Storage) CleanExpiredTokens() error {
	query := `DELETE FROM registration_tokens WHERE expires_at < CURRENT_TIMESTAMP`
	_, err := s.db.Exec(query)
	return err
}

// === CLEANUP FUNCTIONS ===

// CleanupUnregisteredDevices remove dispositivos não cadastrados inativos há mais de X dias
func (s *Storage) CleanupUnregisteredDevices(daysOld int) (int, error) {
	// PASSO 1: Atualiza o campo is_active no banco baseado no timeout (20 minutos padrão)
	updateQuery := `
		UPDATE devices
		SET is_active = 0
		WHERE datetime(last_seen) < datetime('now', '-20 minutes')
	`
	if _, err := s.db.Exec(updateQuery); err != nil {
		log.Printf(" Erro ao atualizar status: %v", err)
	} else {
		log.Printf(" Status de dispositivos atualizado baseado em last_seen")
	}

	// PASSO 2: Remove dispositivos que:
	// 1. Não estão na tabela employees (não cadastrados)
	// 2. Estão inativos (is_active = 0)
	// 3. Se daysOld > 0, considera também o tempo desde last_seen
	var query string
	var result sql.Result
	var err error

	// Debug: verifica quantos dispositivos inativos sem cadastro existem
	var countBefore int
	debugQuery := `
		SELECT COUNT(*) FROM devices
		WHERE mac_address NOT IN (SELECT mac_address FROM employees)
		 AND is_active = 0
	`
	if err := s.db.QueryRow(debugQuery).Scan(&countBefore); err != nil {
		log.Printf(" Erro ao contar dispositivos: %v", err)
	}
	log.Printf(" DEBUG: Dispositivos inativos sem cadastro antes da limpeza: %d", countBefore)

	if daysOld == 0 {
		// Remove TODOS os dispositivos inativos sem cadastro, independente do tempo
		query = `
		DELETE FROM devices
		WHERE mac_address NOT IN (SELECT mac_address FROM employees)
		 AND is_active = 0
		`
		log.Printf(" Executando limpeza: removendo TODOS os inativos sem cadastro")
		result, err = s.db.Exec(query)
	} else {
		// Remove dispositivos inativos há mais de X dias
		query = `
		DELETE FROM devices
		WHERE mac_address NOT IN (SELECT mac_address FROM employees)
		 AND is_active = 0
		 AND last_seen < datetime('now', '-' || ? || ' days')
		`
		log.Printf(" Executando limpeza: removendo inativos há mais de %d dias", daysOld)
		result, err = s.db.Exec(query, daysOld)
	}

	if err != nil {
		return 0, fmt.Errorf("erro ao limpar dispositivos não cadastrados: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("erro ao obter dispositivos removidos: %v", err)
	}

	log.Printf(" Dispositivos removidos: %d", int(affected))
	return int(affected), nil
}

// CleanupOldEvents remove eventos antigos (mantém apenas os últimos X dias)
func (s *Storage) CleanupOldEvents(daysToKeep int) (int, error) {
	// Remove apenas eventos de unknown_device antigos
	// Mantém eventos de funcionários cadastrados
	query := `
	DELETE FROM events
	WHERE event_type = 'unknown_device'
	 AND timestamp < datetime('now', '-' || ? || ' days')
	 AND mac_address NOT IN (SELECT mac_address FROM employees)
	`

	result, err := s.db.Exec(query, daysToKeep)
	if err != nil {
		return 0, fmt.Errorf("erro ao limpar eventos antigos: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("erro ao obter eventos removidos: %v", err)
	}

	return int(affected), nil
}

// GetUnregisteredDevicesCount retorna a quantidade de dispositivos não cadastrados
func (s *Storage) GetUnregisteredDevicesCount() (total, inactive int, err error) {
	// Primeiro, atualiza o campo is_active baseado no timeout
	updateQuery := `
		UPDATE devices
		SET is_active = 0
		WHERE datetime(last_seen) < datetime('now', '-20 minutes')
	`
	if _, execErr := s.db.Exec(updateQuery); execErr != nil {
		log.Printf(" Erro ao atualizar status: %v", execErr)
	}

	// Total de dispositivos não cadastrados
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM devices
		WHERE mac_address NOT IN (SELECT mac_address FROM employees)
	`).Scan(&total)
	if err != nil {
		return 0, 0, fmt.Errorf("erro ao contar dispositivos não cadastrados: %v", err)
	}

	// Dispositivos não cadastrados inativos (baseado no campo is_active do banco)
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM devices
		WHERE mac_address NOT IN (SELECT mac_address FROM employees)
		 AND is_active = 0
	`).Scan(&inactive)
	if err != nil {
		return 0, 0, fmt.Errorf("erro ao contar dispositivos inativos: %v", err)
	}

	return total, inactive, nil
}
