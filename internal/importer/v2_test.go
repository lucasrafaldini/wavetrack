package importer

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/testutil"
)

func TestImportsPeopleAndPairsArrivalsWithDepartures(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v2.db")
	old, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = old.Exec(`
		CREATE TABLE employees (id INTEGER PRIMARY KEY, mac_address TEXT, name TEXT, department TEXT);
		CREATE TABLE events (id INTEGER PRIMARY KEY, event_type TEXT, mac_address TEXT, employee_name TEXT, timestamp TIMESTAMP);
		INSERT INTO employees(mac_address, name, department) VALUES ('AA:BB:CC:00:11:22', 'Ana', 'Design'), ('not a mac', 'Broken', '');
		INSERT INTO events(event_type, mac_address, timestamp) VALUES
			('arrival',   'AA:BB:CC:00:11:22', '2026-10-28 08:30:00.123456-03:00'),
			('departure', 'AA:BB:CC:00:11:22', '2026-10-28 12:00:00-03:00'),
			('arrival',   'AA:BB:CC:00:11:22', '2026-10-28 13:10:00 +0000 UTC m=+1.5'),
			('arrival',   'AA:BB:CC:00:11:22', '2026-10-29 09:00:00'),
			('unknown_device', 'DE:AD:BE:EF:00:01', '2026-10-28 09:00:00');`)
	old.Close()
	if err != nil {
		t.Fatal(err)
	}

	st := testutil.Store(t)
	res, err := V2(path, st, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if res.People != 1 || res.Visits != 1 {
		t.Fatalf("result = %+v, want 1 person and 1 complete visit", res)
	}
	people, _ := st.People(false)
	if people[0].Name != "Ana" || people[0].Team != "Design" || people[0].Devices[0].MAC != "aa:bb:cc:00:11:22" {
		t.Fatalf("person = %+v", people[0])
	}
}

func TestRejectsANonWaveTrackDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "other.db")
	db, _ := sql.Open("sqlite3", path)
	db.Exec(`CREATE TABLE notes (body TEXT)`)
	db.Close()
	if _, err := V2(path, testutil.Store(t), time.Now()); err == nil {
		t.Fatal("imported a database without employees")
	}
}
