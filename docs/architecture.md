# WaveTrack architecture

This document explains how WaveTrack 3 is put together and, mostly, why. Each decision lists the alternative it replaced, often the 2.x approach, and what went wrong with it.

## The pipeline

```
          ┌──────────────┐  sightings   ┌──────────────────┐  visits   ┌─────────┐
packets ─▶│   capture    │─────────────▶│ presence tracker │──────────▶│  store  │
          │ live │ demo  │  (MAC, time,  │  state machine   │           │ SQLite  │
          └──────────────┘   signal)     └────────┬─────────┘           └────┬────┘
                                                  │ nearby (memory)          │
                                                  ▼                          ▼
                                          ┌──────────────────────────────────────┐
                                          │ server: JSON API + embedded dashboard │
                                          └──────────────────────────────────────┘
```

| Package | Responsibility |
| --- | --- |
| `capture` | Turns packets into `Sighting{MAC, At, Signal}`. `Live` reads libpcap; the demo package provides a simulated source behind the same interface. |
| `presence` | A state machine from sightings to visits. No I/O besides its store interface; clock injected. |
| `store` | SQLite schema, migrations and queries. |
| `report` | Pure functions from visits to timelines and daily totals. |
| `server` | HTTP API, authentication, invites and the embedded dashboard. |
| `oui`, `netinfo` | Manufacturer lookup; interface choice and ARP lookups. |
| `demo`, `importer` | The simulated office; migration from 2.x. |

## Decisions

### 1. Sightings are emitted continuously, throttled per device

**2.x** sent a device to the tracker only the first time it was seen. The tracker never heard about it again, so after `away_after` every employee was marked as gone, and never came back, even while sitting at their desk.

**3.x** emits a sighting every time a device is seen, at most once per device every 15 seconds. The throttle keeps the tracker's work proportional to the number of devices, not the volume of their traffic. A regression test walks a full working day of sightings and expects one continuous visit.

### 2. Visits, not events

**2.x** stored `arrival` and `departure` events and paired them up at query time. Any missed or duplicated event (a crash, a restart, the bug above) corrupted every report downstream.

**3.x** stores **visits**: one row from the first sighting to the last. A day's hours are the sum of visit overlaps with that day, timelines are the visits themselves, and an open visit (`ended_at IS NULL`) is someone currently in. Arrivals and departures for the feed are derived from visits, so they survive restarts.

### 3. Departures are back-dated to the last sighting

The tracker notices someone has left only after `away_after` without a sighting, 20 minutes by default. Recording the departure at that moment would add 20 phantom minutes to every stay. The visit's end is set to the **last sighting** instead, which is when the person actually left.

### 4. A pure, injectable tracker

The tracker takes sightings and a clock, and talks to storage through a five-method interface. That makes the hard cases cheap to test: restarts in the middle of a visit, stale visits from a crash (closed at their last sighting on start), weak signals, devices reassigned from Nearby, and concurrent use under the race detector. 2.x had none of these tests, and two goroutines shared plain maps, which Go can abort with `concurrent map read and map write`.

### 5. Time is stored as Unix seconds; days are computed in Go

2.x stored Go's string rendering of local times and compared them with SQLite's `date('now','localtime')`: string comparisons across mixed offsets. 3.x stores integers in UTC and computes days with `time.Date(y, m, d, 0, 0, 0, 0, loc)`, so a daylight-saving day is correctly 23 or 25 hours long. The report package has 100% test coverage, including a DST case.

### 6. Unknown devices never touch the disk

Every phone that walks past the office shows up in the capture. Storing those addresses would build a log of strangers. Unregistered devices live only in memory, for the Nearby page, and fall out after twice `away_after`. Only devices someone has deliberately registered are persisted.

### 7. Registration trusts the network, never the client

2.x took the registering phone's MAC address from an `X-MAC-Address` header the client could set, and when that was missing it **assigned the most recently seen device**: someone else's phone. 3.x resolves the address only from the ARP entry of the request's source IP, and refuses when there is none. A test sends a forged header and checks it is ignored.

### 8. Invites are single-use under concurrency

An invite token is 160 random bits; only its SHA-256 is stored. Redeeming it is one transaction: `UPDATE invites SET used_at = ? WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?`, then insert the person and device. If the update touches no row, nothing else happens, so two phones racing on one invite produce exactly one registration, and a failure (for instance, an already-registered phone) rolls back without burning the invite. A test runs eight concurrent redemptions.

### 9. Authentication without accounts

An office presence tool rarely needs user management, but it does need a door. The first start prints an admin token (shown once, stored hashed). Signing in exchanges it for a random session in an `HttpOnly`, `SameSite=Strict` cookie. Requests that change state must be JSON and are rejected when their `Origin` is another site. Every response carries a strict Content Security Policy with no inline scripts or styles, and the dashboard writes server data with `textContent` only. 2.x had no authentication, `Access-Control-Allow-Origin: *`, and inserted registration names with `innerHTML`.

### 10. One binary, no build step

The dashboard is plain HTML, CSS and JavaScript compiled into the binary with `embed`. 2.x served files from a relative `web/` directory, so it only worked when started from the repository. There is no bundler and no framework: about 700 lines of JavaScript, a 28-icon SVG sprite and one stylesheet. Dynamic sizes go through the CSSOM (`style.setProperty`), which the CSP allows, rather than inline `style` attributes, which it blocks.

### 11. A demo that is the same code path

`--demo` replaces only the capture source and the clock. The simulated office uses deterministic schedules (seeded per person and day), seeds two weeks of history through the real store, and leaves current visits open so the real tracker resumes them. The dashboard, API and tracker that run in the demo are the production ones, which makes the demo a credible end-to-end test: CI starts it, signs in with the printed token and checks the API.

### 12. Privacy choices are explicit operations

Removing a person offers **anonymize** (rename to "Former member #n", forget devices, keep hours) or **erase** (delete person, devices and visits via `ON DELETE CASCADE`). Retention deletes finished visits older than a configurable number of days. Consent timestamps are stored for self-registered people.

## Design

The dashboard's job is to answer "who's in?" at a glance, then get out of the way.

- **One idea per page.** Now is a single number with context; Timeline is shape over a day; History is a matrix over weeks.
- **Warm neutrals and one accent.** Off-white paper and deep teal by day, graphite and sea-glass by night. Teal is reserved for presence, so color always means "here".
- **Numbers that line up.** Times and durations use tabular figures; MAC addresses use a monospace face.
- **Avatars without photos.** Initials on a hue derived from the name: distinct, stable and private.
- **Motion with a purpose.** A slow ripple on live indicators, a short rise on dialogs, both disabled under *Reduce Motion*.
- **Explain at the point of doubt.** Nearby explains private addresses where they appear; the registration page says what is and isn't recorded before asking for consent.

## Testing

| Area | What is covered |
| --- | --- |
| Tracker | arrivals, all-day presence, back-dated departures, weak signals, restarts, reassignment, concurrency |
| Reports | midnight clipping, overlap merging, ongoing visits, daily totals, DST |
| Store | migrations, uniqueness, anonymize and erase, invite races, expiry, retention |
| Server | auth on every route, cookie flags, cross-site writes, JSON-only writes, CSP, forged registration headers, CSV formula injection |
| Capture | synthetic ARP, 802.11 and multicast packets, throttling |
| Demo, importer | deterministic schedules, seeding, clock shift; pairing 2.x events |

`go test -race ./...` runs everything; CI runs it on Linux and macOS and ends with the demo smoke test.
