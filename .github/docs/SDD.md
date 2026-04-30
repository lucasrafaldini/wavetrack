# WaveTrack - Software Design Document

**Versão:** 2.0
**Autor:** Lucas Rafaldini
**Licença:** MIT
**Última atualização:** Abril 2026

---

## 1. Introdução

### 1.1 Propósito

Este documento descreve a arquitetura, os componentes e as decisões de design do WaveTrack, um sistema de monitoramento de presença baseado em detecção de dispositivos Wi-Fi.

### 1.2 Escopo

O WaveTrack captura pacotes de gerenciamento Wi-Fi (probe requests, beacons) para detectar dispositivos próximos, associá-los a colaboradores cadastrados e registrar automaticamente eventos de presença (chegada/saída).

O sistema inclui:
- Scanner Wi-Fi com suporte a modo monitor (802.11) e modo compatibilidade (ARP)
- API REST para gerenciamento de colaboradores e consulta de dados
- Interface web (SPA) para visualização em tempo real e histórico
- Identificação de fabricantes via biblioteca OUIja (base IEEE oficial)
- Banco de dados SQLite para persistência de histórico

### 1.3 Glossário

| Termo | Definição |
|---|---|
| OUI | Organizationally Unique Identifier - primeiros 3 octetos de um MAC address |
| OUIja | Biblioteca Go para consulta de fabricantes via base IEEE/Wireshark |
| RSSI | Received Signal Strength Indicator - força do sinal em dBm |
| BPF | Berkeley Packet Filter - filtro aplicado na captura de pacotes |
| Probe Request | Pacote 802.11 enviado por dispositivos buscando redes Wi-Fi |

### 1.4 Licenciamento

- **WaveTrack**: MIT License
- **Dependências Go**: gopacket (BSD-3), yaml.v3 (Apache-2.0/MIT), go-sqlite3 (MIT), ouija (MIT)
- **Frontend**: Vanilla JS/CSS, sem dependências externas
- **Privacidade**: MAC addresses são dados pessoais (LGPD/GDPR). Implementar anonimização e obter consentimento em ambientes de produção.

---

## 2. Visão Geral do Sistema

### 2.1 Contexto

O WaveTrack opera em um ponto de acesso Wi-Fi (ou dispositivo com antena Wi-Fi) capturando passivamente pacotes de gerenciamento para detectar a presença de dispositivos. Gestores acessam o dashboard via navegador para monitorar presença e gerenciar colaboradores.

### 2.2 Requisitos Funcionais

| ID | Requisito | Status |
|---|---|---|
| RF-01 | Capturar pacotes Wi-Fi e extrair MAC addresses | Implementado |
| RF-02 | Identificar fabricante do dispositivo via OUI | Implementado (OUIja) |
| RF-03 | Cadastrar/editar/excluir colaboradores (CRUD) | Implementado |
| RF-04 | Detectar chegada e saída automaticamente | Implementado |
| RF-05 | Exibir dashboard em tempo real | Implementado |
| RF-06 | Histórico de presença de 7 dias | Implementado |
| RF-07 | Preservar histórico ao excluir colaboradores | Implementado |
| RF-08 | Migrar histórico ao trocar MAC de colaborador | Implementado |
| RF-09 | Campos customizados (tipo dispositivo, fabricante) | Implementado |
| RF-10 | API de consulta de fabricantes (OUIja) | Implementado |

### 2.3 Requisitos Não-Funcionais

| Requisito | Meta |
|---|---|
| Tempo de resposta API | < 100ms |
| Carregamento do dashboard | < 500ms |
| Atualização de dados | A cada 5 segundos |
| Uso de memória (runtime) | 10-20 MB |
| Uso de memória (base OUI) | 2-5 MB |
| Dependências frontend | Zero (vanilla JS/CSS) |
| Compatibilidade | Linux, macOS |

---

## 3. Arquitetura

### 3.1 Visão Geral

```
                    ┌──────────────┐
                    │   Browser    │
                    │  (Dashboard) │
                    └──────┬───────┘
                           │ HTTP
                    ┌──────┴───────┐
                    │   API REST   │
                    │  (net/http)  │
                    └──────┬───────┘
              ┌────────────┼────────────┐
              │            │            │
       ┌──────┴──────┐ ┌──┴───┐ ┌──────┴──────┐
       │   Tracker   │ │ OUIja│ │   Storage   │
       │ (presença)  │ │(IEEE)│ │  (SQLite)   │
       └──────┬──────┘ └──────┘ └─────────────┘
              │
       ┌──────┴──────┐
       │   Scanner   │
       │  (gopacket) │
       └──────┬──────┘
              │
       ┌──────┴──────┐
       │   libpcap   │
       │ (802.11/ARP)│
       └─────────────┘
```

### 3.2 Componentes

```
wavetrack/
├── cmd/
│   ├── wavetrack/          # Aplicação principal
│   │   └── main.go         # Entrypoint, wiring de componentes
│   └── seed_history/       # Ferramenta de seed para desenvolvimento
│       └── main.go
├── internal/
│   ├── api/                # Handlers REST e rotas
│   │   └── handlers.go
│   ├── config/             # Leitura de configuração YAML
│   │   └── config.go
│   ├── deviceid/           # Identificação de fabricantes (OUIja + inferência)
│   │   ├── deviceid.go
│   │   └── deviceid_test.go
│   ├── logger/             # Sistema de logs com rotação diária (JSON)
│   │   └── logger.go
│   ├── models/             # Structs de domínio
│   │   └── models.go
│   ├── storage/            # Persistência SQLite
│   │   ├── storage.go
│   │   ├── storage_test.go
│   │   └── schema.sql
│   ├── tracker/            # Lógica de detecção de presença
│   │   └── tracker.go
│   └── wifi/               # Scanner de pacotes Wi-Fi
│       └── scanner.go
├── web/                    # Frontend SPA (HTML, CSS, JS)
│   ├── index.html
│   ├── styles.css
│   └── app.js
├── config.yaml             # Configuração padrão
└── data/                   # Diretório de dados (runtime)
    └── wavetrack.db        # Banco SQLite
```

### 3.3 Fluxo de Dados

1. **Captura**: Scanner em modo promíscuo detecta pacotes Wi-Fi via libpcap
2. **Filtro**: BPF filtra pacotes de gerenciamento (probe requests, beacons)
3. **Extração**: Obtém MAC address, RSSI, frequência, canal (quando em modo monitor)
4. **Identificação**: OUIja resolve fabricante a partir do OUI (3 primeiros octetos)
5. **Classificação**: `inferDeviceType()` classifica o dispositivo (smartphone, laptop, router, etc.)
6. **Tracking**: Verifica associação com colaborador, gera eventos arrival/departure
7. **Persistência**: Salva dispositivos e eventos no SQLite
8. **API**: Expõe dados via REST para o dashboard
9. **Dashboard**: Atualiza automaticamente a cada 5 segundos via fetch

### 3.4 Tecnologias

| Camada | Tecnologia | Justificativa |
|---|---|---|
| Backend | Go 1.21+ | Performance, concorrência nativa, binário único |
| Captura | gopacket + libpcap | Padrão da indústria para captura de pacotes |
| Storage | SQLite (go-sqlite3) | Zero configuração, arquivo único, SQL completo |
| OUI Lookup | OUIja | Base IEEE oficial (38k+ OUIs), auto-atualização |
| Config | YAML (gopkg.in/yaml.v3) | Legível, padrão em projetos Go |
| Frontend | HTML5 + CSS3 + Vanilla JS | Zero dependências, carregamento rápido |
| QR Code | go-qrcode | Geração de QR codes para registro via mobile |

---

## 4. Design Detalhado

### 4.1 Captura Wi-Fi e Modo Monitor

O scanner suporta dois modos de operação, detectados automaticamente na inicialização:

**Modo Monitor (Linux com antena Wi-Fi)**:
- Filtro BPF: `type mgt` (pacotes 802.11)
- Dados: MAC, RSSI (-30 a -90 dBm), frequência (2400-7115 MHz), canal (1-177)
- Suporta 2.4 GHz (canais 1-14), 5 GHz (canais 32-177) e 6 GHz/Wi-Fi 6E (canais 1-233)

**Modo Compatibilidade (macOS, Linux sem monitor)**:
- Fallback automático para captura ARP/Ethernet
- Dados: MAC apenas (RSSI, frequência e canal reportados como 0)

Cálculo de canal a partir da frequência:
- 2.4 GHz: `canal = (freq - 2412) / 5 + 1`
- 5 GHz: `canal = (freq - 5000) / 5`
- 6 GHz: `canal = (freq - 5950) / 5`

### 4.2 Identificação de Dispositivos (OUIja)

A identificação opera em duas camadas:

**Camada 1 - Resolução de fabricante (OUIja)**:
- Base IEEE oficial via Wireshark (~38k entradas)
- Cache em memória com `sync.RWMutex` (thread-safe)
- TTL de 7 dias para atualização automática
- Cache local em `~/.cache/ouija/manuf`

**Camada 2 - Inferência de tipo (deviceid)**:
- O OUIja retorna apenas o nome do fabricante (ex: "Apple")
- A função `inferDeviceType()` classifica o dispositivo com base no fabricante
- Tipos: smartphone, laptop, tablet, router, iot, smarttv, console, unknown
- Fabricantes ambiguos (Apple, Samsung, etc.) retornam `IsAmbiguous: true` com `PossibleTypes`

**Fluxo de lookup**:
```
MAC → Cache hit? → retorna
        ↓ miss
      OUIja (base IEEE) → found? → inferDeviceType() → cache → retorna
        ↓ not found
      Unknown → cache → retorna
```

**Performance**:
- Primeiro acesso: ~100-500ms (download da base)
- Acessos subsequentes: ~1-5ms (cache em memória)

**API endpoints**:

| Endpoint | Método | Descrição |
|---|---|---|
| `/api/vendor/details` | POST | Detalhes de um MAC (vendor, OUI, tipo) |
| `/api/vendor/search` | POST | Busca fabricantes por padrão |
| `/api/vendor/top` | GET | Top fabricantes por OUIs registrados |
| `/api/vendor/stats` | GET | Estatísticas da base OUI |

### 4.3 Storage e Persistência (SQLite)

**Tabelas principais**:

| Tabela | Propósito |
|---|---|
| `devices` | Dispositivos detectados (MAC, vendor, tipo, sinal, timestamps) |
| `employees` | Colaboradores cadastrados (MAC, nome, departamento, campos custom) |
| `events` | Eventos de presença (arrival/departure com timestamps) |
| `registration_tokens` | Tokens para registro via QR Code |

**Campos customizados** (tabela `employees`):
- `custom_device_type`: Tipo de dispositivo definido pelo usuário (sobrescreve detecção automática)
- `custom_vendor`: Fabricante definido pelo usuário (sobrescreve OUIja)

**Queries de histórico**:
- `GetHistory7Days()`: Agrega eventos dos últimos 7 dias por colaborador
- `GetFirstArrivalToday()`: Primeiro evento de arrival do dia
- `GetOnlineDurationToday()`: Soma de intervalos arrival→departure

**Limpeza automática**:
- Scheduler diário remove dispositivos nao-cadastrados inativos ha 7+ dias
- Eventos com 30+ dias sao removidos automaticamente
- Limpeza manual disponivel via `POST /api/cleanup/inactive`

### 4.4 Interface Web

**Arquitetura**: Single Page Application (SPA) com vanilla JS, sem frameworks.

**Sistema de abas**:

| Aba | Conteúdo |
|---|---|
| Tempo Real | Dispositivos conectados, status, sinal, ações de cadastro |
| Histórico (7 dias) | Cards diários com presença por colaborador, percentuais |
| Colaboradores | CRUD completo com modal de cadastro/edição |

**Indicadores visuais**:
- Barra de sinal: 7 níveis baseados em RSSI (-30 a -100 dBm)
- Status: ativo (verde) / inativo (vermelho)
- Badges: percentual de presença colorido por faixa
- Auto-refresh: 5 segundos (apenas na aba Tempo Real)

**Paleta de cores**:
```css
--primary: #667eea;    /* Roxo/Azul */
--secondary: #764ba2;  /* Roxo escuro */
--success: #28a745;    /* Verde - Ativo */
--danger: #dc3545;     /* Vermelho - Inativo */
```

**Performance**:
- Bundle HTML: ~15KB
- Zero dependencias JS externas
- Tempo de carregamento: < 500ms

### 4.5 API REST

**Endpoints principais**:

| Endpoint | Método | Descrição |
|---|---|---|
| `/api/devices` | GET | Lista dispositivos detectados |
| `/api/employees` | GET | Lista colaboradores cadastrados |
| `/api/employees/{mac}` | DELETE | Remove colaborador (preserva histórico) |
| `/api/associate` | POST | Cadastra/edita colaborador |
| `/api/stats` | GET | Estatísticas do sistema |
| `/api/events` | GET | Últimos 100 eventos |
| `/api/report/today` | GET | Relatório de presença do dia |
| `/api/history/7days` | GET | Histórico de 7 dias |
| `/api/cleanup/inactive` | POST | Limpeza manual de inativos |
| `/api/register/token` | POST | Gera QR Code para registro |
| `/api/register/submit` | POST | Submete registro via QR Code |
| `/api/vendor/details` | POST | Detalhes de fabricante (OUIja) |
| `/api/vendor/search` | POST | Busca fabricantes por padrão |
| `/api/vendor/top` | GET | Top fabricantes por OUIs |
| `/api/vendor/stats` | GET | Estatísticas da base OUI |

**Exemplos de uso**:

```bash
# Listar dispositivos ativos
curl -s http://localhost:8080/api/devices | jq '.[] | select(.is_active == true)'

# Cadastrar colaborador
curl -X POST http://localhost:8080/api/associate \
  -H "Content-Type: application/json" \
  -d '{"mac_address": "AA:BB:CC:DD:EE:FF", "name": "Maria Santos", "department": "RH"}'

# Consultar fabricante de um MAC
curl -X POST http://localhost:8080/api/vendor/details \
  -H "Content-Type: application/json" \
  -d '{"mac_address": "A4:5E:60:12:34:56"}'

# Estatísticas da base OUI
curl http://localhost:8080/api/vendor/stats
```

**CORS**: Habilitado para `GET`, `POST`, `DELETE`, `OPTIONS`.

---

## 5. Deploy e Operação

Consulte o [DEPLOY.md](DEPLOY.md) para instruções detalhadas de deploy em diferentes ambientes.

Consulte o [QUICKSTART.md](QUICKSTART.md) para setup rapido em menos de 5 minutos.

### 5.1 Requisitos de Sistema

- Go 1.21+
- libpcap-dev (Linux) ou libpcap nativa (macOS)
- Permissões de root/sudo (captura de pacotes)
- Conexão com internet (apenas no primeiro uso, para download da base OUI)

### 5.2 Configuração

Arquivo `config.yaml`:

```yaml
network:
  interface: "wlan0"     # Auto-detectado se disponível
  scan_interval: 30      # Segundos entre scans

presence:
  timeout_minutes: 20    # Tempo para considerar offline (mínimo: 20)

logging:
  log_dir: "./logs"      # Diretório de logs JSON (rotação diária)
```

### 5.3 Monitoramento

- **Logs**: Arquivos JSON com rotação diária em `./logs/`
- **Limpeza automática**: Dispositivos inativos (7 dias) e eventos antigos (30 dias)
- **Health check**: `GET /api/stats` retorna estado do sistema

---

## 6. Decisões de Design

### 6.1 Preservação de Histórico

**Problema**: Ao excluir um colaborador, o histórico de presença não pode ser perdido (compliance trabalhista, auditorias).

**Decisão**: Remover `ON DELETE CASCADE` da FK na tabela events. Ao deletar um colaborador, apenas o registro em `employees` é removido. Eventos, dispositivos e dados históricos são preservados permanentemente.

**Impacto**:
- `DeleteEmployee()` remove apenas da tabela employees
- Eventos mantêm o nome do colaborador como campo denormalizado
- Relatórios históricos continuam funcionando após exclusão

### 6.2 Migração de MAC Address

**Problema**: Quando um colaborador troca de dispositivo (novo MAC), o histórico precisa ser unificado.

**Decisão**: `UpdateHistoryMAC()` atualiza o MAC em todas as tabelas (events, devices) de forma atômica. O histórico antigo passa a ser acessível pelo novo MAC.

**Fluxo**:
```
Edição de MAC (AA:BB → CC:DD)
  → UPDATE events SET mac_address = 'CC:DD' WHERE mac_address = 'AA:BB'
  → UPDATE devices SET mac_address = 'CC:DD' WHERE mac_address = 'AA:BB'
  → DELETE FROM employees WHERE mac_address = 'AA:BB'
  → INSERT/UPDATE employees com novo MAC
```

### 6.3 Campos Customizados

**Problema**: A detecção automática de tipo/fabricante via OUI é imprecisa para fabricantes ambíguos (Apple pode ser iPhone, MacBook ou iPad).

**Decisão**: Campos `custom_device_type` e `custom_vendor` na tabela employees permitem que o gestor sobrescreva a detecção automática. Quando preenchidos, a API retorna o valor customizado em vez do detectado.

**Regra**: Se `custom_device_type != ""`, usa o valor customizado. Caso contrário, usa o valor detectado pelo OUIja + inferDeviceType().

### 6.4 OUIja como Fonte Unica

**Problema**: A base OUI hardcoded (~50 entradas) era insuficiente para identificação precisa.

**Decisão**: Migrar para a biblioteca OUIja como fonte única de verdade (38k+ OUIs da base IEEE oficial). A base hardcoded foi completamente removida na v2.0.

**Trade-offs**:
- Primeira execução requer download (~2MB)
- Necessita internet a cada 7 dias para atualização (funciona offline com cache)
- Código reduzido de 738 para 507 linhas

---

## Referências

- [QUICKSTART.md](QUICKSTART.md) - Guia rápido de início
- [DEPLOY.md](DEPLOY.md) - Guia de deploy
- [CONTRIBUTING.md](CONTRIBUTING.md) - Guia de contribuição
- [CHANGELOG.md](CHANGELOG.md) - Histórico de mudanças
- [ROADMAP.md](ROADMAP.md) - Roadmap de desenvolvimento
- [OUIja](https://github.com/lucasrafaldini/ouija) - Biblioteca de lookup MAC Address
