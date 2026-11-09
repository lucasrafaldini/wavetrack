// WaveTrack shows who is in the office by noticing their phones on the Wi-Fi.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gopacket/gopacket/pcap"

	"github.com/lucasrafaldini/wavetrack/internal/capture"
	"github.com/lucasrafaldini/wavetrack/internal/config"
	"github.com/lucasrafaldini/wavetrack/internal/demo"
	"github.com/lucasrafaldini/wavetrack/internal/importer"
	"github.com/lucasrafaldini/wavetrack/internal/netinfo"
	"github.com/lucasrafaldini/wavetrack/internal/oui"
	"github.com/lucasrafaldini/wavetrack/internal/presence"
	"github.com/lucasrafaldini/wavetrack/internal/server"
	"github.com/lucasrafaldini/wavetrack/internal/store"
	"github.com/lucasrafaldini/wavetrack/web"
)

var version = "3.1.0-dev"

const usage = `WaveTrack %s: see who's in, from the phones on your Wi-Fi.

Usage:
  wavetrack [serve] [flags]     run the tracker and dashboard
  wavetrack serve --demo        try it with a simulated office, no setup needed
  wavetrack interfaces          list network interfaces that can be captured
  wavetrack doctor              check capture permissions and configuration
  wavetrack token reset         issue a new admin access token
  wavetrack import-v2 <db>      import people and history from WaveTrack 2.x
  wavetrack version

Flags for serve:
`

func main() {
	args := os.Args[1:]
	command := "serve"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command, args = args[0], args[1:]
	}

	var err error
	switch command {
	case "serve":
		err = serve(args)
	case "interfaces":
		err = listInterfaces(len(args) > 0 && args[0] == "--all")
	case "doctor":
		err = doctor(args)
	case "token":
		err = token(args)
	case "import-v2":
		err = importV2(args)
	case "version", "--version":
		fmt.Println("wavetrack", version)
	case "help", "-h", "--help":
		fmt.Printf(usage, version)
		serveFlags(new(serveOptions)).PrintDefaults()
	default:
		err = fmt.Errorf("unknown command %q; run `wavetrack help`", command)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "wavetrack:", err)
		os.Exit(1)
	}
}

type serveOptions struct {
	config, iface, listen, data string
	demo, monitor               bool
}

func serveFlags(o *serveOptions) *flag.FlagSet {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.StringVar(&o.config, "config", "config.yaml", "configuration file")
	fs.StringVar(&o.iface, "interface", "", "network interface to capture (overrides config)")
	fs.StringVar(&o.listen, "listen", "", "address for the dashboard, e.g. :8080 (overrides config)")
	fs.StringVar(&o.data, "data", "", "data directory (overrides config)")
	fs.BoolVar(&o.demo, "demo", false, "simulate an office instead of capturing")
	fs.BoolVar(&o.monitor, "monitor", false, "capture 802.11 frames in monitor mode (overrides config)")
	return fs
}

func loadConfig(o serveOptions) (config.Config, error) {
	cfg, err := config.Load(o.config)
	if err != nil {
		return cfg, err
	}
	if o.iface != "" {
		cfg.Capture.Interface = o.iface
	}
	if o.listen != "" {
		cfg.Listen = o.listen
	}
	if o.data != "" {
		cfg.DataDir = o.data
	}
	if o.monitor {
		cfg.Capture.MonitorMode = true
	}
	return cfg, cfg.Validate()
}

func serve(args []string) error {
	var o serveOptions
	if err := serveFlags(&o).Parse(args); err != nil {
		return err
	}
	cfg, err := loadConfig(o)
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	clock := time.Now
	if o.demo {
		if o.data == "" {
			dir, err := os.MkdirTemp("", "wavetrack-demo-")
			if err != nil {
				return err
			}
			defer os.RemoveAll(dir)
			cfg.DataDir = dir
		}
		clock = demo.Clock(time.Now)
	}

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return err
	}
	defer st.Close()

	var ident presence.Identifier = oui.Default()
	var source capture.Source
	if o.demo {
		ident = demo.Identifier{Fallback: ident}
		if people, _ := st.People(true); len(people) == 0 {
			if err := demo.Seed(st, clock(), 14); err != nil {
				return fmt.Errorf("seeding demo: %w", err)
			}
		}
		source = demo.NewSource(clock)
	} else {
		candidates, _ := netinfo.Interfaces()
		iface, err := netinfo.Choose(cfg.Capture.Interface, candidates)
		if err != nil {
			return err
		}
		live, err := capture.OpenLive(iface, cfg.Capture.MonitorMode)
		if err != nil {
			return err
		}
		source = live
	}
	defer source.Close()

	tracker := presence.New(st, ident, presence.Options{
		AwayAfter: cfg.Presence.AwayAfter.Duration,
		MinSignal: cfg.Presence.MinSignal,
		Logger:    logger,
		Clock:     clock,
	})
	if err := tracker.Start(clock()); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go tracker.Run(ctx, source)
	go retention(ctx, st, cfg.Privacy, clock, logger)

	_, port, _ := net.SplitHostPort(cfg.Listen)
	if port == "" {
		port = "8080"
	}
	desc := source.Describe()
	srv := server.New(server.Options{
		Store: st, Tracker: tracker, Ident: ident, Capture: desc, Assets: web.FS,
		Clock: clock, AwayAfter: cfg.Presence.AwayAfter.Duration,
		LookupMAC:  netinfo.MACForIP,
		InviteBase: inviteBase(cfg.PublicURL, scheme(cfg), desc.Interface, port),
		Version:    version, Logger: logger,
	})

	listener, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		return fmt.Errorf("can't listen on %s: %w", cfg.Listen, err)
	}
	// Every phase of a request is bounded, so slow or stalled clients can't hold
	// connections open. The write timeout leaves room for a year-long CSV export.
	httpServer := &http.Server{
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    64 << 10,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}
	go func() {
		var err error
		if cfg.TLS.Enabled() {
			err = httpServer.ServeTLS(listener, cfg.TLS.CertFile, cfg.TLS.KeyFile)
		} else {
			err = httpServer.Serve(listener)
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("dashboard stopped", "err", err)
			stop()
		}
	}()

	announce(st, desc, scheme(cfg), port, o.demo, logger)

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	tracker.Flush()
	return nil
}

func scheme(cfg config.Config) string {
	if cfg.TLS.Enabled() {
		return "https"
	}
	return "http"
}

func announce(st *store.Store, desc capture.Description, scheme, port string, isDemo bool, logger *slog.Logger) {
	what := "capturing on " + desc.Interface + " (" + desc.Mode + " mode)"
	if isDemo {
		what = "running a simulated office"
	}
	url := scheme + "://localhost:" + port
	fmt.Fprintf(os.Stderr, "\n  WaveTrack %s · %s\n  Dashboard: %s\n", version, what, url)

	token, err := server.EnsureAdminToken(st)
	switch {
	case err != nil:
		logger.Error("admin token", "err", err)
	case token != "":
		fmt.Fprintf(os.Stderr, "\n  Your admin access token (shown once, keep it safe):\n    %s\n  Or open %s/#token=%s to sign in directly.\n\n", token, url, token)
	default:
		fmt.Fprintf(os.Stderr, "  Sign in with your admin token. Lost it? Run `wavetrack token reset`.\n\n")
	}
}

// inviteBase builds the address phones use to open invite links. A browser on the
// WaveTrack machine itself says "localhost", which a phone can't reach, so use the LAN address.
func inviteBase(publicURL, scheme, iface, port string) func(*http.Request) string {
	return func(r *http.Request) string {
		if publicURL != "" {
			return strings.TrimRight(publicURL, "/")
		}
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return scheme + "://" + r.Host
		}
		if ip := netinfo.IPv4(iface); ip != "" {
			return scheme + "://" + ip + ":" + port
		}
		if list, err := netinfo.Interfaces(); err == nil {
			for _, c := range list {
				if c.Up && c.IPv4 != "" {
					return scheme + "://" + c.IPv4 + ":" + port
				}
			}
		}
		return scheme + "://" + r.Host
	}
}

// retention deletes what the configuration says not to keep, at start and every six
// hours: old visits, old audit entries, and expired sessions and invites.
func retention(ctx context.Context, st *store.Store, privacy config.Privacy, clock func() time.Time, logger *slog.Logger) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		now := clock()
		if privacy.RetentionDays > 0 {
			cutoff := now.AddDate(0, 0, -privacy.RetentionDays)
			if n, err := st.PurgeVisitsBefore(cutoff); err != nil {
				logger.Error("retention", "err", err)
			} else if n > 0 {
				logger.Info("retention", "deleted_visits", n, "older_than", cutoff.Format(time.DateOnly))
				_ = st.Audit(store.AuditEntry{At: now, Actor: "system", Action: "retention.visits", Detail: fmt.Sprintf("%d visits before %s", n, cutoff.Format(time.DateOnly))})
			}
		}
		if privacy.AuditRetentionDays > 0 {
			if _, err := st.PurgeAuditBefore(now.AddDate(0, 0, -privacy.AuditRetentionDays)); err != nil {
				logger.Error("audit retention", "err", err)
			}
		}
		if err := st.PurgeExpired(now); err != nil {
			logger.Error("purging expired sessions", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func listInterfaces(all bool) error {
	list, err := netinfo.Interfaces()
	if err != nil {
		return err
	}
	chosen, _ := netinfo.Choose("", list)
	fmt.Println("Interfaces (the one WaveTrack would pick is marked →):")
	hidden := 0
	for _, i := range list {
		// Hide virtual and unconnected interfaces (bridges, AWDL, Apple's internal links).
		if !all && !i.Wireless && i.IPv4 == "" {
			hidden++
			continue
		}
		mark := " "
		if i.Name == chosen {
			mark = "→"
		}
		kind := "wired"
		if i.Wireless {
			kind = "Wi-Fi"
		}
		state := "down"
		if i.Up {
			state = "up"
		}
		fmt.Printf("  %s %-10s %-6s %-5s %s\n", mark, i.Name, kind, state, i.IPv4)
	}
	if hidden > 0 {
		fmt.Printf("  (%d virtual or unconnected interfaces hidden; --all shows them)\n", hidden)
	}
	if chosen == "" {
		fmt.Println("\nNo connected Wi-Fi interface. If Wi-Fi and wired devices share one network,")
		fmt.Println("capturing on the wired interface works too: wavetrack --interface <name>")
	}
	return nil
}

func doctor(args []string) error {
	var o serveOptions
	if err := serveFlags(&o).Parse(args); err != nil {
		return err
	}
	ok := true
	check := func(pass bool, label, fix string) {
		if pass {
			fmt.Printf("  ✓ %s\n", label)
			return
		}
		ok = false
		fmt.Printf("  ✗ %s\n      %s\n", label, strings.ReplaceAll(fix, "\n", "\n      "))
	}

	fmt.Printf("WaveTrack %s on %s/%s\n\n", version, runtime.GOOS, runtime.GOARCH)
	cfg, err := loadConfig(o)
	label := "configuration " + o.config
	if _, statErr := os.Stat(o.config); statErr != nil {
		label = "configuration: defaults (" + o.config + " not found)"
	}
	check(err == nil, label, fmt.Sprint(err))

	fmt.Printf("  ✓ %s\n", pcap.Version())

	list, _ := netinfo.Interfaces()
	iface, err := netinfo.Choose(cfg.Capture.Interface, list)
	check(err == nil, "capture interface: "+iface, fmt.Sprint(err))

	if iface != "" {
		live, err := capture.OpenLive(iface, cfg.Capture.MonitorMode)
		if err == nil {
			live.Close()
		}
		fix := "Run WaveTrack with sudo, or grant capture permission once:\n"
		switch runtime.GOOS {
		case "darwin":
			fix += "brew install --cask wireshark-chmodbpf   (then log out and back in)"
		case "windows":
			fix = "Install Npcap from https://npcap.com and check \"WinPcap API-compatible mode\",\nor run WaveTrack from an Administrator prompt."
		default:
			exe, _ := os.Executable()
			fix += "sudo setcap cap_net_raw,cap_net_admin=eip " + exe
		}
		if err != nil && !strings.Contains(err.Error(), "permission") && !strings.Contains(err.Error(), "Npcap") {
			fix = err.Error()
		}
		check(err == nil, "permission to capture on "+iface, fix)
	}

	if cfg.TLS.Enabled() {
		_, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		check(err == nil, "TLS certificate "+cfg.TLS.CertFile, fmt.Sprint(err))
	} else if host, _, _ := net.SplitHostPort(cfg.Listen); host == "" || (net.ParseIP(host) != nil && !net.ParseIP(host).IsLoopback()) {
		fmt.Println("  · the dashboard is served over plain HTTP on the network; add tls: to config.yaml to encrypt it")
	}

	probe := filepath.Join(cfg.DataDir, ".write-test")
	err = os.MkdirAll(cfg.DataDir, 0o750)
	if err == nil {
		err = os.WriteFile(probe, nil, 0o600)
		os.Remove(probe)
	}
	check(err == nil, "data directory "+cfg.DataDir+" is writable", fmt.Sprint(err))

	if l, err := net.Listen("tcp", cfg.Listen); err == nil {
		l.Close()
		check(true, "dashboard address "+cfg.Listen+" is free", "")
	} else {
		check(false, "dashboard address "+cfg.Listen+" is free", err.Error()+"\nPick another with --listen or listen: in config.yaml.")
	}

	if !ok {
		fmt.Println("\nFix the items marked ✗, then run `wavetrack doctor` again.")
		os.Exit(1)
	}
	fmt.Println("\nAll set. Start with `wavetrack`.")
	return nil
}

func token(args []string) error {
	if len(args) == 0 || args[0] != "reset" {
		return errors.New("usage: wavetrack token reset [--config file] [--data dir]")
	}
	var o serveOptions
	if err := serveFlags(&o).Parse(args[1:]); err != nil {
		return err
	}
	cfg, err := loadConfig(o)
	if err != nil {
		return err
	}
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return err
	}
	defer st.Close()
	t, err := server.ResetAdminToken(st)
	if err != nil {
		return err
	}
	_ = st.Audit(store.AuditEntry{At: time.Now(), Actor: "cli", Action: "token.reset", Detail: "every browser signed out"})
	fmt.Printf("New admin token (every browser was signed out):\n  %s\n", t)
	return nil
}

func importV2(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: wavetrack import-v2 <path to 2.x wavetrack.db> [--data dir]")
	}
	var o serveOptions
	if err := serveFlags(&o).Parse(args[1:]); err != nil {
		return err
	}
	cfg, err := loadConfig(o)
	if err != nil {
		return err
	}
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return err
	}
	defer st.Close()
	if people, _ := st.People(true); len(people) > 0 {
		return fmt.Errorf("%s already has people; import into an empty data directory", cfg.DataDir)
	}
	res, err := importer.V2(args[0], st, time.Now())
	if err != nil {
		return err
	}
	fmt.Printf("Imported %d people and %d visits into %s.\n", res.People, res.Visits, cfg.DataDir)
	return nil
}
