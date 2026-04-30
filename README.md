# WaveTrack

Sistema de monitoramento de presença baseado em detecção de dispositivos Wi-Fi. O WaveTrack captura handshakes de dispositivos em uma rede Wi-Fi para registrar automaticamente a presença de funcionários.

> **[Quick Start Guide](.github/docs/QUICKSTART.md)** | [Documentação Completa](.github/docs/SUMMARY.md) | [API Examples](.github/docs/API_EXAMPLES.md) | [Deploy Guide](.github/docs/DEPLOY.md) | [Contributing](.github/docs/CONTRIBUTING.md)

## Funcionalidades

### Sistema de Monitoramento
- **Monitoramento Wi-Fi**: Captura pacotes de gerenciamento Wi-Fi para detectar dispositivos próximos
- **Registro de Presença**: Associa dispositivos (MACs) a funcionários cadastrados
- **Eventos Automáticos**: Detecta automaticamente chegadas e saídas
- **Sistema de Logs**: Registra todos os eventos em arquivos JSON com rotação diária
- **Banco de Dados SQLite**: Histórico permanente de presença para análises e relatórios

### Interface Web Completa
- **Dashboard em Tempo Real**: Visualização de dispositivos ativos e suas informações
- **Sistema de Abas**:
 - **Tempo Real**: Monitore dispositivos conectados no momento
 - **Histórico (7 dias)**: Analise presença dos últimos 7 dias com percentuais
 - **Colaboradores**: Gerencie cadastros de funcionários (criar, editar, deletar)
- **Cadastro Dinâmico**: Adicione e edite funcionários diretamente pela interface
- **Ordenação e Filtros**: Organize dados por nome, tempo online, chegada, etc.

### Gestão de Colaboradores
- **CRUD Completo**: Criar, visualizar, editar e deletar colaboradores
- **Edição com Preservação de Histórico**:
 - Altere MAC address e todo o histórico é migrado automaticamente
 - Atualize nome e os eventos passados refletem a mudança
 - Edite departamento sem afetar o histórico
- **Exclusão Inteligente**: Remova colaboradores mantendo 100% do histórico para relatórios

### Histórico e Relatórios
- **Histórico de 7 Dias**: Visualize presença diária com:
 - Horário de chegada e saída
 - Total de horas trabalhadas
 - Percentual de presença (base: 8 horas = 100%)
 - Média de presença semanal
- **Dados Preservados**: Histórico nunca é deletado, mesmo removendo colaboradores
- **API REST**: Endpoints completos para integração com outros sistemas

## Requisitos

### Sistema Operacional
- **macOS** ou **Linux** com interface Wi-Fi
- Permissões de administrador (necessário para captura de pacotes)

### Dependências
- Go 1.21 ou superior
- libpcap (para captura de pacotes)

#### Instalação do libpcap

**macOS:**
```bash
brew install libpcap
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get install libpcap-dev
```

**Linux (Fedora/RHEL):**
```bash
sudo dnf install libpcap-devel
```

## Instalação

1. **Clone o repositório:**
```bash
git clone https://github.com/lucasrafaldini/wavetrack.git
cd wavetrack
```

2. **Instale as dependências:**
```bash
go mod download
```

3. **Configure o projeto:**
 - Edite `config.yaml` com suas configurações
 - Ajuste a interface de rede (geralmente `en0` no macOS, `wlan0` no Linux)
 - Funcionários serão cadastrados via interface web após iniciar o sistema

4. **Compile o projeto:**
```bash
go build -o wavetrack cmd/wavetrack/main.go
```

5. **Ajuste permissões do banco de dados (macOS):**

Se o banco de dados for criado com `sudo`, você precisará ajustar as permissões para permitir acesso sem root:

```bash
# Após a primeira execução, ajuste o proprietário do diretório de dados
sudo chown -R $(whoami):staff ./data

# Ou rode o app sem sudo, mas com permissões de captura:
# (requer configuração adicional do sistema)
```

## Configuração

Edite o arquivo `config.yaml`:

```yaml
network:
 interface: "en0" # Interface de rede Wi-Fi
 channel: 6 # Canal Wi-Fi
 scan_interval: 10 # Intervalo de scan em segundos

logging:
 log_dir: "./logs"
 log_level: "info"

presence:
 timeout_minutes: 20 # Tempo de inatividade antes de marcar offline (mín. 20 min)
 signal_threshold: -75 # Sinal mínimo em dBm (0 = ignora; usado no modo não-monitor)
```

**Nota**: Funcionários agora são cadastrados via interface web e armazenados no banco de dados SQLite (`data/wavetrack.db`). Não há mais seção de employees no config.yaml.

### Como descobrir o endereço MAC de um dispositivo:

**iPhone/iPad:**
- Configurações → Geral → Sobre → Endereço Wi-Fi

**Android:**
- Configurações → Sobre o telefone → Status → Endereço MAC Wi-Fi

**Laptop:**
```bash
# macOS
ifconfig en0 | grep ether

# Linux
ip link show wlan0
```

## API REST

O WaveTrack expõe uma API REST para integração com outros sistemas:

### Endpoints Disponíveis:

#### `GET /api/devices`
Lista todos os dispositivos detectados

**Resposta:**
```json
[
 {
 "mac_address": "00:11:22:33:44:55",
 "type": "smartphone",
 "vendor": "Apple",
 "signal_strength": -45,
 "first_seen": "2025-10-28T14:30:00Z",
 "last_seen": "2025-10-28T15:45:00Z",
 "is_active": true,
 "employee_name": "João Silva",
 "department": "TI",
 "first_seen_today": "2025-10-28T08:15:00Z",
 "online_duration_seconds": 14400
 }
]
```

#### `GET /api/employees`
Lista todos os funcionários cadastrados

**Resposta:**
```json
[
 {
 "id": 1,
 "mac_address": "00:11:22:33:44:55",
 "name": "João Silva",
 "department": "TI",
 "custom_device_type": "iPhone",
 "custom_vendor": "Apple",
 "created_at": "2025-10-28T10:00:00Z",
 "updated_at": "2025-10-28T10:00:00Z"
 }
]
```

#### `POST /api/associate`
Cria ou atualiza um funcionário

**Requisição (Novo):**
```json
{
 "mac_address": "00:11:22:33:44:55",
 "name": "João Silva",
 "department": "TI",
 "custom_device_type": "iPhone 15 Pro",
 "custom_vendor": "Apple"
}
```

**Requisição (Edição com alteração de MAC):**
```json
{
 "mac_address": "00:11:22:33:44:66",
 "old_mac_address": "00:11:22:33:44:55",
 "name": "João Silva",
 "department": "TI",
 "custom_device_type": "iPhone 15 Pro",
 "custom_vendor": "Apple"
}
```

**Resposta:**
```json
{
 "message": "Funcionário cadastrado com sucesso"
}
```

**Nota:** Ao alterar o MAC address, todo o histórico de eventos é automaticamente migrado para o novo MAC.

#### `DELETE /api/employees/{mac}`
Remove um funcionário (preserva histórico)

**Exemplo:**
```bash
curl -X DELETE http://localhost:8080/api/employees/00:11:22:33:44:55
```

**Resposta:**
```json
{
 "message": "Funcionário deletado com sucesso"
}
```

**Nota:** O histórico de presença é preservado para relatórios e análises.

#### `GET /api/history/7days`
Retorna histórico de presença dos últimos 7 dias

**Resposta:**
```json
[
 {
 "date": "2025-10-28",
 "day_of_week": "Segunda",
 "mac_address": "00:11:22:33:44:55",
 "name": "João Silva",
 "department": "TI",
 "arrival": "08:15:23",
 "departure": "18:30:45",
 "total_hours": "10.25",
 "percentage": "128%"
 }
]
```

#### `GET /api/stats`
Retorna estatísticas do sistema

**Resposta:**
```json
{
 "total_devices": 15,
 "active_devices": 8,
 "total_employees": 5,
 "events_today": 24,
 "offline_threshold_minutes": 20
}
```

## Uso

**Execute com permissões de administrador:**

```bash
# macOS
sudo ./wavetrack

# Linux
sudo ./wavetrack

# Com porta customizada para o servidor web (padrão: 8080)
sudo ./wavetrack -port 3000

# Com arquivo de configuração customizado
sudo ./wavetrack -config /caminho/para/config.yaml
```

### Acessando a Interface Web

Após iniciar o sistema, abra seu navegador e acesse:

```
http://localhost:8080
```

### Funcionalidades da Interface:

#### 1. **Tempo Real (Tab Principal)**
 - Visualização de todos os dispositivos detectados no momento
 - Status de atividade (Ativo/Inativo) com threshold de 20 minutos
 - Força do sinal em tempo real (N/A em modo não-monitor)
 - Primeira chegada do dia ("Chegou às")
 - Tempo total online no dia atual
 - Estatísticas do sistema em tempo real

#### 2. **Histórico de 7 Dias**
 - Visualização completa dos últimos 7 dias de presença
 - Para cada colaborador e dia:
 - Data e dia da semana
 - Horário de chegada (primeiro arrival)
 - Horário de saída (último departure)
 - Total de horas trabalhadas
 - Percentual de presença (8 horas = 100%)
 - Média semanal de presença por colaborador
 - Dados carregados do banco de dados SQLite

#### 3. **Gerenciamento de Colaboradores**
 - **Listar**: Visualize todos os colaboradores cadastrados
 - **Criar**: Adicione novos colaboradores com nome, MAC, departamento
 - **Editar**: Modifique dados de colaboradores existentes
 - Alterar MAC: Histórico é migrado automaticamente
 - Alterar Nome: Todos os eventos são atualizados
 - Alterar Departamento: Registro atualizado
 - **Deletar**: Remova colaboradores (com confirmação)
 - O colaborador é removido da lista ativa
 - Todo o histórico é preservado para relatórios

#### 4. **Ordenação e Filtros**
 - Ordenar por: Nome do funcionário, Chegou às, Tempo online
 - Filtro: Somente funcionários (oculta dispositivos não cadastrados)
 - Direção: Ascendente ou Descendente

### Saída Esperada (Console):
```
=== WaveTrack - Sistema de Monitoramento de Presença ===
Configuração carregada: interface=en0, intervalo=10s
Sistema de armazenamento iniciado
Sistema de logs iniciado: ./logs/presence_2025-10-28.log
Scanner iniciado na interface en0
Monitorando 2 funcionários cadastrados
 Servidor web iniciado em http://localhost:8080
 Acesse o dashboard no navegador!
Sistema iniciado! Pressione Ctrl+C para parar...
 João Silva chegou (Departamento: TI)
 Maria Santos chegou (Departamento: RH)
```

## Estrutura do Projeto

```
wavetrack/
├── cmd/
│ ├── wavetrack/
│ │ └── main.go # Aplicação principal
│ └── seed_history/
│ └── main.go # Gerador de dados históricos (dev)
├── internal/
│ ├── api/
│ │ └── handlers.go # Endpoints da API REST
│ ├── config/
│ │ └── config.go # Gerenciamento de configuração
│ ├── deviceid/
│ │ └── deviceid.go # Identificação de vendor/tipo via OUI
│ ├── logger/
│ │ └── logger.go # Sistema de logging
│ ├── models/
│ │ └── models.go # Modelos de dados
│ ├── storage/
│ │ ├── storage.go # Persistência SQLite
│ │ ├── schema.sql # Schema do banco
│ │ └── storage_test.go # Testes de storage
│ ├── tracker/
│ │ └── tracker.go # Lógica de rastreamento
│ └── wifi/
│ └── scanner.go # Captura de pacotes Wi-Fi
├── web/
│ ├── index.html # Interface web (estrutura)
│ ├── css/
│ │ └── style.css # Estilos da interface
│ └── js/
│ └── app.js # Lógica JavaScript
├── data/ # Banco SQLite (criado automaticamente)
│ └── wavetrack.db
├── logs/ # Arquivos de log (criado automaticamente)
├── config.yaml # Arquivo de configuração
├── migrate_preserve_history.sql # Migração do banco (aplicada)
├── go.mod # Dependências Go
└── README.md # Este arquivo
```

## Formato dos Logs

Os eventos são salvos em formato JSON rotacionado diariamente (`logs/presence_YYYY-MM-DD.log`):

```json
{
 "timestamp": "2025-10-28T14:30:00Z",
 "event_type": "arrival",
 "employee_name": "João Silva",
 "mac_address": "00:11:22:33:44:55",
 "signal_strength": -45,
 "metadata": ""
}
```

**Tipos de eventos:**
- `arrival`: Funcionário chegou
- `departure`: Funcionário saiu
- `unknown_device`: Dispositivo desconhecido detectado

## Considerações Importantes

1. **Permissões**: O programa precisa rodar como root/sudo para capturar pacotes
2. **Permissões de Banco (macOS)**: Se o banco SQLite for criado com sudo, ajuste o proprietário com `sudo chown -R $(whoami):staff ./data` após a primeira execução
3. **Modo Monitor vs. Compatibilidade**: No macOS, o scanner usa modo ARP/Ethernet compatível quando 802.11 não está disponível. Força de sinal (RSSI) não é capturada nesse modo
4. **Privacidade**: Este sistema captura endereços MAC. Certifique-se de estar em conformidade com as leis locais de privacidade (LGPD, GDPR, etc.)
5. **Alcance**: A detecção depende do alcance do sinal Wi-Fi (tipicamente 10-50 metros)
6. **Threshold Offline**: Dispositivos são marcados como offline após 20 minutos de inatividade (configurável em `presence.timeout_minutes`, com mínimo de 20 minutos)
7. **Histórico Preservado**: Ao deletar um colaborador, todo o histórico de presença é mantido no banco para fins de relatórios e auditoria
8. **Migração de Dados**: Ao editar o MAC de um colaborador, todo o histórico é automaticamente migrado para o novo MAC

## Documentação Adicional

- **[HISTORY_PRESERVATION.md](.github/docs/HISTORY_PRESERVATION.md)** - Como o sistema preserva e gerencia histórico
- **[SEED_HISTORY.md](.github/docs/SEED_HISTORY.md)** - Como gerar dados de teste para desenvolvimento
- **[TABS_HISTORICO.md](.github/docs/TABS_HISTORICO.md)** - Documentação do sistema de abas e histórico
- **[QUICKSTART.md](.github/docs/QUICKSTART.md)** - Guia rápido de início
- **[API_EXAMPLES.md](.github/docs/API_EXAMPLES.md)** - Exemplos de uso da API
- **[DEPLOY.md](.github/docs/DEPLOY.md)** - Guia de deployment
- **[CONTRIBUTING.md](.github/docs/CONTRIBUTING.md)** - Como contribuir com o projeto

## Roadmap

### Implementado (v1.0)
- [x] Interface web para visualização em tempo real
- [x] Dashboard para gerentes com sistema de abas
- [x] Cadastro dinâmico de funcionários via web (CRUD completo)
- [x] API REST para integração
- [x] Banco de dados SQLite para histórico de presença
- [x] Cálculo de tempo online e primeira chegada do dia
- [x] Ordenação e filtro de dispositivos no dashboard
- [x] Histórico de 7 dias com percentuais de presença
- [x] Edição de colaboradores com migração automática de histórico
- [x] Preservação de histórico ao deletar colaboradores
- [x] Sistema de abas (Tempo Real, Histórico, Colaboradores)
- [x] Organização de código (HTML, CSS, JS separados)

### Implementado (v2.0)
- [x] **Integração com OUIja** - Biblioteca própria para consulta MAC Address
  - Base IEEE oficial via Wireshark (38k+ OUIs)
  - Cache inteligente em memória com sync.RWMutex
  - Atualização automática da base (TTL de 7 dias)
  - API endpoints: vendor details, search, top vendors, stats
- [x] **Remoção da base OUI hardcoded** - Substituída pelo OUIja
- [x] **CI/CD com GitHub Actions** - Lint, testes com coverage, benchmark, build matrix
- [x] **Endpoints OUIja na API REST** - `/api/vendor/details`, `/search`, `/top`, `/stats`

### Próximas Funcionalidades

#### v2.1 - Interface e UX
- [ ] Interface web responsiva (mobile-first)
- [ ] Dashboard em tempo real aprimorado com gráficos interativos
- [ ] Sistema de notificações push

#### v2.2 - Recursos Empresariais
- [ ] Relatório diário de presença (PDF/CSV)
- [ ] Notificações por webhook (email/Slack)
- [ ] Suporte a múltiplos dispositivos por funcionário
- [ ] Autenticação e controle de acesso
- [ ] Integração com sistemas de RH

> **Roadmap Completo**: Veja o [ROADMAP.md](.github/docs/ROADMAP.md) para detalhes técnicos e cronograma completo

## Licença

MIT License

## Autor

Lucas Rafaldini (@lucasrafaldini)

---

**Versão:** 2.0
**Status:** Production Ready
**Última Atualização:** Abril de 2026

**Nota**: Sistema completo com interface web, histórico de 7 dias, CRUD de colaboradores, preservação inteligente de dados históricos e identificação de dispositivos via OUIja (base IEEE oficial).