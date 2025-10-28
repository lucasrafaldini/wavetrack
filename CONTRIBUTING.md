# Guia de Desenvolvimento - WaveTrack

Bem-vindo ao guia de desenvolvimento do WaveTrack! Este documento ajudará você a configurar o ambiente e contribuir com o projeto.

## 🛠️ Setup do Ambiente

### Pré-requisitos

- Go 1.21 ou superior
- Git
- libpcap (para captura de pacotes Wi-Fi)
- Editor de código (recomendado: VS Code)

### Instalação Local

```bash
# Clone o repositório
git clone https://github.com/lucasrafaldini/wavetrack.git
cd wavetrack

# Instale dependências
go mod download

# Execute em modo desenvolvimento
sudo go run cmd/wavetrack/main.go
```

### VS Code Setup

Extensões recomendadas:
- Go (golang.go)
- Rest Client (humao.rest-client)
- GitLens (eamodio.gitlens)

Configuração `.vscode/settings.json`:
```json
{
    "go.lintTool": "golangci-lint",
    "go.formatTool": "goimports",
    "editor.formatOnSave": true
}
```

## 📐 Arquitetura

```
┌─────────────────────────────────────────────────────────┐
│                     WaveTrack System                     │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────┐     ┌──────────────┐                 │
│  │   Web UI     │────▶│  API Server  │                 │
│  │ (Dashboard)  │     │   (REST)     │                 │
│  └──────────────┘     └──────┬───────┘                 │
│                              │                          │
│                              ▼                          │
│  ┌──────────────┐     ┌──────────────┐                 │
│  │   Storage    │◀────│   Tracker    │                 │
│  │  (JSON DB)   │     │  (Presence)  │                 │
│  └──────────────┘     └──────┬───────┘                 │
│                              │                          │
│                              ▼                          │
│  ┌──────────────┐     ┌──────────────┐                 │
│  │    Logger    │◀────│   Scanner    │                 │
│  │   (Events)   │     │   (Wi-Fi)    │                 │
│  └──────────────┘     └──────────────┘                 │
│                              │                          │
│                              ▼                          │
│                       [Network Interface]               │
│                              │                          │
└──────────────────────────────┼──────────────────────────┘
                               │
                               ▼
                        [Wi-Fi Packets]
```

## 📦 Estrutura de Pacotes

### `internal/wifi`
Responsável pela captura de pacotes Wi-Fi usando libpcap.

**Principais componentes:**
- `Scanner`: Gerencia captura de pacotes
- Filtros BPF para pacotes de gerenciamento
- Processamento de camadas 802.11

### `internal/models`
Define as estruturas de dados do sistema.

**Principais modelos:**
- `Device`: Representa um dispositivo detectado
- `Employee`: Representa um funcionário
- `PresenceEvent`: Evento de presença

### `internal/storage`
Gerencia persistência de dados em JSON.

**Funcionalidades:**
- Save/Load de funcionários
- Thread-safe com RWMutex
- Busca por MAC address

### `internal/tracker`
Lógica de rastreamento de presença.

**Responsabilidades:**
- Associar dispositivos a funcionários
- Detectar chegadas/saídas
- Gerenciar timeouts

### `internal/logger`
Sistema de logging de eventos.

**Features:**
- Rotação diária de logs
- Formato JSON
- Registro em console e arquivo

### `internal/api`
Servidor HTTP e endpoints REST.

**Endpoints:**
- `/api/devices` - Lista dispositivos
- `/api/employees` - Gerencia funcionários
- `/api/associate` - Associa MAC a funcionário
- `/api/stats` - Estatísticas do sistema

### `internal/config`
Gerenciamento de configuração YAML.

## 🔧 Desenvolvimento

### Adicionar Nova Feature

1. **Criar branch**
```bash
git checkout -b feature/nome-da-feature
```

2. **Implementar**
- Escrever código
- Adicionar testes
- Documentar

3. **Testar**
```bash
go test ./...
go vet ./...
```

4. **Commit**
```bash
git add .
git commit -m "feat: descrição da feature"
```

5. **Push e PR**
```bash
git push origin feature/nome-da-feature
# Abrir Pull Request no GitHub
```

### Padrões de Código

#### Nomenclatura
```go
// Structs: PascalCase
type PresenceTracker struct { }

// Métodos públicos: PascalCase
func (t *Tracker) Start() { }

// Métodos privados: camelCase
func (t *Tracker) handleDevice() { }

// Constantes: PascalCase ou UPPER_SNAKE
const DefaultTimeout = 5 * time.Minute
```

#### Comentários
```go
// Package scanner implementa captura de pacotes Wi-Fi
package scanner

// Scanner gerencia a captura de pacotes na interface de rede
type Scanner struct {
    // ...
}

// Start inicia o processo de captura
func (s *Scanner) Start() error {
    // ...
}
```

#### Error Handling
```go
// ✅ Bom
if err != nil {
    return fmt.Errorf("erro ao processar dispositivo: %w", err)
}

// ❌ Evitar
if err != nil {
    log.Println(err)
}
```

### Testes

#### Estrutura de Teste
```go
func TestScanner_Start(t *testing.T) {
    // Arrange
    scanner := NewScanner("test0")
    
    // Act
    err := scanner.Start()
    
    // Assert
    if err == nil {
        t.Error("Expected error for invalid interface")
    }
}
```

#### Executar Testes
```bash
# Todos os testes
go test ./...

# Com coverage
go test -cover ./...

# Verbose
go test -v ./...

# Específico
go test ./internal/storage
```

### Debugging

#### Com Delve
```bash
# Instalar
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug
sudo dlv debug cmd/wavetrack/main.go
```

#### Logs de Debug
```go
// Adicionar temporariamente
log.Printf("[DEBUG] Device: %+v", device)
```

## 🎨 Frontend

### Estrutura do HTML
O dashboard é um SPA (Single Page Application) em vanilla JavaScript.

**Componentes principais:**
- Stats cards (estatísticas)
- Devices table (tabela de dispositivos)
- Associate modal (modal de cadastro)

### Adicionar Nova Feature no Dashboard

1. **Adicionar HTML**
```html
<div id="myNewFeature">
    <!-- conteúdo -->
</div>
```

2. **Adicionar JavaScript**
```javascript
async function loadMyFeature() {
    const response = await fetch('/api/my-endpoint');
    const data = await response.json();
    // Renderizar...
}
```

3. **Adicionar CSS**
```css
.my-feature {
    /* estilos */
}
```

## 🚀 Build e Deploy

### Build para Produção
```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o wavetrack-linux cmd/wavetrack/main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o wavetrack-macos cmd/wavetrack/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o wavetrack.exe cmd/wavetrack/main.go
```

### Otimizar Binary
```bash
go build -ldflags="-s -w" -o wavetrack cmd/wavetrack/main.go
```

## 📝 Convenções de Commit

Seguimos [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` Nova feature
- `fix:` Correção de bug
- `docs:` Documentação
- `style:` Formatação
- `refactor:` Refatoração
- `test:` Testes
- `chore:` Manutenção

Exemplos:
```
feat: adicionar suporte a múltiplas interfaces
fix: corrigir detecção de timeout
docs: atualizar README com instruções de API
```

## 🐛 Debugging de Problemas Comuns

### Scanner não detecta dispositivos
```go
// Verificar filtro BPF
log.Printf("Filter: %s", filter)

// Verificar interface
log.Printf("Interface: %s", iface)

// Verificar permissões
// Deve rodar com sudo/root
```

### API não responde
```go
// Verificar porta
log.Printf("Server listening on :%d", port)

// Verificar CORS
// Headers devem estar configurados
```

## 📚 Recursos Úteis

- [Go Documentation](https://go.dev/doc/)
- [gopacket Guide](https://github.com/google/gopacket)
- [802.11 Standard](https://standards.ieee.org/ieee/802.11/)
- [pcap Programming](https://www.tcpdump.org/pcap.html)

## 🤝 Contribuindo

1. Fork o projeto
2. Crie sua branch de feature
3. Commit suas mudanças
4. Push para a branch
5. Abra um Pull Request

### Checklist do PR

- [ ] Código funciona e foi testado
- [ ] Testes unitários adicionados/atualizados
- [ ] Documentação atualizada
- [ ] Código segue os padrões do projeto
- [ ] Commits seguem Conventional Commits
- [ ] Branch está atualizada com main

## 📞 Suporte

- Issues: [GitHub Issues](https://github.com/lucasrafaldini/wavetrack/issues)
- Discussões: [GitHub Discussions](https://github.com/lucasrafaldini/wavetrack/discussions)

## 📄 Licença

MIT License - veja LICENSE para detalhes.
