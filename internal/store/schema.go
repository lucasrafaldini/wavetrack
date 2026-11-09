package store

// Each entry migrates the schema one version forward. Never edit a shipped entry; append a new one.
var migrations = []string{
	// 1: initial schema
	`
	CREATE TABLE meta (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE people (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		name         TEXT    NOT NULL,
		team         TEXT    NOT NULL DEFAULT '',
		created_at   INTEGER NOT NULL,
		consented_at INTEGER,          -- set when the person registered through an invite
		archived_at  INTEGER           -- anonymized: name replaced, devices removed, visits kept
	);

	CREATE TABLE devices (
		mac       TEXT PRIMARY KEY,
		person_id INTEGER NOT NULL REFERENCES people(id) ON DELETE CASCADE,
		label     TEXT    NOT NULL DEFAULT '',
		vendor    TEXT    NOT NULL DEFAULT '',
		kind      TEXT    NOT NULL DEFAULT 'unknown',
		added_at  INTEGER NOT NULL,
		last_seen INTEGER
	);
	CREATE INDEX devices_person ON devices(person_id);

	-- One row per stay: from the first sighting until the person goes unseen for presence.away_after.
	CREATE TABLE visits (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		person_id  INTEGER NOT NULL REFERENCES people(id) ON DELETE CASCADE,
		started_at INTEGER NOT NULL,
		last_seen  INTEGER NOT NULL,
		ended_at   INTEGER
	);
	CREATE INDEX visits_person_started ON visits(person_id, started_at);
	CREATE INDEX visits_started ON visits(started_at);
	CREATE INDEX visits_open ON visits(ended_at) WHERE ended_at IS NULL;

	-- Tokens are stored hashed; the plain value only ever exists in the invite link or cookie.
	CREATE TABLE invites (
		token_hash TEXT PRIMARY KEY,
		created_at INTEGER NOT NULL,
		expires_at INTEGER NOT NULL,
		used_at    INTEGER
	);

	CREATE TABLE sessions (
		token_hash TEXT PRIMARY KEY,
		created_at INTEGER NOT NULL,
		expires_at INTEGER NOT NULL
	);
	`,

	// 2: today's visits without scanning the whole history (see VisitsBetween). ended_at is
	// written once per visit, so unlike last_seen it costs nothing on the tracker's updates.
	`
	CREATE INDEX visits_ended ON visits(ended_at);
	`,

	// 3: audit log of administrative actions
	`
	CREATE TABLE audit (
		id     INTEGER PRIMARY KEY AUTOINCREMENT,
		at     INTEGER NOT NULL,
		actor  TEXT    NOT NULL,           -- "admin", "invite", "system" or "cli"
		ip     TEXT    NOT NULL DEFAULT '',
		action TEXT    NOT NULL,           -- e.g. "person.erase"
		target TEXT    NOT NULL DEFAULT '', -- whom or what it was done to
		detail TEXT    NOT NULL DEFAULT ''
	);
	CREATE INDEX audit_at ON audit(at);
	`,
}
