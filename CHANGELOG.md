# Changelog

All notable changes to WaveTrack are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses [Semantic Versioning](https://semver.org/).

## [3.1.0] - Unreleased

Themes, an audit log, a security review and a faster dashboard. The review is in [docs/security.md](docs/security.md), the measurements in [docs/benchmarks.md](docs/benchmarks.md).

### Added
- Five themes, chosen from the top bar: **Light**, **Dark** (or **System**), **Classic** (a Windows 2000 desktop application: bevelled panels, navy title bars, tab strip), **Code** (a code editor's dark chrome) and **High contrast**.
- A status bar: what WaveTrack is capturing, how many people are in, the server time of the last update and the version.
- **Audit log**: every sign-in (including failed ones), change to people and devices, invite, export, token reset and retention run, with actor, IP and target. People appear by number and devices with masked addresses, so erasing someone leaves no personal data in the log. Filters, paging and CSV export; `privacy.audit_retention_days` (730 by default).
- Optional **HTTPS** with `tls.cert_file` and `tls.key_file`, with HSTS and `Secure` cookies; `doctor` checks the certificate and points out plain HTTP on the network.
- **Rate limits** on sign-in (10 a minute per IP) and on invite checks and registrations (30 a minute), with `Retry-After`.
- **Install scripts**: one command on macOS and Linux (`install.sh`) and on Windows (`install.ps1`), both verifying the release checksum.
- **Benchmarks** for the API, the store, the tracker, the reports and packet decoding, over an office of 100 people with a year of history; `make bench`, `make audit`.
- `Cross-Origin-Opener-Policy` and `Cross-Origin-Resource-Policy` headers; read, write and idle timeouts on the HTTP server.
- CI: `govulncheck`, `staticcheck`, one pass of every benchmark, and end-to-end checks of the audit log and the sign-in limit.

### Changed
- The dashboard refresh is 14 times faster with a year of history (5.8 ms to 0.4 ms for `/api/now`), and 40 times faster with ten dashboards open at once: today's visits are found by index instead of by scanning the history, and reads use their own connection pool.
- SQLite runs with `synchronous=NORMAL` in WAL mode, as SQLite recommends.
- The database and its WAL files are created readable by their owner only; the systemd unit is further restricted (kernel, namespaces, address families, `UMask=0077`).
- Moved from the archived `google/gopacket` and unmaintained `gopkg.in/yaml.v3` to their maintained successors (the new packet library models radiotap headers with several namespaces; WaveTrack reads the signal from the first one that reports it); updated `go-sqlite3` and `x/sys`; Go 1.26 or newer, built with Go 1.27.3.
- Expired sessions and invites are purged with the retention job.
- Avatar colors no longer include purples; initials skip numbers and symbols.
- The `/api/status` response includes how many people are in.

### Fixed
- On weekends the demo showed an empty office; it now moves to Friday afternoon.
- The Go toolchain was affected by GO-2026-6809 (`crypto/x509`).

## [3.0.0] - 2026-11-07

A rewrite focused on correctness, privacy and setup. See [docs/architecture.md](docs/architecture.md) for the reasoning behind each change.

### Fixed
- Presence broke after 20 minutes: devices were reported only the first time they were seen, so everyone was marked as gone and never came back.
- Two goroutines shared unsynchronized maps, which could crash the process.
- Self-registration trusted a client-supplied `X-MAC-Address` header and, failing that, assigned the most recently seen device (someone else's phone).
- Names from self-registration were rendered with `innerHTML` in the admin dashboard (stored XSS).
- Invite tokens were predictable timestamps; they are now 160 random bits, stored hashed and single-use.
- The configured interface was ignored whenever any Wi-Fi-looking interface existed; a wired `en0` could be picked silently.
- Time comparisons mixed string formats and offsets in SQLite.
- The dashboard only worked when started from the repository directory.

### Added
- Authentication: an admin token, `HttpOnly`/`SameSite=Strict` sessions, cross-site write protection and a strict Content Security Policy.
- A redesigned dashboard: Now, Timeline, History, People and Nearby, with light and dark themes and a phone layout.
- Visits as the core model, with departures recorded at the last sighting.
- Several devices per person; assigning devices from Nearby.
- Anonymize or erase when removing someone; configurable retention; consent recorded on self-registration.
- CSV export of daily hours, with spreadsheet formula injection defused.
- `--demo`: a simulated office with two weeks of history.
- `wavetrack doctor`, `wavetrack interfaces`, `wavetrack token reset` and `wavetrack import-v2`.
- Capture of IPv6 neighbor discovery and mDNS in addition to ARP and DHCP; optional monitor mode with signal strength.
- Windows support (Npcap) and release binaries for Linux x64/ARM64, macOS and Windows.
- systemd and launchd service files; graceful shutdown.
- Tests for the tracker, reports, store, server, capture, demo and importer; CI on Linux and macOS with an end-to-end demo check.

### Changed
- The database layout is new; use `wavetrack import-v2` to bring 2.x data over.
- Unknown devices are kept in memory only and never written to disk.
- Configuration has defaults for everything; `config.yaml` is optional.

### Removed
- Daily JSON event logs (the database is the record; use the CSV export).
- The separate OUIja vendor API endpoints.

## [1.0.0] - 2025-10-28

### Release Inicial - Production Ready

Esta é a primeira versão completa e estável do WaveTrack, pronta para uso em produção.

### Adicionado

#### Interface Web Completa
- **Sistema de Abas**: Organização em 3 abas principais
 - **Tempo Real**: Monitoramento de dispositivos ativos
 - **Histórico (7 dias)**: Visualização de presença dos últimos 7 dias
 - **Colaboradores**: Gerenciamento completo (CRUD)
- **Organização de Código**: HTML, CSS e JavaScript em arquivos separados
 - `web/index.html` - Estrutura
 - `web/css/style.css` - Estilos
 - `web/js/app.js` - Lógica

#### Gerenciamento de Colaboradores
- **CRUD Completo**: Criar, visualizar, editar e deletar colaboradores
- **Edição Inteligente**:
 - Alteração de MAC com migração automática de histórico
 - Atualização de nome refletida em todos os eventos
 - Edição de departamento e tipo de dispositivo
- **Exclusão com Preservação**: Remover colaborador mantém histórico completo
- **Validações**: Campos obrigatórios e confirmação de deleção

#### Histórico e Relatórios
- **Histórico de 7 Dias**: Visualização completa de presença
 - Horários de chegada e saída por dia
 - Total de horas trabalhadas
 - Percentual de presença (8h = 100%)
 - Média semanal por colaborador
- **Preservação Permanente**: Histórico nunca é deletado
- **Migração Automática**: Eventos seguem o colaborador ao alterar MAC

#### API REST
- `GET /api/employees` - Lista todos os colaboradores
- `POST /api/associate` - Cria/atualiza colaborador (suporta `old_mac_address`)
- `DELETE /api/employees/{mac}` - Remove colaborador (preserva histórico)
- `GET /api/history/7days` - Retorna histórico dos últimos 7 dias
- `GET /api/devices` - Lista dispositivos detectados
- `GET /api/stats` - Estatísticas do sistema

#### Banco de Dados
- **Schema Atualizado**: Sem `ON DELETE CASCADE` para preservar histórico
- **Migração**: `migrate_preserve_history.sql` aplicada
- **Funções de Storage**:
 - `UpdateHistoryMAC()` - Migra histórico ao alterar MAC
 - `UpdateHistoryEmployee()` - Atualiza nome em eventos
 - `DeleteEmployee()` - Remove colaborador preservando eventos

#### Ferramentas de Desenvolvimento
- **Seed History**: Gerador de dados históricos para testes
 - 12 colaboradores fake com padrões realistas
 - 7 dias de eventos de presença
 - Variação de horários e durações
- **Documentação Completa**:
 - `HISTORY_PRESERVATION.md` - Preservação de histórico
 - `SEED_HISTORY.md` - Gerador de dados
 - `TABS_HISTORICO.md` - Sistema de abas
 - `CHANGELOG.md` - Este arquivo

### Alterado

#### Interface
- Dashboard reorganizado em sistema de abas para melhor UX
- Cálculo de percentual ajustado (8 horas = 100%, sem cap)
- Estilos modernizados com melhor responsividade
- Ordenação e filtros aprimorados

#### Backend
- Refatoração de handlers para suportar edição com histórico
- Logs mais detalhados de operações críticas
- Validações aprimoradas em todas as operações

### Removido

#### Arquivos Obsoletos
- `data/employees.json` - Substituído por SQLite
- `web/index_old.html` - Backup não mais necessário
- `internal/api/handlers_old.go.bak` - Backup de desenvolvimento
- `internal/storage/storage_json.go.bak` - Sistema JSON antigo
- `migrate_add_custom_fields.sql` - Migração já aplicada
- `migrate_add_frequency_channel.sql` - Migração já aplicada

#### Funcionalidades
- Sistema de storage JSON (substituído por SQLite)
- Configuração de employees em `config.yaml` (agora via interface)

### Corrigido

- **Percentual de Presença**: Corrigido para usar 8 horas como base (não mais 9h)
- **Limite de 100%**: Removido cap para mostrar horas extras corretamente
- **MAC Readonly**: Campo MAC agora editável em modo de edição
- **Duplicação de Eventos**: Event listener duplicado removido
- **Cascade Delete**: Histórico não é mais deletado ao remover colaborador

### Segurança

- Confirmação obrigatória antes de deletar colaborador
- Validação de campos obrigatórios em todas as operações
- Logs de auditoria para alterações de MAC e nome

### Estatísticas da Release

- **Linhas de Código**: ~3.500 linhas (Go + JS + CSS + HTML)
- **Arquivos**: 25+ arquivos de código
- **Endpoints API**: 6 endpoints REST
- **Cobertura de Testes**: Storage layer testado
- **Documentação**: 8 arquivos markdown (~2.500 linhas)

### Próximos Passos (v1.1)

- Relatórios em PDF/CSV
- Exportação de dados históricos
- Gráficos e dashboards analíticos
- Notificações por webhook/email
- Autenticação e controle de acesso

---

## [0.x.x] - Desenvolvimento

Versões anteriores foram de desenvolvimento interno e não foram documentadas formalmente.

---

**Formato do Changelog**: [Keep a Changelog](https://keepachangelog.com/pt-BR/1.0.0/)
**Versionamento**: [Semantic Versioning](https://semver.org/lang/pt-BR/)
