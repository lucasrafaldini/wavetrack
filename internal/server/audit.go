package server

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/store"
)

// The audit log answers "who changed what, and when". It never stores names or full
// MAC addresses: people appear as "person #12" and devices with their middle octets
// masked, so erasing someone doesn't leave their data behind in the log.

const (
	actorAnonymous = "anonymous"
	actorInvite    = "invite"
	actorSystem    = "system"
	actorCLI       = "cli"
)

// sessionActor identifies the browser session behind an admin request: "admin:3f9a1c".
// Everyone shares the admin token, so the session is what tells two browsers apart.
func sessionActor(r *http.Request) string {
	if cookie, err := r.Cookie(sessionCookie); err == nil && cookie.Value != "" {
		return "admin:" + Hash(cookie.Value)[:6]
	}
	return "admin"
}

func (s *Server) record(r *http.Request, actor, action, target, detail string) {
	entry := store.AuditEntry{At: s.clock(), Actor: actor, IP: clientIP(r), Action: action, Target: target, Detail: detail}
	if err := s.store.Audit(entry); err != nil {
		s.Logger.Error("audit log", "action", action, "err", err)
	}
}

func personRef(id int64) string { return fmt.Sprintf("person #%d", id) }

// maskMAC keeps the manufacturer prefix and the last octet: enough to tell devices
// apart in the log, not enough to track anyone with it.
func maskMAC(mac string) string {
	parts := strings.Split(mac, ":")
	if len(parts) != 6 {
		return "device"
	}
	return strings.Join([]string{parts[0], parts[1], parts[2], "••", "••", parts[5]}, ":")
}

func (s *Server) auditLog(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	before, _ := strconv.ParseInt(q.Get("before"), 10, 64)
	limit := clampInt(q.Get("limit"), 100, 1, 500)
	entries, err := s.store.AuditLog(before, limit, q.Get("action"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, map[string]any{"entries": entries, "more": len(entries) == limit})
}

func (s *Server) auditCSV(w http.ResponseWriter, r *http.Request) {
	s.record(r, sessionActor(r), "audit.export", "", "")
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="wavetrack-audit-%s.csv"`, s.clock().In(s.Location).Format(time.DateOnly)))
	out := csv.NewWriter(w)
	out.Write([]string{"time", "actor", "ip", "action", "target", "detail"})
	var before int64
	for {
		page, err := s.store.AuditLog(before, 500, "")
		if err != nil {
			s.Logger.Error("audit export", "err", err)
			break
		}
		for _, e := range page {
			out.Write([]string{e.At.In(s.Location).Format(time.RFC3339), e.Actor, e.IP, e.Action, csvSafe(e.Target), csvSafe(e.Detail)})
		}
		if len(page) < 500 {
			break
		}
		before = page[len(page)-1].ID
	}
	out.Flush()
}
