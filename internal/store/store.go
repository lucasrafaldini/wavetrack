// Package store keeps people, devices and visits in SQLite.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrDeviceTaken = errors.New("device already belongs to someone")
)

// Store writes through one connection, which serializes writers and rules out
// SQLITE_BUSY, and reads through a pool: in WAL mode readers never wait for the writer,
// so many dashboards refreshing at once are served in parallel.
type Store struct {
	db  *sql.DB // writes, transactions and migrations
	rdb *sql.DB // reads
}

type Person struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Team        string     `json:"team"`
	CreatedAt   time.Time  `json:"created_at"`
	ConsentedAt *time.Time `json:"consented_at,omitempty"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
	Devices     []Device   `json:"devices"`
}

type Device struct {
	MAC      string     `json:"mac"`
	PersonID int64      `json:"person_id"`
	Label    string     `json:"label"`
	Vendor   string     `json:"vendor"`
	Kind     string     `json:"kind"`
	AddedAt  time.Time  `json:"added_at"`
	LastSeen *time.Time `json:"last_seen,omitempty"`
}

type Visit struct {
	ID        int64
	PersonID  int64
	StartedAt time.Time
	LastSeen  time.Time
	EndedAt   *time.Time
}

// Open creates dir if needed and opens (and migrates) dir/wavetrack.db.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	return OpenFile(filepath.Join(dir, "wavetrack.db"))
}

func OpenFile(path string) (*Store, error) {
	// The database holds who was where and when: readable by its owner only. SQLite
	// gives the -wal and -shm files the same permissions as the database.
	if f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600); err == nil {
		f.Close()
		_ = os.Chmod(path, 0o600)
	}
	// synchronous=NORMAL is the recommended setting in WAL mode: a power cut can lose the
	// last commits but never corrupts the database, and commits skip a sync.
	common := "file:" + path + "?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL"
	db, err := sql.Open("sqlite3", common+"&_txlock=immediate")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating %s: %w", path, err)
	}
	rdb, err := sql.Open("sqlite3", common+"&_query_only=true")
	if err != nil {
		db.Close()
		return nil, err
	}
	readers := max(4, runtime.NumCPU())
	rdb.SetMaxOpenConns(readers)
	rdb.SetMaxIdleConns(readers)
	s.rdb = rdb
	return s, nil
}

func (s *Store) Close() error { return errors.Join(s.rdb.Close(), s.db.Close()) }

func (s *Store) migrate() error {
	var version int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	for ; version < len(migrations); version++ {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migrations[version]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", version+1, err)
		}
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, version+1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func unix(t time.Time) int64 { return t.Unix() }

func fromUnix(v sql.NullInt64) *time.Time {
	if !v.Valid {
		return nil
	}
	t := time.Unix(v.Int64, 0)
	return &t
}

// NormalizeMAC parses any common MAC notation into lowercase colon form.
func NormalizeMAC(raw string) (string, error) {
	mac, err := net.ParseMAC(strings.TrimSpace(raw))
	if err != nil || len(mac) != 6 {
		return "", fmt.Errorf("%q isn't a valid MAC address", raw)
	}
	return mac.String(), nil
}

// MARK: meta

func (s *Store) Meta(key string) (string, error) {
	var value string
	err := s.rdb.QueryRow(`SELECT value FROM meta WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return value, err
}

func (s *Store) SetMeta(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO meta(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// MARK: people

func (s *Store) CreatePerson(name, team string, consentedAt *time.Time, now time.Time) (Person, error) {
	var consent any
	if consentedAt != nil {
		consent = unix(*consentedAt)
	}
	res, err := s.db.Exec(`INSERT INTO people(name, team, created_at, consented_at) VALUES(?, ?, ?, ?)`,
		strings.TrimSpace(name), strings.TrimSpace(team), unix(now), consent)
	if err != nil {
		return Person{}, err
	}
	id, _ := res.LastInsertId()
	return s.Person(id)
}

func (s *Store) UpdatePerson(id int64, name, team string) error {
	res, err := s.db.Exec(`UPDATE people SET name = ?, team = ? WHERE id = ? AND archived_at IS NULL`,
		strings.TrimSpace(name), strings.TrimSpace(team), id)
	return expectOne(res, err)
}

func (s *Store) Person(id int64) (Person, error) {
	people, err := s.queryPeople(`WHERE p.id = ?`, id)
	if err != nil {
		return Person{}, err
	}
	if len(people) == 0 {
		return Person{}, ErrNotFound
	}
	return people[0], nil
}

// People lists everyone, optionally including anonymized former members.
func (s *Store) People(includeArchived bool) ([]Person, error) {
	if includeArchived {
		return s.queryPeople(``)
	}
	return s.queryPeople(`WHERE p.archived_at IS NULL`)
}

func (s *Store) queryPeople(where string, args ...any) ([]Person, error) {
	rows, err := s.rdb.Query(`SELECT p.id, p.name, p.team, p.created_at, p.consented_at, p.archived_at
		FROM people p `+where+` ORDER BY p.name COLLATE NOCASE`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var people []Person
	index := map[int64]int{}
	for rows.Next() {
		var p Person
		var created int64
		var consented, archived sql.NullInt64
		if err := rows.Scan(&p.ID, &p.Name, &p.Team, &created, &consented, &archived); err != nil {
			return nil, err
		}
		p.CreatedAt = time.Unix(created, 0)
		p.ConsentedAt = fromUnix(consented)
		p.ArchivedAt = fromUnix(archived)
		p.Devices = []Device{}
		index[p.ID] = len(people)
		people = append(people, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	devices, err := s.Devices()
	if err != nil {
		return nil, err
	}
	for _, d := range devices {
		if i, ok := index[d.PersonID]; ok {
			people[i].Devices = append(people[i].Devices, d)
		}
	}
	return people, nil
}

// ArchivePerson anonymizes someone: their name is replaced and devices forgotten,
// but past visits stay so attendance totals remain accurate.
func (s *Store) ArchivePerson(id int64, now time.Time) error {
	return s.tx(func(tx *sql.Tx) error {
		res, err := tx.Exec(`UPDATE people SET name = ?, team = '', archived_at = ? WHERE id = ? AND archived_at IS NULL`,
			fmt.Sprintf("Former member #%d", id), unix(now), id)
		if err := expectOne(res, err); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM devices WHERE person_id = ?`, id); err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE visits SET ended_at = last_seen WHERE person_id = ? AND ended_at IS NULL`, id)
		return err
	})
}

// ErasePerson deletes a person with all their devices and visits.
func (s *Store) ErasePerson(id int64) error {
	res, err := s.db.Exec(`DELETE FROM people WHERE id = ?`, id)
	return expectOne(res, err)
}

// MARK: devices

func (s *Store) AddDevice(d Device) error {
	mac, err := NormalizeMAC(d.MAC)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO devices(mac, person_id, label, vendor, kind, added_at) VALUES(?, ?, ?, ?, ?, ?)`,
		mac, d.PersonID, strings.TrimSpace(d.Label), d.Vendor, d.Kind, unix(d.AddedAt))
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return ErrDeviceTaken
	}
	if err != nil && strings.Contains(err.Error(), "FOREIGN KEY") {
		return ErrNotFound
	}
	return err
}

func (s *Store) RemoveDevice(mac string) error {
	normalized, err := NormalizeMAC(mac)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(`DELETE FROM devices WHERE mac = ?`, normalized)
	return expectOne(res, err)
}

func (s *Store) Devices() ([]Device, error) {
	rows, err := s.rdb.Query(`SELECT mac, person_id, label, vendor, kind, added_at, last_seen FROM devices ORDER BY added_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var devices []Device
	for rows.Next() {
		var d Device
		var added int64
		var seen sql.NullInt64
		if err := rows.Scan(&d.MAC, &d.PersonID, &d.Label, &d.Vendor, &d.Kind, &added, &seen); err != nil {
			return nil, err
		}
		d.AddedAt = time.Unix(added, 0)
		d.LastSeen = fromUnix(seen)
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func (s *Store) TouchDevice(mac string, at time.Time) error {
	_, err := s.db.Exec(`UPDATE devices SET last_seen = ? WHERE mac = ?`, unix(at), mac)
	return err
}

// MARK: visits

func (s *Store) StartVisit(personID int64, at time.Time) (int64, error) {
	res, err := s.db.Exec(`INSERT INTO visits(person_id, started_at, last_seen) VALUES(?, ?, ?)`, personID, unix(at), unix(at))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) TouchVisit(id int64, at time.Time) error {
	_, err := s.db.Exec(`UPDATE visits SET last_seen = MAX(last_seen, ?) WHERE id = ?`, unix(at), id)
	return err
}

func (s *Store) EndVisit(id int64, at time.Time) error {
	_, err := s.db.Exec(`UPDATE visits SET ended_at = ?, last_seen = MAX(last_seen, ?) WHERE id = ?`, unix(at), unix(at), id)
	return err
}

func (s *Store) OpenVisits() ([]Visit, error) {
	return s.queryVisits(`WHERE ended_at IS NULL`)
}

// VisitsBetween returns visits overlapping [from, to): those that started before `to` and
// were still going on at `from`.
//
// Left to itself, SQLite filters on visits_started, which reads the whole history before
// `to`: a year of it on every dashboard refresh. Instead, finished visits are found by
// their end (visits_ended) and open ones through the partial index of open visits, so
// the cost follows the size of the window, not of the history.
func (s *Store) VisitsBetween(from, to time.Time) ([]Visit, error) {
	f, t := unix(from), unix(to)
	rows, err := s.rdb.Query(`
		SELECT id, person_id, started_at, last_seen, ended_at FROM visits INDEXED BY visits_ended
		 WHERE ended_at >= ? AND started_at < ?
		UNION ALL
		SELECT id, person_id, started_at, last_seen, ended_at FROM visits INDEXED BY visits_open
		 WHERE ended_at IS NULL AND last_seen >= ? AND started_at < ?
		ORDER BY started_at`, f, t, f, t)
	if err != nil {
		return nil, err
	}
	return scanVisits(rows)
}

func (s *Store) queryVisits(where string, args ...any) ([]Visit, error) {
	rows, err := s.rdb.Query(`SELECT id, person_id, started_at, last_seen, ended_at FROM visits `+where+` ORDER BY started_at`, args...)
	if err != nil {
		return nil, err
	}
	return scanVisits(rows)
}

func scanVisits(rows *sql.Rows) ([]Visit, error) {
	defer rows.Close()
	var visits []Visit
	for rows.Next() {
		var v Visit
		var started, seen int64
		var ended sql.NullInt64
		if err := rows.Scan(&v.ID, &v.PersonID, &started, &seen, &ended); err != nil {
			return nil, err
		}
		v.StartedAt = time.Unix(started, 0)
		v.LastSeen = time.Unix(seen, 0)
		v.EndedAt = fromUnix(ended)
		visits = append(visits, v)
	}
	return visits, rows.Err()
}

// InsertVisit writes a complete visit. Used by the demo and the v2 importer.
func (s *Store) InsertVisit(personID int64, start, end time.Time) error {
	_, err := s.db.Exec(`INSERT INTO visits(person_id, started_at, last_seen, ended_at) VALUES(?, ?, ?, ?)`,
		personID, unix(start), unix(end), unix(end))
	return err
}

// InsertVisits writes many visits in one transaction. An open visit has a nil EndedAt.
func (s *Store) InsertVisits(visits []Visit) error {
	return s.tx(func(tx *sql.Tx) error {
		stmt, err := tx.Prepare(`INSERT INTO visits(person_id, started_at, last_seen, ended_at) VALUES(?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, v := range visits {
			var ended any
			if v.EndedAt != nil {
				ended = unix(*v.EndedAt)
			}
			if _, err := stmt.Exec(v.PersonID, unix(v.StartedAt), unix(v.LastSeen), ended); err != nil {
				return err
			}
		}
		return nil
	})
}

// PurgeVisitsBefore deletes finished visits that ended before cutoff.
func (s *Store) PurgeVisitsBefore(cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM visits WHERE ended_at IS NOT NULL AND ended_at < ?`, unix(cutoff))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// MARK: invites and sessions

func (s *Store) CreateInvite(tokenHash string, now, expires time.Time) error {
	_, err := s.db.Exec(`INSERT INTO invites(token_hash, created_at, expires_at) VALUES(?, ?, ?)`, tokenHash, unix(now), unix(expires))
	return err
}

func (s *Store) InviteValid(tokenHash string, now time.Time) (bool, error) {
	var n int
	err := s.rdb.QueryRow(`SELECT COUNT(*) FROM invites WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?`,
		tokenHash, unix(now)).Scan(&n)
	return n == 1, err
}

// RedeemInvite registers a person through an invite in one transaction. The invite can
// only be used once, even if two phones submit at the same moment.
func (s *Store) RedeemInvite(tokenHash string, now time.Time, name, team string, device Device) (Person, error) {
	var personID int64
	err := s.tx(func(tx *sql.Tx) error {
		res, err := tx.Exec(`UPDATE invites SET used_at = ? WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?`,
			unix(now), tokenHash, unix(now))
		if err := expectOne(res, err); err != nil {
			return ErrNotFound
		}
		mac, err := NormalizeMAC(device.MAC)
		if err != nil {
			return err
		}
		var taken int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM devices WHERE mac = ?`, mac).Scan(&taken); err != nil {
			return err
		}
		if taken > 0 {
			return ErrDeviceTaken
		}
		res, err = tx.Exec(`INSERT INTO people(name, team, created_at, consented_at) VALUES(?, ?, ?, ?)`,
			strings.TrimSpace(name), strings.TrimSpace(team), unix(now), unix(now))
		if err != nil {
			return err
		}
		personID, _ = res.LastInsertId()
		_, err = tx.Exec(`INSERT INTO devices(mac, person_id, label, vendor, kind, added_at, last_seen) VALUES(?, ?, ?, ?, ?, ?, ?)`,
			mac, personID, strings.TrimSpace(device.Label), device.Vendor, device.Kind, unix(now), unix(now))
		return err
	})
	if err != nil {
		return Person{}, err
	}
	return s.Person(personID)
}

func (s *Store) CreateSession(tokenHash string, now, expires time.Time) error {
	if _, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, unix(now)); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO sessions(token_hash, created_at, expires_at) VALUES(?, ?, ?)`, tokenHash, unix(now), unix(expires))
	return err
}

func (s *Store) SessionValid(tokenHash string, now time.Time) (bool, error) {
	var n int
	err := s.rdb.QueryRow(`SELECT COUNT(*) FROM sessions WHERE token_hash = ? AND expires_at > ?`, tokenHash, unix(now)).Scan(&n)
	return n == 1, err
}

// PurgeExpired deletes sessions and invites that can no longer be used.
func (s *Store) PurgeExpired(now time.Time) error {
	return s.tx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, unix(now)); err != nil {
			return err
		}
		_, err := tx.Exec(`DELETE FROM invites WHERE expires_at <= ? OR used_at IS NOT NULL`, unix(now))
		return err
	})
}

func (s *Store) DeleteSession(tokenHash string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}

func (s *Store) DeleteAllSessions() error {
	_, err := s.db.Exec(`DELETE FROM sessions`)
	return err
}

// MARK: helpers

func (s *Store) tx(fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func expectOne(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// MARK: audit

// AuditEntry records one administrative action: who did what, to whom, from where.
type AuditEntry struct {
	ID     int64     `json:"id"`
	At     time.Time `json:"at"`
	Actor  string    `json:"actor"`
	IP     string    `json:"ip"`
	Action string    `json:"action"`
	Target string    `json:"target"`
	Detail string    `json:"detail"`
}

func (s *Store) Audit(e AuditEntry) error {
	_, err := s.db.Exec(`INSERT INTO audit(at, actor, ip, action, target, detail) VALUES(?, ?, ?, ?, ?, ?)`,
		unix(e.At), e.Actor, e.IP, e.Action, e.Target, e.Detail)
	return err
}

// AuditLog returns entries newest first. A beforeID of 0 starts at the newest; pass the
// last ID of a page to get the next one. An action ending in "." matches a whole group,
// e.g. "person." for every change to people.
func (s *Store) AuditLog(beforeID int64, limit int, action string) ([]AuditEntry, error) {
	where, args := []string{}, []any{}
	if beforeID > 0 {
		where, args = append(where, "id < ?"), append(args, beforeID)
	}
	switch {
	case strings.HasSuffix(action, "."):
		where, args = append(where, "action LIKE ? ESCAPE '\\'"), append(args, strings.ReplaceAll(action, "_", `\_`)+"%")
	case action != "":
		where, args = append(where, "action = ?"), append(args, action)
	}
	query := `SELECT id, at, actor, ip, action, target, detail FROM audit`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	rows, err := s.rdb.Query(query+` ORDER BY id DESC LIMIT ?`, append(args, limit)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		var at int64
		if err := rows.Scan(&e.ID, &at, &e.Actor, &e.IP, &e.Action, &e.Target, &e.Detail); err != nil {
			return nil, err
		}
		e.At = time.Unix(at, 0)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// PurgeAuditBefore deletes audit entries older than cutoff.
func (s *Store) PurgeAuditBefore(cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM audit WHERE at < ?`, unix(cutoff))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
