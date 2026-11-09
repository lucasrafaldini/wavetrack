# Security review

This document records the security review done for WaveTrack 3.1: what WaveTrack protects, from whom, what was found and what was done about it. It is meant to be read by whoever decides whether to run WaveTrack in their office. To report a vulnerability, see [SECURITY.md](../SECURITY.md).

## What is at stake

| Asset | Why it matters |
| --- | --- |
| Presence history | Who was in the office, and when. Personal data under LGPD and GDPR. |
| Admin access | Reads the history, adds and removes people, exports everything. |
| The registration flow | Decides which device counts as which person. |
| The host | WaveTrack needs packet-capture rights, which are powerful. |

## Who it is protected from

| Actor | Can | Must not be able to |
| --- | --- | --- |
| Anyone on the office Wi-Fi | reach the dashboard's sign-in page and invite links | read presence data, sign in by guessing, flood the server, or see the admin token on the wire |
| A person with an invite | register **their own** device once | register someone else's device, reuse the invite |
| Another user on the WaveTrack host | use the machine | read the database |
| A passer-by's phone | be seen briefly | be stored |
| An administrator | manage everything | do it without leaving a trace |

## Controls

**Authentication.** One admin token, 160 random bits, shown once and stored as a SHA-256 hash. Signing in exchanges it for a session cookie (`HttpOnly`, `SameSite=Strict`, `Secure` under TLS) whose hash is stored server-side. Resetting the token from the command line signs every browser out.

**Requests.** State-changing requests must be JSON and same-origin; bodies are capped at 64 KB and unknown fields are rejected. Every response carries a Content Security Policy without inline scripts or styles, `frame-ancestors 'none'`, `nosniff`, `no-referrer`, and cross-origin opener and resource policies. The dashboard writes server data with `textContent` only. CSV exports defuse spreadsheet formulas.

**Registration.** The device's address comes only from the ARP entry of the request's source IP, never from anything the client sends. Invites are single-use under concurrency (one conditional `UPDATE` in a transaction).

**Rate limits.** Per client IP, as a token bucket: 10 sign-in attempts per minute, 30 invite checks or registrations per minute. Over the limit, `429 Too Many Requests` with `Retry-After`. `X-Forwarded-For` is ignored, because clients can set it.

**Transport.** Optional TLS (`tls.cert_file`, `tls.key_file`), with HTTP/2 and `Strict-Transport-Security`. `wavetrack doctor` points out when the dashboard is served over plain HTTP on the network.

**Timeouts.** Headers 10 s, whole request 30 s, response 2 min (room for a year-long export), idle connections 2 min, headers capped at 64 KB.

**Data at rest.** The database and its WAL files are created `0600`. The systemd unit adds `UMask=0077`.

**Privacy by default.** Unregistered devices are kept in memory only. Visits are deleted after `privacy.retention_days`; expired sessions and invites are purged with them.

**Accountability.** An [audit log](#the-audit-log) records every administrative action.

**Dependencies.** Five direct dependencies: the packet library, SQLite, YAML, the manufacturer lookup and a QR encoder. `govulncheck` runs in CI on every push and reports reachable vulnerabilities only.

## Findings

Severity follows the impact on the assets above, given how WaveTrack is deployed (one office network, one host).

| # | Finding | Severity | Status |
| --- | --- | --- | --- |
| S1 | The dashboard was served only over plain HTTP. On a shared Wi-Fi network, the admin token and session cookie could be read by anyone capturing traffic. | High | **Fixed.** Optional TLS with HSTS; the cookie becomes `Secure`; `doctor` warns about plain HTTP on the network. |
| S2 | The HTTP server bounded only the time to read headers. A slow client could hold connections open indefinitely. | Medium | **Fixed.** Read, write and idle timeouts; header size limit. |
| S3 | Administrative actions left no record. After an erasure or an export there was no way to tell who did it, or when. | Medium | **Fixed.** Audit log, kept for `privacy.audit_retention_days` (730 by default). |
| S4 | The database was created with the process umask, usually `0644`: readable by every user on the host. | Medium | **Fixed.** `0600` for the database, WAL and shared-memory files; `UMask=0077` in the service. |
| S5 | Two dependencies were abandoned: `google/gopacket` (archived since 2020) and `gopkg.in/yaml.v3` (marked unmaintained). `golang.org/x/sys` dated from 2022. | Medium | **Fixed.** Moved to `gopacket/gopacket` and `go.yaml.in/yaml/v3`, the maintained successors; everything updated. |
| S6 | The pinned Go toolchain was affected by GO-2026-6809 (`crypto/x509`), reachable through `crypto/rand`. | Medium | **Fixed.** Toolchain go1.27.3. |
| S7 | Failed sign-ins slept 400 ms inside the handler. Attempts were not limited, parallel guesses all went through, and each held a goroutine. Guessing a 160-bit token is hopeless, but the log and the database could be flooded. | Low | **Fixed.** Per-IP rate limits on every unauthenticated endpoint. |
| S8 | Expired sessions and used or expired invites were never deleted. | Low | **Fixed.** Purged every six hours. |
| S9 | No cross-origin isolation headers. | Low | **Fixed.** `Cross-Origin-Opener-Policy` and `Cross-Origin-Resource-Policy: same-origin`. |
| S10 | The systemd unit allowed more than capture needs. | Low | **Fixed.** Kernel, namespace and device protections; address families limited to IP, Unix, packet and netlink sockets. |

### Accepted risks

| # | Risk | Why it stays |
| --- | --- | --- |
| A1 | One shared admin token, no per-user accounts. | Most offices have one or two administrators. The audit log tells sessions apart (`admin:3f9a1c`), and resetting the token revokes every session. Per-user accounts can come later without changing the data model. |
| A2 | The first-run token is printed to the terminal, so under systemd it lands in the journal. | Shown once is the simplest safe onboarding. Whoever can read the journal can already read the service's environment. Run `wavetrack token reset` after the first sign-in if the journal is shared. |
| A3 | Registration trusts the network's ARP table. An attacker on the same network who spoofs ARP could get their device registered under an invite they hold. | They can only register a device they control, under an invite given to them. The audit log records every registration. |
| A4 | Behind a reverse proxy, every client shares the proxy's IP for rate limiting. | Registration needs the phone on the same network segment anyway, so WaveTrack is meant to be reached directly. `X-Forwarded-For` is deliberately not trusted. |
| A5 | The IEEE manufacturer list is downloaded at runtime, over HTTPS. | It only labels devices; if the download fails or is tampered with, labels read "Unknown". No MAC address is ever sent out. |
| A6 | `skip2/go-qrcode` hasn't changed since 2020. | It is small, pure Go, and only ever encodes the invite URL the server builds itself, never anything a client sends. `govulncheck` watches it like the rest. |

## The audit log

The **Audit** page, `GET /api/audit` and `GET /api/audit.csv` list what administrators did: sign-ins (including failed ones), sign-outs, people added, edited, anonymized or erased, devices added or removed, invites created and redeemed, exports, token resets and retention runs. Each entry has the time, the actor (an admin session, an invite link, the command line or WaveTrack itself), the client IP, the target and a short detail.

The log never stores names or full MAC addresses. People appear as `person #12`; devices keep their manufacturer prefix and last octet (`a4:83:e7:••:••:10`). Erasing someone therefore leaves no personal data behind in the log, while the fact that an erasure happened, by whom and when, remains: the record a data-protection request needs. The page shows the current name next to the number for people who still exist.

## How this was verified

- `go test -race ./...` covers the controls above: authentication on every route including the audit log, cookie flags, cross-site writes, JSON-only writes, headers, forged registration headers, rate limiting and refill, audit entries for every action with no names or full addresses in them, database permissions, purging, TLS configuration.
- The demo was run over TLS with a self-signed certificate: HTTP/2, HSTS and all headers present, plain HTTP refused, database files `0600`.
- `staticcheck ./...`: no findings. `govulncheck ./...`: no vulnerabilities.
- CI runs the tests on Linux, macOS and Windows, `govulncheck`, `staticcheck`, and an end-to-end check against the demo that includes the audit log and the sign-in rate limit.

Not verified: the hardened systemd unit on a live Linux host, and the Windows installer script, which was written without a Windows machine at hand.
