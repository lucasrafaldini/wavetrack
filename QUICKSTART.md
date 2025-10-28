# 🚀 Quick Start - WaveTrack

Comece a usar o WaveTrack em menos de 5 minutos!

## ⚡ Início Rápido

### 1️⃣ Pré-requisitos

Certifique-se de ter instalado:

```bash
# macOS
brew install libpcap go

# Linux (Ubuntu/Debian)
sudo apt-get install libpcap-dev golang-go

# Linux (Fedora/RHEL)
sudo dnf install libpcap-devel golang
```

### 2️⃣ Baixar Dependências

```bash
cd /Users/outis/Desktop/wavetrack
go mod download
```

### 3️⃣ Configurar Interface de Rede

Edite `config.yaml` e ajuste a interface de rede:

```bash
# Descobrir sua interface Wi-Fi
# macOS:
ifconfig | grep -A 5 "^en"

# Linux:
ip link show | grep wlan
```

Atualize em `config.yaml`:
```yaml
network:
  interface: "en0"  # ← Substitua pela sua interface
```

### 4️⃣ Compilar

```bash
go build -o wavetrack cmd/wavetrack/main.go
```

### 5️⃣ Executar

```bash
sudo ./wavetrack
```

### 6️⃣ Acessar Dashboard

Abra seu navegador em:
```
http://localhost:8080
```

## 🎯 Primeiros Passos na Interface

### Passo 1: Visualize os Dispositivos
Você verá uma lista de dispositivos detectados na sua rede Wi-Fi.

### Passo 2: Cadastre um Funcionário
1. Encontre um dispositivo com "Não cadastrado"
2. Clique no botão **[Cadastrar]**
3. Digite o nome do funcionário
4. Adicione o departamento (opcional)
5. Clique em **Salvar**

### Passo 3: Monitore em Tempo Real
- O dashboard atualiza automaticamente a cada 5 segundos
- Veja quem está presente/ausente
- Acompanhe a força do sinal

## 📱 Como Descobrir o MAC de um Dispositivo

### iPhone/iPad
1. Abra **Configurações**
2. Vá em **Geral** → **Sobre**
3. Procure por **Endereço Wi-Fi**

### Android
1. Abra **Configurações**
2. Vá em **Sobre o telefone** → **Status**
3. Procure por **Endereço MAC Wi-Fi**

### Computador (macOS)
```bash
ifconfig en0 | grep ether
```

### Computador (Linux)
```bash
ip link show wlan0 | grep link/ether
```

### Computador (Windows)
```cmd
ipconfig /all
```
Procure por "Endereço Físico" da interface Wi-Fi.

## 🔧 Teste Rápido da API

### Ver Estatísticas
```bash
curl http://localhost:8080/api/stats
```

### Listar Dispositivos
```bash
curl http://localhost:8080/api/devices
```

### Cadastrar Funcionário via API
```bash
curl -X POST http://localhost:8080/api/associate \
  -H "Content-Type: application/json" \
  -d '{
    "mac_address": "AA:BB:CC:DD:EE:FF",
    "employee_name": "João Silva",
    "department": "TI"
  }'
```

## 🎨 Interface - Tour Rápido

### Cards de Estatísticas (Topo)
```
┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐
│     15     │ │      8     │ │      5     │ │      3     │
│Dispositivos│ │   Ativos   │ │Funcionários│ │ Presentes  │
└────────────┘ └────────────┘ └────────────┘ └────────────┘
```

### Tabela de Dispositivos
- **🟢 Verde**: Dispositivo ativo (visto recentemente)
- **🔴 Vermelho**: Dispositivo inativo (não visto há tempo)
- **Barra de Sinal**: Mostra força do Wi-Fi visualmente
- **[Cadastrar]**: Botão para associar funcionário

### Botão de Atualização (Canto inferior direito)
- Clique para forçar atualização imediata
- Gira ao atualizar

## 🐛 Troubleshooting Rápido

### Erro: "Permission denied"
**Solução**: Execute com `sudo`
```bash
sudo ./wavetrack
```

### Erro: "No such device"
**Solução**: Interface incorreta no config.yaml
```bash
# Liste interfaces disponíveis
ifconfig  # macOS
ip link   # Linux
```

### Erro: "Address already in use"
**Solução**: Porta 8080 ocupada
```bash
# Use porta diferente
sudo ./wavetrack -port 8081
```

### Nenhum dispositivo aparece
**Possíveis causas**:
1. Interface de rede incorreta
2. Placa Wi-Fi não suporta modo monitor
3. Nenhum dispositivo próximo
4. Filtros de rede bloqueando

**Solução**:
```bash
# Verifique logs
tail -f logs/presence_*.log

# Teste interface manualmente
sudo tcpdump -i en0 -c 10
```

### Dashboard não carrega
**Verifique**:
1. Servidor está rodando?
   ```bash
   curl http://localhost:8080/api/stats
   ```
2. Porta correta? (padrão: 8080)
3. Firewall bloqueando?

## 📊 Exemplo de Saída (Console)

```
=== WaveTrack - Sistema de Monitoramento de Presença ===
Configuração carregada: interface=en0, intervalo=10s
Sistema de armazenamento iniciado
Sistema de logs iniciado: ./logs/presence_2025-10-28.log
Scanner iniciado na interface en0
Monitorando 0 funcionários cadastrados
🌐 Servidor web iniciado em http://localhost:8080
   Acesse o dashboard no navegador!
Sistema iniciado! Pressione Ctrl+C para parar...

✓ João Silva chegou (Departamento: TI)
✓ Maria Santos chegou (Departamento: RH)
✗ João Silva saiu
```

## 📁 Arquivos Gerados

Após executar, você terá:

```
wavetrack/
├── data/
│   └── employees.json      # Funcionários cadastrados
├── logs/
│   └── presence_*.log      # Logs de eventos por dia
└── wavetrack              # Binário compilado
```

## ⚙️ Configurações Comuns

### Ajustar Timeout de Saída
```yaml
# config.yaml
presence:
  timeout_minutes: 5  # ← Altere aqui (em minutos)
```

### Ajustar Sensibilidade do Sinal
```yaml
# config.yaml
presence:
  signal_threshold: -75  # ← Menor = mais sensível
                         #   -50: Muito próximo
                         #   -70: Proximidade normal
                         #   -90: Longe
```

### Mudar Porta do Servidor
```bash
./wavetrack -port 3000
```

### Usar Config Alternativo
```bash
./wavetrack -config config.production.yaml
```

## 🎓 Próximos Passos

Após configurar o básico:

1. **Leia a documentação completa**: [README.md](README.md)
2. **Explore a API**: [API_EXAMPLES.md](API_EXAMPLES.md)
3. **Faça deploy em produção**: [DEPLOY.md](DEPLOY.md)
4. **Contribua**: [CONTRIBUTING.md](CONTRIBUTING.md)

## 💡 Dicas

### Performance
- Para ambientes grandes, aumente o intervalo de scan
- Use SSD para logs em produção
- Configure log level para "warn" ou "error"

### Segurança
- Use HTTPS em produção (nginx reverse proxy)
- Adicione autenticação na interface
- Configure firewall adequadamente

### Backup
```bash
# Backup dos dados
cp -r data/ backup_data_$(date +%Y%m%d)/
```

## 🆘 Precisa de Ajuda?

- **Bug/Issue**: Abra uma issue no GitHub
- **Dúvidas**: Veja FAQ no README.md
- **Features**: Sugira no GitHub Discussions

## ✅ Checklist Inicial

- [ ] libpcap instalado
- [ ] Go 1.21+ instalado
- [ ] Interface de rede configurada
- [ ] Dependências baixadas (`go mod download`)
- [ ] Projeto compilado
- [ ] Executando com sudo/root
- [ ] Dashboard acessível no navegador
- [ ] Dispositivos sendo detectados
- [ ] Primeiro funcionário cadastrado

---

**🎉 Pronto! Você está monitorando presença via Wi-Fi!**

*Para mais detalhes, veja [SUMMARY.md](SUMMARY.md)*
