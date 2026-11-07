// Package server is WaveTrack's HTTP API and dashboard.
package server

import (
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/lucasrafaldini/wavetrack/internal/capture"
	"github.com/lucasrafaldini/wavetrack/internal/presence"
	"github.com/lucasrafaldini/wavetrack/internal/report"
	"github.com/lucasrafaldini/wavetrack/internal/store"
)

const inviteLifetime = 24 * time.Hour

type Options struct {
	Store     *store.Store
	Tracker   *presence.Tracker
	Ident     presence.Identifier
	Capture   capture.Description
	Assets    fs.FS
	Clock     func() time.Time
	Location  *time.Location
	AwayAfter time.Duration
	// LookupMAC resolves a registering phone's IP to its MAC. Swappable in tests.
	LookupMAC func(ip string) (net.HardwareAddr, error)
	// InviteBase is the scheme://host:port phones use to reach WaveTrack, or empty to derive it.
	InviteBase func(r *http.Request) string
	Version    string
	Logger     *slog.Logger
}

type Server struct {
	Options
	store *store.Store
	clock func() time.Time
}

func New(opts Options) *Server {
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	if opts.Location == nil {
		opts.Location = time.Local
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Server{Options: opts, store: opts.Store, clock: opts.Clock}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	auth := s.requireSession

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "ok\n") })

	mux.HandleFunc("POST /api/login", s.login)
	mux.HandleFunc("POST /api/logout", s.logout)
	mux.HandleFunc("GET /api/me", auth(func(w http.ResponseWriter, r *http.Request) { writeJSON(w, map[string]bool{"signed_in": true}) }))

	mux.HandleFunc("GET /api/status", auth(s.status))
	mux.HandleFunc("GET /api/now", auth(s.now))
	mux.HandleFunc("GET /api/timeline", auth(s.timeline))
	mux.HandleFunc("GET /api/history", auth(s.history))
	mux.HandleFunc("GET /api/export.csv", auth(s.exportCSV))
	mux.HandleFunc("GET /api/nearby", auth(s.nearby))

	mux.HandleFunc("GET /api/people", auth(s.listPeople))
	mux.HandleFunc("POST /api/people", auth(s.createPerson))
	mux.HandleFunc("PATCH /api/people/{id}", auth(s.updatePerson))
	mux.HandleFunc("DELETE /api/people/{id}", auth(s.deletePerson))
	mux.HandleFunc("POST /api/people/{id}/devices", auth(s.addDevice))
	mux.HandleFunc("DELETE /api/devices/{mac}", auth(s.removeDevice))

	mux.HandleFunc("POST /api/invites", auth(s.createInvite))
	mux.HandleFunc("GET /api/invites/{token}", s.checkInvite)
	mux.HandleFunc("POST /api/register", s.register)

	files := http.FileServerFS(s.Assets)
	mux.Handle("GET /assets/", files)
	mux.HandleFunc("GET /join/{token}", func(w http.ResponseWriter, r *http.Request) { s.page(w, "join.html") })
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { s.page(w, "index.html") })

	return securityHeaders(sameOrigin(mux))
}

func (s *Server) page(w http.ResponseWriter, name string) {
	data, err := fs.ReadFile(s.Assets, name)
	if err != nil {
		http.NotFound(w, nil)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(data)
}

// MARK: session

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if !s.checkAdminToken(body.Token) {
		time.Sleep(400 * time.Millisecond) // slow down guessing
		writeError(w, http.StatusUnauthorized, "That access token isn’t right.")
		return
	}
	if err := s.startSession(w, r); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, map[string]bool{"signed_in": true})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	s.endSession(w, r)
	w.WriteHeader(http.StatusNoContent)
}

// MARK: now

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"now": s.clock(), "capture": s.Capture, "version": s.Version})
}

type personNow struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Team         string     `json:"team"`
	Present      bool       `json:"present"`
	Since        *time.Time `json:"since,omitempty"`
	LastSeen     *time.Time `json:"last_seen,omitempty"`
	TodaySeconds int64      `json:"today_seconds"`
	FirstToday   *time.Time `json:"first_today,omitempty"`
	LastToday    *time.Time `json:"last_today,omitempty"`
	Devices      int        `json:"devices"`
}

type eventView struct {
	Kind presence.EventKind `json:"kind"`
	Name string             `json:"name"`
	At   time.Time          `json:"at"`
}

func (s *Server) now(w http.ResponseWriter, r *http.Request) {
	now := s.clock()
	people, err := s.store.People(false)
	if err != nil {
		s.fail(w, err)
		return
	}
	today := report.StartOfDay(now, s.Location)
	visits, err := s.store.VisitsBetween(today, today.AddDate(0, 0, 1))
	if err != nil {
		s.fail(w, err)
		return
	}
	totals := report.Daily(visits, []time.Time{today}, now)
	snap := s.Tracker.Snapshot(now)
	stays := map[int64]presence.Stay{}
	for _, st := range snap.Present {
		stays[st.PersonID] = st
	}

	names := map[int64]string{}
	list := make([]personNow, 0, len(people))
	for _, p := range people {
		names[p.ID] = p.Name
		item := personNow{ID: p.ID, Name: p.Name, Team: p.Team, Devices: len(p.Devices)}
		if st, ok := stays[p.ID]; ok {
			since, seen := st.Since, st.LastSeen
			item.Present, item.Since, item.LastSeen = true, &since, &seen
		}
		if days, ok := totals[p.ID]; ok {
			item.TodaySeconds = days[0].Seconds
			item.FirstToday, item.LastToday = days[0].First, days[0].Last
		}
		list = append(list, item)
	}

	// The day's arrivals and departures come from visits, so they survive restarts.
	events := []eventView{}
	for _, v := range visits {
		name, ok := names[v.PersonID]
		if !ok {
			continue
		}
		if !v.StartedAt.Before(today) {
			events = append(events, eventView{Kind: presence.Arrived, Name: name, At: v.StartedAt})
		}
		if v.EndedAt != nil && !v.EndedAt.Before(today) {
			events = append(events, eventView{Kind: presence.Left, Name: name, At: *v.EndedAt})
		}
	}
	sort.Slice(events, func(i, j int) bool { return events[i].At.After(events[j].At) })

	writeJSON(w, map[string]any{
		"now":        now,
		"away_after": s.AwayAfter.Seconds(),
		"capture":    s.Capture,
		"version":    s.Version,
		"people":     list,
		"events":     events,
		"nearby":     len(snap.Nearby),
	})
}

// MARK: timeline and history

func (s *Server) timeline(w http.ResponseWriter, r *http.Request) {
	now := s.clock()
	day := report.StartOfDay(now, s.Location)
	if raw := r.URL.Query().Get("date"); raw != "" {
		parsed, err := time.ParseInLocation(time.DateOnly, raw, s.Location)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Use a date like 2026-11-05.")
			return
		}
		day = parsed
	}
	people, err := s.store.People(true)
	if err != nil {
		s.fail(w, err)
		return
	}
	visits, err := s.store.VisitsBetween(day, day.AddDate(0, 0, 1))
	if err != nil {
		s.fail(w, err)
		return
	}
	intervals := report.Timeline(visits, day, now)

	type row struct {
		ID        int64             `json:"id"`
		Name      string            `json:"name"`
		Team      string            `json:"team"`
		Archived  bool              `json:"archived"`
		Intervals []report.Interval `json:"intervals"`
	}
	rows := []row{}
	for _, p := range people {
		ivs := intervals[p.ID]
		if p.ArchivedAt != nil && len(ivs) == 0 {
			continue
		}
		if ivs == nil {
			ivs = []report.Interval{}
		}
		rows = append(rows, row{p.ID, p.Name, p.Team, p.ArchivedAt != nil, ivs})
	}
	writeJSON(w, map[string]any{"date": day.Format(time.DateOnly), "now": now, "people": rows})
}

type historyRow struct {
	ID       int64        `json:"id"`
	Name     string       `json:"name"`
	Team     string       `json:"team"`
	Archived bool         `json:"archived"`
	Days     []report.Day `json:"days"`
	Total    int64        `json:"total_seconds"`
}

func (s *Server) historyRows(days int) ([]string, []historyRow, error) {
	now := s.clock()
	starts := report.LastDays(now, days, s.Location)
	visits, err := s.store.VisitsBetween(starts[0], starts[len(starts)-1].AddDate(0, 0, 1))
	if err != nil {
		return nil, nil, err
	}
	people, err := s.store.People(true)
	if err != nil {
		return nil, nil, err
	}
	totals := report.Daily(visits, starts, now)

	dates := make([]string, len(starts))
	for i, d := range starts {
		dates[i] = d.Format(time.DateOnly)
	}
	rows := []historyRow{}
	for _, p := range people {
		days, ok := totals[p.ID]
		if !ok {
			if p.ArchivedAt != nil {
				continue
			}
			days = make([]report.Day, len(starts))
			for i := range days {
				days[i].Date = dates[i]
			}
		}
		row := historyRow{ID: p.ID, Name: p.Name, Team: p.Team, Archived: p.ArchivedAt != nil, Days: days}
		for _, d := range days {
			row.Total += d.Seconds
		}
		rows = append(rows, row)
	}
	return dates, rows, nil
}

func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	days := clampInt(r.URL.Query().Get("days"), 7, 1, 31)
	dates, rows, err := s.historyRows(days)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, map[string]any{"dates": dates, "people": rows})
}

func (s *Server) exportCSV(w http.ResponseWriter, r *http.Request) {
	days := clampInt(r.URL.Query().Get("days"), 30, 1, 366)
	dates, rows, err := s.historyRows(days)
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="wavetrack-%s-to-%s.csv"`, dates[0], dates[len(dates)-1]))
	out := csv.NewWriter(w)
	out.Write([]string{"date", "name", "team", "first_seen", "last_seen", "hours"})
	for _, row := range rows {
		for _, d := range row.Days {
			if d.Seconds == 0 {
				continue
			}
			out.Write([]string{d.Date, csvSafe(row.Name), csvSafe(row.Team), clock(d.First, s.Location), clock(d.Last, s.Location),
				strconv.FormatFloat(float64(d.Seconds)/3600, 'f', 2, 64)})
		}
	}
	out.Flush()
}

// csvSafe defuses spreadsheet formula injection from user-entered names.
func csvSafe(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}

func clock(t *time.Time, loc *time.Location) string {
	if t == nil {
		return ""
	}
	return t.In(loc).Format("15:04")
}

// MARK: nearby

func (s *Server) nearby(w http.ResponseWriter, r *http.Request) {
	now := s.clock()
	writeJSON(w, map[string]any{"now": now, "devices": s.Tracker.Snapshot(now).Nearby})
}

// MARK: people

type personInput struct {
	Name string `json:"name"`
	Team string `json:"team"`
}

func (in personInput) problem() string {
	name := strings.TrimSpace(in.Name)
	switch {
	case name == "":
		return "A name is required."
	case utf8.RuneCountInString(name) > 80:
		return "Keep the name under 80 characters."
	case utf8.RuneCountInString(in.Team) > 60:
		return "Keep the team under 60 characters."
	}
	return ""
}

func (s *Server) listPeople(w http.ResponseWriter, r *http.Request) {
	people, err := s.store.People(false)
	if err != nil {
		s.fail(w, err)
		return
	}
	if people == nil {
		people = []store.Person{}
	}
	writeJSON(w, map[string]any{"people": people})
}

func (s *Server) createPerson(w http.ResponseWriter, r *http.Request) {
	var body struct {
		personInput
		Devices []deviceInput `json:"devices"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if msg := body.problem(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	for _, d := range body.Devices {
		if msg := d.problem(); msg != "" {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
	}
	now := s.clock()
	person, err := s.store.CreatePerson(body.Name, body.Team, nil, now)
	if err != nil {
		s.fail(w, err)
		return
	}
	for _, d := range body.Devices {
		if err := s.store.AddDevice(s.device(d, person.ID, now)); err != nil {
			_ = s.store.ErasePerson(person.ID)
			s.deviceError(w, err)
			return
		}
	}
	s.reload()
	person, _ = s.store.Person(person.ID)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, person)
}

func (s *Server) updatePerson(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body personInput
	if !readJSON(w, r, &body) {
		return
	}
	if msg := body.problem(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if err := s.store.UpdatePerson(id, body.Name, body.Team); err != nil {
		s.notFoundOr(w, err)
		return
	}
	person, _ := s.store.Person(id)
	writeJSON(w, person)
}

// deletePerson either anonymizes (keeps attendance totals) or erases everything.
func (s *Server) deletePerson(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var err error
	switch r.URL.Query().Get("mode") {
	case "erase":
		err = s.store.ErasePerson(id)
	case "", "archive":
		err = s.store.ArchivePerson(id, s.clock())
	default:
		writeError(w, http.StatusBadRequest, "mode must be archive or erase.")
		return
	}
	if err != nil {
		s.notFoundOr(w, err)
		return
	}
	s.Tracker.Forget(id)
	s.reload()
	w.WriteHeader(http.StatusNoContent)
}

type deviceInput struct {
	MAC   string `json:"mac"`
	Label string `json:"label"`
}

func (d deviceInput) problem() string {
	if _, err := store.NormalizeMAC(d.MAC); err != nil {
		return "“" + d.MAC + "” isn’t a valid MAC address."
	}
	if utf8.RuneCountInString(d.Label) > 40 {
		return "Keep device labels under 40 characters."
	}
	return ""
}

func (s *Server) device(in deviceInput, personID int64, now time.Time) store.Device {
	mac, _ := store.NormalizeMAC(in.MAC)
	hw, _ := net.ParseMAC(mac)
	info := s.Ident.Identify(hw)
	label := strings.TrimSpace(in.Label)
	if label == "" {
		label = info.Vendor
	}
	return store.Device{MAC: mac, PersonID: personID, Label: label, Vendor: info.Vendor, Kind: string(info.Kind), AddedAt: now}
}

func (s *Server) addDevice(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body deviceInput
	if !readJSON(w, r, &body) {
		return
	}
	if msg := body.problem(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if err := s.store.AddDevice(s.device(body, id, s.clock())); err != nil {
		s.deviceError(w, err)
		return
	}
	s.reload()
	person, _ := s.store.Person(id)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, person)
}

func (s *Server) removeDevice(w http.ResponseWriter, r *http.Request) {
	if err := s.store.RemoveDevice(r.PathValue("mac")); err != nil {
		s.notFoundOr(w, err)
		return
	}
	s.reload()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deviceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrDeviceTaken):
		writeError(w, http.StatusConflict, "That device already belongs to someone.")
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "That person doesn’t exist.")
	default:
		s.fail(w, err)
	}
}

func (s *Server) reload() {
	if err := s.Tracker.ReloadDevices(); err != nil {
		s.Logger.Error("reloading devices", "err", err)
	}
}

// MARK: invites and self-registration

func (s *Server) createInvite(w http.ResponseWriter, r *http.Request) {
	token := NewToken("")
	now := s.clock()
	expires := now.Add(inviteLifetime)
	if err := s.store.CreateInvite(Hash(token), now, expires); err != nil {
		s.fail(w, err)
		return
	}
	url := s.InviteBase(r) + "/join/" + token
	png, err := qrcode.Encode(url, qrcode.Medium, 512)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, map[string]any{
		"url":        url,
		"qr":         "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
		"expires_at": expires,
	})
}

func (s *Server) checkInvite(w http.ResponseWriter, r *http.Request) {
	ok, err := s.store.InviteValid(Hash(r.PathValue("token")), s.clock())
	if err != nil {
		s.fail(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusGone, "This invite has expired or was already used. Ask for a new one.")
		return
	}
	writeJSON(w, map[string]bool{"valid": true})
}

// register lets someone add their own phone. The MAC comes only from the ARP entry of the
// request's source address: never from anything the client sends, and never guessed.
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token   string `json:"token"`
		Consent bool   `json:"consent"`
		personInput
	}
	if !readJSON(w, r, &body) {
		return
	}
	if !body.Consent {
		writeError(w, http.StatusBadRequest, "Please confirm you agree to presence tracking.")
		return
	}
	if msg := body.problem(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	mac, err := s.LookupMAC(host)
	if err != nil {
		s.Logger.Info("registration without a resolvable device", "ip", host, "err", err)
		writeError(w, http.StatusUnprocessableEntity, "We couldn’t identify this phone. Connect it to the office Wi-Fi (not mobile data) and try again.")
		return
	}
	now := s.clock()
	device := s.device(deviceInput{MAC: mac.String(), Label: "Phone"}, 0, now)
	person, err := s.store.RedeemInvite(Hash(body.Token), now, body.Name, body.Team, device)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusGone, "This invite has expired or was already used. Ask for a new one.")
		return
	case errors.Is(err, store.ErrDeviceTaken):
		writeError(w, http.StatusConflict, "This phone is already registered.")
		return
	case err != nil:
		s.fail(w, err)
		return
	}
	s.reload()
	writeJSON(w, map[string]any{"name": person.Name})
}

// MARK: helpers

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Send JSON.")
		return false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "That request couldn’t be read.")
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusNotFound, "Not found.")
		return 0, false
	}
	return id, true
}

func clampInt(raw string, fallback, lo, hi int) int {
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return max(lo, min(hi, v))
}

func (s *Server) notFoundOr(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not found.")
		return
	}
	if strings.Contains(err.Error(), "valid MAC") {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.fail(w, err)
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	s.Logger.Error("request failed", "err", err)
	writeError(w, http.StatusInternalServerError, "Something went wrong. Check WaveTrack’s log.")
}
