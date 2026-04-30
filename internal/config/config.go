package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config representa a configuração completa da aplicação
type Config struct {
	Network  NetworkConfig  `yaml:"network"`
	Logging  LoggingConfig  `yaml:"logging"`
	Presence PresenceConfig `yaml:"presence"`
}

// NetworkConfig contém configurações de rede
type NetworkConfig struct {
	Interface    string `yaml:"interface"`
	Channel      int    `yaml:"channel"`
	ScanInterval int    `yaml:"scan_interval"`
}

// LoggingConfig contém configurações de logging
type LoggingConfig struct {
	LogDir   string `yaml:"log_dir"`
	LogLevel string `yaml:"log_level"`
}

// PresenceConfig contém configurações de detecção de presença
type PresenceConfig struct {
	TimeoutMinutes  int `yaml:"timeout_minutes"`
	SignalThreshold int `yaml:"signal_threshold"`
}

// LoadConfig carrega a configuração do arquivo YAML
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo de configuração: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("erro ao parsear configuração: %v", err)
	}

	return &config, nil
}
