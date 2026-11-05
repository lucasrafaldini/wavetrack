# Contributing to WaveTrack

Thanks for helping. A few things keep the project easy to work on.

## Setup

You need Go 1.23+ and libpcap (`sudo apt install libpcap-dev` on Debian/Ubuntu; macOS already has it). Node is only used to syntax-check the dashboard scripts.

```bash
make test    # go test -race ./... and a syntax check of web/assets/*.js
make lint    # go vet and gofmt
make demo    # run against the simulated office at http://localhost:8080
```

The demo is the fastest way to work on the dashboard: it needs no capture permissions and has two weeks of data.

## Guidelines

- **Keep the tracker pure.** `internal/presence` must not do I/O beyond its `Store` interface, and must take time from its inputs or the injected clock, so every case stays testable.
- **Test behavior, especially regressions.** Each fixed bug gets a test that fails without the fix. Run with `-race`.
- **No new runtime dependencies without discussion.** The dashboard has no framework or build step on purpose; keep it that way.
- **Write to the DOM with `textContent`.** Never `innerHTML` with server data. The Content Security Policy forbids inline scripts and styles; set dynamic sizes with `style.setProperty`.
- **Privacy is a feature.** Don't persist devices nobody registered, and don't add outbound network calls.
- **Screenshots** in `docs/screenshots` come from the demo: `swift scripts/screenshot.swift <url> <token> docs/screenshots now=/now …` (macOS).

## Pull requests

Describe the change and how you tested it, link related issues, and keep unrelated changes out. Use [Conventional Commits](https://www.conventionalcommits.org/) for messages (`fix: …`, `feat: …`).
