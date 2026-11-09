// Package config loads WaveTrack's YAML configuration and fills in defaults.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Listen    string   `yaml:"listen"`
	PublicURL string   `yaml:"public_url"`
	DataDir   string   `yaml:"data_dir"`
	TLS       TLS      `yaml:"tls"`
	Capture   Capture  `yaml:"capture"`
	Presence  Presence `yaml:"presence"`
	Privacy   Privacy  `yaml:"privacy"`
}

// TLS serves the dashboard over HTTPS, so the admin token and session cookie never cross
// the office network in clear text. Both files or neither.
type TLS struct {
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

func (t TLS) Enabled() bool { return t.CertFile != "" && t.KeyFile != "" }

type Capture struct {
	// Interface to listen on. Empty means pick the active Wi-Fi interface.
	Interface string `yaml:"interface"`
	// MonitorMode captures 802.11 frames (with signal strength) on adapters that support it.
	MonitorMode bool `yaml:"monitor_mode"`
}

type Presence struct {
	// AwayAfter is how long a person goes unseen before they count as gone.
	AwayAfter Duration `yaml:"away_after"`
	// MinSignal ignores weaker sightings, in dBm. Only monitor mode reports signal.
	MinSignal int `yaml:"min_signal_dbm"`
}

type Privacy struct {
	// RetentionDays deletes visits older than this. Zero keeps them forever.
	RetentionDays int `yaml:"retention_days"`
	// AuditRetentionDays deletes audit log entries older than this. Zero keeps them forever.
	AuditRetentionDays int `yaml:"audit_retention_days"`
}

// Duration accepts Go duration strings ("20m") in YAML.
type Duration struct{ time.Duration }

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	parsed, err := time.ParseDuration(node.Value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", node.Value, err)
	}
	d.Duration = parsed
	return nil
}

func (d Duration) MarshalYAML() (any, error) { return d.String(), nil }

func Default() Config {
	return Config{
		Listen:   ":8080",
		DataDir:  "./data",
		Presence: Presence{AwayAfter: Duration{20 * time.Minute}, MinSignal: -85},
		Privacy:  Privacy{RetentionDays: 365, AuditRetentionDays: 730},
	}
}

// Load reads path over the defaults. A missing file is not an error.
func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, cfg.Validate()
	}
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	var problems []error
	if c.Listen == "" {
		problems = append(problems, errors.New("listen must not be empty"))
	}
	if c.DataDir == "" {
		problems = append(problems, errors.New("data_dir must not be empty"))
	}
	if c.Presence.AwayAfter.Duration < time.Minute {
		problems = append(problems, errors.New("presence.away_after must be at least 1m"))
	}
	if c.Presence.MinSignal > 0 || c.Presence.MinSignal < -120 {
		problems = append(problems, errors.New("presence.min_signal_dbm must be between -120 and 0"))
	}
	if c.Privacy.RetentionDays < 0 {
		problems = append(problems, errors.New("privacy.retention_days must not be negative"))
	}
	if c.Privacy.AuditRetentionDays < 0 {
		problems = append(problems, errors.New("privacy.audit_retention_days must not be negative"))
	}
	if (c.TLS.CertFile == "") != (c.TLS.KeyFile == "") {
		problems = append(problems, errors.New("tls needs both cert_file and key_file"))
	}
	return errors.Join(problems...)
}
