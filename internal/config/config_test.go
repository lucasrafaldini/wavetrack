package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMissingFileUsesDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "absent.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":8080" || cfg.Presence.AwayAfter.Duration != 20*time.Minute || cfg.Privacy.RetentionDays != 365 {
		t.Fatalf("defaults = %+v", cfg)
	}
}

func TestFileOverridesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.yaml")
	os.WriteFile(path, []byte("capture:\n  interface: wlan1\npresence:\n  away_after: 45m\n"), 0o600)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Capture.Interface != "wlan1" || cfg.Presence.AwayAfter.Duration != 45*time.Minute || cfg.Listen != ":8080" {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestValidationExplainsEveryProblem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.yaml")
	os.WriteFile(path, []byte("listen: \"\"\npresence:\n  away_after: 10s\n  min_signal_dbm: 5\nprivacy:\n  retention_days: -1\n"), 0o600)
	_, err := Load(path)
	if err == nil {
		t.Fatal("invalid config accepted")
	}
	for _, want := range []string{"listen", "away_after", "min_signal_dbm", "retention_days"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error doesn't mention %s: %v", want, err)
		}
	}
}

func TestBadDurationIsReported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.yaml")
	os.WriteFile(path, []byte("presence:\n  away_after: twenty\n"), 0o600)
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "twenty") {
		t.Fatalf("err = %v", err)
	}
}
