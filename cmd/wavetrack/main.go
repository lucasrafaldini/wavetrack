package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/api"
	"github.com/lucasrafaldini/wavetrack/internal/config"
	"github.com/lucasrafaldini/wavetrack/internal/logger"
	"github.com/lucasrafaldini/wavetrack/internal/storage"
	"github.com/lucasrafaldini/wavetrack/internal/tracker"
	"github.com/lucasrafaldini/wavetrack/internal/wifi"
)

func main() {
	// Flags de linha de comando
	configPath := flag.String("config", "config.yaml", "Caminho para o arquivo de configuração")
	port := flag.Int("port", 8080, "Porta do servidor web")
	flag.Parse()

	log.Println("=== WaveTrack - Sistema de Monitoramento de Presença ===")

	// Carrega configuração
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
	}
	log.Printf("Configuração carregada: interface=%s, intervalo=%ds",
		cfg.Network.Interface, cfg.Network.ScanInterval)

	// Inicializa o storage (SQLite)
	dataStorage, err := storage.NewStorage("./data")
	if err != nil {
		log.Fatalf("Erro ao inicializar storage: %v", err)
	}
	defer dataStorage.Close()
	log.Println("Sistema de armazenamento iniciado (SQLite)")

	// Inicializa o logger de eventos
	eventLogger, err := logger.NewEventLogger(cfg.Logging.LogDir)
	if err != nil {
		log.Fatalf("Erro ao inicializar logger: %v", err)
	}
	defer eventLogger.Close()
	log.Printf("Sistema de logs iniciado: %s", eventLogger.GetLogPath())

	// Inicializa o scanner Wi-Fi
	scanner := wifi.NewScanner(cfg.Network.Interface)
	if err := scanner.Start(); err != nil {
		log.Fatalf("Erro ao iniciar scanner: %v", err)
	}
	defer scanner.Stop()

	// Inicializa o tracker de presença
	presenceTracker := tracker.NewPresenceTracker(cfg, scanner, eventLogger, dataStorage)
	presenceTracker.Start()

	// Inicia scheduler de limpeza automática
	go startCleanupScheduler(dataStorage)

	// Inicializa o servidor web
	apiServer := api.NewServer(dataStorage, cfg)
	go func() {
		addr := fmt.Sprintf("0.0.0.0:%d", *port)
		log.Printf("🌐 Servidor web iniciado em http://0.0.0.0%s", addr[7:])
		log.Println("   Acesse o dashboard no navegador!")
		log.Println("   📱 Dispositivos na rede podem acessar via IP local")
		if err := http.ListenAndServe(addr, apiServer.SetupRoutes()); err != nil {
			log.Fatalf("Erro ao iniciar servidor web: %v", err)
		}
	}()

	log.Println("Sistema iniciado! Pressione Ctrl+C para parar...")

	// Aguarda sinal de interrupção
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("\nEncerrando WaveTrack...")
	log.Println("Sistema finalizado com sucesso!")
}

// startCleanupScheduler inicia o agendador de limpeza automática
func startCleanupScheduler(storage *storage.Storage) {
	// Executa limpeza a cada 24 horas
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Executa limpeza inicial após 1 hora de funcionamento
	time.Sleep(1 * time.Hour)
	runCleanup(storage)

	// Loop principal do scheduler
	for range ticker.C {
		runCleanup(storage)
	}
}

// runCleanup executa a limpeza de dispositivos e eventos antigos
func runCleanup(storage *storage.Storage) {
	log.Println("🧹 Iniciando limpeza automática...")

	// 1. Conta dispositivos não cadastrados antes da limpeza
	totalBefore, inactiveBefore, err := storage.GetUnregisteredDevicesCount()
	if err != nil {
		log.Printf("❌ Erro ao contar dispositivos: %v", err)
		return
	}

	// 2. Remove dispositivos não cadastrados inativos há mais de 7 dias
	removedDevices, err := storage.CleanupUnregisteredDevices(7)
	if err != nil {
		log.Printf("❌ Erro na limpeza de dispositivos: %v", err)
		return
	}

	// 3. Remove eventos antigos (mantém últimos 30 dias)
	removedEvents, err := storage.CleanupOldEvents(30)
	if err != nil {
		log.Printf("❌ Erro na limpeza de eventos: %v", err)
		return
	}

	// 4. Relatório final
	totalAfter, inactiveAfter, err := storage.GetUnregisteredDevicesCount()
	if err != nil {
		log.Printf("❌ Erro ao contar dispositivos finais: %v", err)
		return
	}

	log.Printf("✅ Limpeza concluída:")
	log.Printf("   📱 Dispositivos removidos: %d", removedDevices)
	log.Printf("   📋 Eventos removidos: %d", removedEvents)
	log.Printf("   📊 Dispositivos não cadastrados: %d → %d", totalBefore, totalAfter)
	log.Printf("   😴 Dispositivos inativos: %d → %d", inactiveBefore, inactiveAfter)

	if removedDevices > 0 || removedEvents > 0 {
		log.Printf("🎯 Base de dados otimizada - removidos %d dispositivos e %d eventos antigos",
			removedDevices, removedEvents)
	}
}
