package server

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/capture"
	"github.com/lucasrafaldini/wavetrack/internal/presence"
	"github.com/lucasrafaldini/wavetrack/internal/store"
	"github.com/lucasrafaldini/wavetrack/internal/testutil"
	"github.com/lucasrafaldini/wavetrack/web"
)

type harness struct {
	t      *testing.T
	srv    *httptest.Server
	client *http.Client
	store  *store.Store
	token  string
	mac    net.HardwareAddr
	macErr error
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{t: t, store: testutil.Store(t)}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clock := func() time.Time { return testutil.Day(10, 0) }
	tracker := presence.New(h.store, testutil.Identifier{}, presence.Options{AwayAfter: 20 * time.Minute, Logger: logger, Clock: clock})
	if err := tracker.Start(clock()); err != nil {
		t.Fatal(err)
	}
	h.mac = testutil.MAC(t, "a4:83:e7:21:5c:10")
	s := New(Options{
		Store: h.store, Tracker: tracker, Ident: testutil.Identifier{}, Capture: capture.Description{Kind: "demo", Mode: "simulated"},
		Assets: web.FS, Clock: clock, AwayAfter: 20 * time.Minute, Logger: logger, Version: "test",
		LookupMAC:  func(string) (net.HardwareAddr, error) { return h.mac, h.macErr },
		InviteBase: func(*http.Request) string { return "http://192.168.1.2:8080" },
	})
	h.srv = httptest.NewServer(s.Handler())
	t.Cleanup(h.srv.Close)
	jar, _ := cookiejar.New(nil)
	h.client = &http.Client{Jar: jar}
	token, err := EnsureAdminToken(h.store)
	if err != nil || token == "" {
		t.Fatalf("token = %q, %v", token, err)
	}
	h.token = token
	return h
}

func (h *harness) do(method, path, body string, headers ...string) *http.Response {
	h.t.Helper()
	req, _ := http.NewRequest(method, h.srv.URL+path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	res, err := h.client.Do(req)
	if err != nil {
		h.t.Fatal(err)
	}
	h.t.Cleanup(func() { res.Body.Close() })
	return res
}

func (h *harness) signIn() {
	h.t.Helper()
	if res := h.do("POST", "/api/login", `{"token":"`+h.token+`"}`); res.StatusCode != 200 {
		h.t.Fatalf("login = %d", res.StatusCode)
	}
}

func decode[T any](t *testing.T, res *http.Response) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestAdminAPIRequiresASession(t *testing.T) {
	h := newHarness(t)
	for _, path := range []string{"/api/now", "/api/people", "/api/history", "/api/nearby", "/api/export.csv"} {
		if res := h.do("GET", path, ""); res.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s = %d without a session", path, res.StatusCode)
		}
	}
	if res := h.do("POST", "/api/login", `{"token":"wt_wrong"}`); res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong token = %d", res.StatusCode)
	}
	h.signIn()
	if res := h.do("GET", "/api/now", ""); res.StatusCode != 200 {
		t.Fatalf("after sign-in = %d", res.StatusCode)
	}
}

func TestSessionCookieIsLockedDown(t *testing.T) {
	h := newHarness(t)
	res := h.do("POST", "/api/login", `{"token":"`+h.token+`"}`)
	cookie := res.Header.Get("Set-Cookie")
	for _, want := range []string{"HttpOnly", "SameSite=Strict", "wavetrack_session="} {
		if !strings.Contains(cookie, want) {
			t.Errorf("cookie %q lacks %s", cookie, want)
		}
	}
}

func TestResettingTheTokenSignsEveryoneOut(t *testing.T) {
	h := newHarness(t)
	h.signIn()
	if _, err := ResetAdminToken(h.store); err != nil {
		t.Fatal(err)
	}
	if res := h.do("GET", "/api/now", ""); res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old session still works: %d", res.StatusCode)
	}
}

func TestCrossSiteWritesAreBlocked(t *testing.T) {
	h := newHarness(t)
	h.signIn()
	res := h.do("POST", "/api/people", `{"name":"Mallory"}`, "Origin", "https://evil.example")
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-site POST = %d", res.StatusCode)
	}
	if res := h.do("POST", "/api/people", `{"name":"Ana"}`, "Origin", h.srv.URL); res.StatusCode != http.StatusCreated {
		t.Fatalf("same-origin POST = %d", res.StatusCode)
	}
}

func TestWritesMustBeJSON(t *testing.T) {
	h := newHarness(t)
	h.signIn()
	req, _ := http.NewRequest("POST", h.srv.URL+"/api/people", strings.NewReader("name=Ana"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := h.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("form POST = %d", res.StatusCode)
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := newHarness(t)
	res := h.do("GET", "/", "")
	if csp := res.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'self'") || strings.Contains(csp, "unsafe-inline") {
		t.Errorf("CSP = %q", csp)
	}
	if res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing nosniff")
	}
}

func TestPeopleAndDevices(t *testing.T) {
	h := newHarness(t)
	h.signIn()
	res := h.do("POST", "/api/people", `{"name":"Ana Ribeiro","team":"Design","devices":[{"mac":"A4-83-E7-21-5C-10","label":"iPhone"}]}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d", res.StatusCode)
	}
	ana := decode[store.Person](t, res)
	if len(ana.Devices) != 1 || ana.Devices[0].MAC != "a4:83:e7:21:5c:10" || ana.Devices[0].Vendor != "Acme" {
		t.Fatalf("person = %+v", ana)
	}
	if res := h.do("POST", "/api/people", `{"name":"Copy","devices":[{"mac":"a4:83:e7:21:5c:10"}]}`); res.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate device = %d", res.StatusCode)
	}
	if people, _ := h.store.People(true); len(people) != 1 {
		t.Fatalf("a failed create left %d people behind", len(people))
	}
	if res := h.do("POST", "/api/people", `{"name":"  "}`); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("blank name = %d", res.StatusCode)
	}
	if res := h.do("POST", "/api/people", `{"name":"Ana","is_admin":true}`); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown field = %d", res.StatusCode)
	}
	if res := h.do("DELETE", "/api/devices/a4:83:e7:21:5c:10", ""); res.StatusCode != http.StatusNoContent {
		t.Fatalf("remove device = %d", res.StatusCode)
	}
}

func TestDeletingAPersonErasesOrAnonymizes(t *testing.T) {
	h := newHarness(t)
	h.signIn()
	ana, _ := h.store.CreatePerson("Ana", "", nil, testutil.Day(8, 0))
	bruno, _ := h.store.CreatePerson("Bruno", "", nil, testutil.Day(8, 0))
	_ = h.store.InsertVisit(ana.ID, testutil.Day(8, 0), testutil.Day(9, 0))
	_ = h.store.InsertVisit(bruno.ID, testutil.Day(8, 0), testutil.Day(9, 0))

	if res := h.do("DELETE", "/api/people/"+itoa(ana.ID)+"?mode=erase", ""); res.StatusCode != http.StatusNoContent {
		t.Fatalf("erase = %d", res.StatusCode)
	}
	if res := h.do("DELETE", "/api/people/"+itoa(bruno.ID), ""); res.StatusCode != http.StatusNoContent {
		t.Fatalf("archive = %d", res.StatusCode)
	}
	visits, _ := h.store.VisitsBetween(testutil.Day(0, 0), testutil.Day(23, 0))
	if len(visits) != 1 || visits[0].PersonID != bruno.ID {
		t.Fatalf("visits = %+v, want only Bruno's anonymized visit", visits)
	}
	if res := h.do("DELETE", "/api/people/"+itoa(bruno.ID)+"?mode=shred", ""); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad mode = %d", res.StatusCode)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func invite(h *harness) string {
	h.t.Helper()
	res := h.do("POST", "/api/invites", `{}`)
	if res.StatusCode != 200 {
		h.t.Fatalf("invite = %d", res.StatusCode)
	}
	body := decode[map[string]string](h.t, res)
	if !strings.HasPrefix(body["url"], "http://192.168.1.2:8080/join/") || !strings.HasPrefix(body["qr"], "data:image/png;base64,") {
		h.t.Fatalf("invite = %+v", body)
	}
	return strings.TrimPrefix(body["url"], "http://192.168.1.2:8080/join/")
}

// Registration takes the MAC from ARP only. 2.x trusted an X-MAC-Address header and,
// failing that, assigned the most recently seen device: someone else's phone.
func TestRegistrationUsesOnlyTheARPAddress(t *testing.T) {
	h := newHarness(t)
	h.signIn()
	token := invite(h)

	h.client.Jar, _ = cookiejar.New(nil) // the phone has no admin session
	if res := h.do("GET", "/api/invites/"+token, ""); res.StatusCode != 200 {
		t.Fatalf("check invite = %d", res.StatusCode)
	}
	if res := h.do("POST", "/api/register", `{"token":"`+token+`","name":"Ana","team":"Design","consent":false}`); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("no consent = %d", res.StatusCode)
	}
	res := h.do("POST", "/api/register", `{"token":"`+token+`","name":"Ana","team":"Design","consent":true}`, "X-MAC-Address", "11:22:33:44:55:66")
	if res.StatusCode != 200 {
		t.Fatalf("register = %d", res.StatusCode)
	}
	devices, _ := h.store.Devices()
	if len(devices) != 1 || devices[0].MAC != "a4:83:e7:21:5c:10" {
		t.Fatalf("devices = %+v, want the ARP address, not the header", devices)
	}
	people, _ := h.store.People(false)
	if people[0].ConsentedAt == nil {
		t.Fatal("consent wasn't recorded")
	}
	if res := h.do("POST", "/api/register", `{"token":"`+token+`","name":"Again","consent":true}`); res.StatusCode != http.StatusGone {
		t.Fatalf("reused invite = %d", res.StatusCode)
	}
}

func TestRegistrationRefusesWhenThePhoneCantBeIdentified(t *testing.T) {
	h := newHarness(t)
	h.signIn()
	token := invite(h)
	h.macErr = errors.New("no ARP entry")
	if res := h.do("POST", "/api/register", `{"token":"`+token+`","name":"Ana","consent":true}`); res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("unidentified phone = %d", res.StatusCode)
	}
	if ok, _ := h.store.InviteValid(Hash(token), testutil.Day(10, 0)); !ok {
		t.Fatal("a failed registration burned the invite")
	}
	if devices, _ := h.store.Devices(); len(devices) != 0 {
		t.Fatalf("guessed a device: %+v", devices)
	}
}

func TestCSVExportDefusesFormulas(t *testing.T) {
	h := newHarness(t)
	h.signIn()
	p, _ := h.store.CreatePerson(`=HYPERLINK("http://evil")`, "Ops", nil, testutil.Day(8, 0))
	_ = h.store.InsertVisit(p.ID, testutil.Day(8, 30), testutil.Day(9, 45))
	res := h.do("GET", "/api/export.csv?days=1", "")
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), `'=HYPERLINK`) || !strings.Contains(string(body), "1.25") {
		t.Fatalf("csv = %s", body)
	}
	if !strings.Contains(res.Header.Get("Content-Disposition"), "attachment") {
		t.Error("export isn't a download")
	}
}

func TestNowReportsPresenceAndTheDaysEvents(t *testing.T) {
	h := newHarness(t)
	h.signIn()
	p, _ := h.store.CreatePerson("Ana", "Design", nil, testutil.Day(8, 0))
	_ = h.store.InsertVisit(p.ID, testutil.Day(8, 30), testutil.Day(9, 15))
	res := h.do("GET", "/api/now", "")
	body := decode[struct {
		People []personNow `json:"people"`
		Events []eventView `json:"events"`
	}](t, res)
	if len(body.People) != 1 || body.People[0].TodaySeconds != 45*60 || body.People[0].Present {
		t.Fatalf("people = %+v", body.People)
	}
	if len(body.Events) != 2 || body.Events[0].Kind != presence.Left {
		t.Fatalf("events = %+v, want departure first", body.Events)
	}
}
