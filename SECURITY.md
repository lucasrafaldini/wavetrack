# Security policy

## Reporting a vulnerability

Please report security problems privately, through GitHub's [private vulnerability reporting](https://github.com/lucasrafaldini/wavetrack/security/advisories/new). Don't open a public issue.

Include what you found, how to reproduce it and what an attacker could do with it. You'll get an answer within a week. Fixes are released as soon as they're ready and credited to you unless you prefer otherwise.

## Supported versions

| Version | Supported |
| --- | --- |
| 3.1.x | Yes |
| 3.0.x | Security fixes until 3.2 |
| 2.x and older | No; [upgrade](README.md#13-upgrading-from-2x) |

## What's in scope

WaveTrack itself: the server, the dashboard, the registration flow, the installer scripts and the service files in `deploy/`. The threat model, the controls and the last review's findings are in [docs/security.md](docs/security.md).
