package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/models"
)

// EventLogger gerencia o registro de eventos de presença
type EventLogger struct {
	logDir   string
	logFile  *os.File
	filePath string
}

// NewEventLogger cria uma nova instância do logger
func NewEventLogger(logDir string) (*EventLogger, error) {
	// Cria o diretório de logs se não existir
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("erro ao criar diretório de logs: %v", err)
	}

	logger := &EventLogger{
		logDir: logDir,
	}

	if err := logger.openLogFile(); err != nil {
		return nil, err
	}

	return logger, nil
}

// openLogFile abre ou cria o arquivo de log do dia
func (l *EventLogger) openLogFile() error {
	// Nome do arquivo baseado na data atual
	filename := fmt.Sprintf("presence_%s.log", time.Now().Format("2006-01-02"))
	l.filePath = filepath.Join(l.logDir, filename)

	var err error
	l.logFile, err = os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("erro ao abrir arquivo de log: %v", err)
	}

	log.Printf("Arquivo de log aberto: %s", l.filePath)
	return nil
}

// LogEvent registra um evento de presença
func (l *EventLogger) LogEvent(event *models.PresenceEvent) error {
	// Verifica se precisa rotacionar o arquivo (novo dia)
	expectedFilename := fmt.Sprintf("presence_%s.log", time.Now().Format("2006-01-02"))
	currentFilename := filepath.Base(l.filePath)

	if expectedFilename != currentFilename {
		_ = l.logFile.Close()
		if err := l.openLogFile(); err != nil {
			return err
		}
	}

	// Serializa o evento em JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("erro ao serializar evento: %v", err)
	}

	// Escreve no arquivo
	if _, err := l.logFile.Write(append(eventJSON, '\n')); err != nil {
		return fmt.Errorf("erro ao escrever no log: %v", err)
	}

	// Log também no console
	log.Printf("Evento registrado: %s | %s | MAC: %s | Sinal: %d dBm",
		event.Timestamp.Format("15:04:05"),
		event.Type,
		event.MACAddress,
		event.SignalStrength)

	return nil
}

// LogDeviceDetection registra a detecção de um dispositivo
func (l *EventLogger) LogDeviceDetection(device *models.Device, eventType string) error {
	event := &models.PresenceEvent{
		Timestamp:      time.Now(),
		Type:           eventType,
		MACAddress:     device.MACAddress,
		SignalStrength: device.SignalStrength,
	}

	return l.LogEvent(event)
}

// Close fecha o arquivo de log
func (l *EventLogger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// GetLogPath retorna o caminho do arquivo de log atual
func (l *EventLogger) GetLogPath() string {
	return l.filePath
}
