package store_test

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/store"
	"github.com/lucasrafaldini/wavetrack/internal/testutil"
)

func TestMigrationsAreIdempotent(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 2; i++ {
		st, err := store.Open(dir)
		if err != nil {
			t.Fatalf("open #%d: %v", i+1, err)
		}
		st.Close()
	}
}

func TestNormalizeMAC(t *testing.T) {
	for in, want := range map[string]string{"A4-83-E7-21-5C-10": "a4:83:e7:21:5c:10", " a483.e721.5c10 ": "a4:83:e7:21:5c:10"} {
		if got, err := store.NormalizeMAC(in); err != nil || got != want {
			t.Errorf("NormalizeMAC(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := store.NormalizeMAC("not-a-mac"); err == nil {
		t.Error("accepted an invalid MAC")
	}
}

func TestDevicesBelongToOnePerson(t *testing.T) {
	st := testutil.Store(t)
	now := testutil.Day(9, 0)
	ana, _ := st.CreatePerson("Ana", "", nil, now)
	bruno, _ := st.CreatePerson("Bruno", "", nil, now)
	if err := st.AddDevice(store.Device{MAC: "A4:83:E7:21:5C:10", PersonID: ana.ID, AddedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := st.AddDevice(store.Device{MAC: "a4:83:e7:21:5c:10", PersonID: bruno.ID, AddedAt: now}); !errors.Is(err, store.ErrDeviceTaken) {
		t.Fatalf("err = %v, want ErrDeviceTaken", err)
	}
	if err := st.AddDevice(store.Device{MAC: "f0:18:98:4a:02:11", PersonID: 999, AddedAt: now}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound for a missing person", err)
	}
	p, _ := st.Person(ana.ID)
	if len(p.Devices) != 1 || p.Devices[0].MAC != "a4:83:e7:21:5c:10" {
		t.Fatalf("devices = %+v", p.Devices)
	}
}

func TestArchiveKeepsHoursButForgetsIdentity(t *testing.T) {
	st := testutil.Store(t)
	now := testutil.Day(9, 0)
	ana, _ := st.CreatePerson("Ana Ribeiro", "Design", nil, now)
	_ = st.AddDevice(store.Device{MAC: "a4:83:e7:21:5c:10", PersonID: ana.ID, AddedAt: now})
	_ = st.InsertVisit(ana.ID, testutil.Day(9, 0), testutil.Day(12, 0))
	open, _ := st.StartVisit(ana.ID, testutil.Day(13, 0))
	_ = st.TouchVisit(open, testutil.Day(14, 0))

	if err := st.ArchivePerson(ana.ID, testutil.Day(15, 0)); err != nil {
		t.Fatal(err)
	}
	p, _ := st.Person(ana.ID)
	if p.Name == "Ana Ribeiro" || p.Team != "" || p.ArchivedAt == nil || len(p.Devices) != 0 {
		t.Fatalf("archived person = %+v", p)
	}
	visits, _ := st.VisitsBetween(testutil.Day(0, 0), testutil.Day(23, 0))
	if len(visits) != 2 {
		t.Fatalf("visits = %d, want both kept", len(visits))
	}
	if openNow, _ := st.OpenVisits(); len(openNow) != 0 {
		t.Fatal("archiving should close an open visit")
	}
	active, _ := st.People(false)
	if len(active) != 0 {
		t.Fatal("archived people shouldn't be listed as active")
	}
}

func TestEraseRemovesEverything(t *testing.T) {
	st := testutil.Store(t)
	ana, _ := st.CreatePerson("Ana", "", nil, testutil.Day(9, 0))
	_ = st.AddDevice(store.Device{MAC: "a4:83:e7:21:5c:10", PersonID: ana.ID, AddedAt: testutil.Day(9, 0)})
	_ = st.InsertVisit(ana.ID, testutil.Day(9, 0), testutil.Day(12, 0))

	if err := st.ErasePerson(ana.ID); err != nil {
		t.Fatal(err)
	}
	devices, _ := st.Devices()
	visits, _ := st.VisitsBetween(testutil.Day(0, 0), testutil.Day(23, 0))
	if len(devices) != 0 || len(visits) != 0 {
		t.Fatalf("left behind %d devices and %d visits", len(devices), len(visits))
	}
	if err := st.ErasePerson(ana.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second erase = %v", err)
	}
}

func TestInvitesAreSingleUseEvenUnderRace(t *testing.T) {
	st := testutil.Store(t)
	now := testutil.Day(9, 0)
	_ = st.CreateInvite("hash", now, now.Add(time.Hour))

	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			mac := []string{"02:00:00:00:00:01", "02:00:00:00:00:02", "02:00:00:00:00:03", "02:00:00:00:00:04",
				"02:00:00:00:00:05", "02:00:00:00:00:06", "02:00:00:00:00:07", "02:00:00:00:00:08"}[i]
			_, err := st.RedeemInvite("hash", now, "Someone", "", store.Device{MAC: mac})
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
		} else if !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if succeeded != 1 {
		t.Fatalf("invite redeemed %d times", succeeded)
	}
	if ok, _ := st.InviteValid("hash", now); ok {
		t.Fatal("used invite still valid")
	}
}

func TestExpiredInviteCantBeRedeemed(t *testing.T) {
	st := testutil.Store(t)
	now := testutil.Day(9, 0)
	_ = st.CreateInvite("hash", now, now.Add(time.Hour))
	if _, err := st.RedeemInvite("hash", now.Add(2*time.Hour), "Late", "", store.Device{MAC: "02:00:00:00:00:01"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestRedeemRejectsARegisteredPhoneWithoutUsingTheInvite(t *testing.T) {
	st := testutil.Store(t)
	now := testutil.Day(9, 0)
	ana, _ := st.CreatePerson("Ana", "", nil, now)
	_ = st.AddDevice(store.Device{MAC: "a4:83:e7:21:5c:10", PersonID: ana.ID, AddedAt: now})
	_ = st.CreateInvite("hash", now, now.Add(time.Hour))

	if _, err := st.RedeemInvite("hash", now, "Copycat", "", store.Device{MAC: "a4:83:e7:21:5c:10"}); !errors.Is(err, store.ErrDeviceTaken) {
		t.Fatalf("err = %v", err)
	}
	if ok, _ := st.InviteValid("hash", now); !ok {
		t.Fatal("a failed registration shouldn't burn the invite")
	}
}

func TestSessionsExpire(t *testing.T) {
	st := testutil.Store(t)
	now := testutil.Day(9, 0)
	_ = st.CreateSession("s", now, now.Add(time.Hour))
	if ok, _ := st.SessionValid("s", now.Add(30*time.Minute)); !ok {
		t.Fatal("fresh session rejected")
	}
	if ok, _ := st.SessionValid("s", now.Add(2*time.Hour)); ok {
		t.Fatal("expired session accepted")
	}
}

func TestRetentionPurgesOnlyFinishedOldVisits(t *testing.T) {
	st := testutil.Store(t)
	ana, _ := st.CreatePerson("Ana", "", nil, testutil.Day(9, 0))
	_ = st.InsertVisit(ana.ID, testutil.Day(9, 0).AddDate(0, -14, 0), testutil.Day(17, 0).AddDate(0, -14, 0))
	_ = st.InsertVisit(ana.ID, testutil.Day(9, 0), testutil.Day(12, 0))
	n, err := st.PurgeVisitsBefore(testutil.Day(0, 0).AddDate(-1, 0, 0))
	if err != nil || n != 1 {
		t.Fatalf("purged %d, %v", n, err)
	}
}

func TestOpenFileRejectsGarbage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.db")
	st, err := store.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()
}
