package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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
