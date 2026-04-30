# Guia - Detecção Automática de Modo Monitor

## O Que Foi Implementado

O sistema agora detecta automaticamente se a interface Wi-Fi suporta **modo monitor (802.11)** e captura dados adicionais quando disponível.

### Detecção Automática

**Ao iniciar, o scanner testa:**
1. **Modo Monitor disponível?** → Usa filtro BPF `type mgt` (pacotes 802.11)
2. **Modo Monitor falha?** → Fallback para modo compatibilidade (ARP/Ethernet)

### Dados Capturados por Modo

| Campo | Modo Monitor (Linux com antena) | Modo Compatibilidade (macOS) |
|-------|----------------------------------|------------------------------|
| MAC Address | Sim | Sim |
| Vendor/Tipo | Sim (via OUI) | Sim (via OUI) |
| **RSSI (dBm)** | **-30 a -90 dBm** | 0 (N/A) |
| **Frequência** | **2400-7115 MHz** | 0 (N/A) |
| **Canal** | **1-177** | 0 (N/A) |

## Logs ao Iniciar

### Com Modo Monitor (Linux):
```
 Modo monitor 802.11 ativado - captura completa habilitada
 Dados disponíveis: RSSI, frequência, canal, taxa de transmissão
Scanner iniciado na interface wlan0
Novo dispositivo detectado: aa:bb:cc:dd:ee:ff [Apple - smartphone] | -45 dBm | 2437 MHz (Canal 6)
```

### Sem Modo Monitor (macOS):
```
Modo 802.11 não disponível, usando modo ARP (compatibilidade)
 Dados limitados: sem RSSI, frequência ou canal
Scanner iniciado na interface en0
Novo dispositivo detectado: aa:bb:cc:dd:ee:ff [Apple - smartphone]
```

## Interface Web

### Coluna "Sinal / Canal"

**Modo Monitor:**
```
━━━━━━━━━━ -45 dBm
 Canal 6 (2437 MHz)
```

**Modo Compatibilidade:**
```
N/A (modo não-monitor)
```

## Migração do Banco de Dados

Para bancos existentes, execute:

```bash
sudo sqlite3 data/wavetrack.db < migrate_add_frequency_channel.sql
```

Isso adiciona as colunas `frequency` e `channel` à tabela `devices`.

## Cálculo de Canal por Banda

O sistema calcula automaticamente o canal a partir da frequência:

### 2.4 GHz (Canais 1-14)
- **Canal 1**: 2412 MHz
- **Canal 6**: 2437 MHz
- **Canal 11**: 2462 MHz
- **Canal 14**: 2484 MHz (Japão)

### 5 GHz (Canais 32-177)
- **Canal 36**: 5180 MHz
- **Canal 149**: 5745 MHz

### 6 GHz (Wi-Fi 6E)
- **Canais 1-233**: 5955-7115 MHz

## Testando no Linux com Antena Wi-Fi

### 1. Coloque a interface em modo monitor:
```bash
sudo ip link set wlan0 down
sudo iw dev wlan0 set type monitor
sudo ip link set wlan0 up
```

### 2. Execute o WaveTrack:
```bash
sudo ./wavetrack -config config.yaml
```

### 3. Verifique os logs:
Deve aparecer:
```
 Modo monitor 802.11 ativado - captura completa habilitada
```

### 4. No dashboard:
Dispositivos mostrarão RSSI real, frequência e canal.

## API - Novos Campos

`GET /api/devices` agora retorna:

```json
{
 "mac_address": "aa:bb:cc:dd:ee:ff",
 "signal_strength": -45,
 "frequency": 2437,
 "channel": 6,
 ...
}
```

**Valores especiais:**
- `signal_strength: 0` = RSSI não disponível
- `frequency: 0` = Frequência não disponível
- `channel: 0` = Canal não disponível

## Arquivos Modificados

### Backend:
- `internal/models/models.go` - campos Frequency e Channel
- `internal/wifi/scanner.go` - detecção automática e extração RadioTap
- `internal/storage/schema.sql` - colunas frequency e channel
- `internal/storage/storage.go` - queries atualizadas
- `internal/api/handlers.go` - DeviceResponse com novos campos

### Frontend:
- `web/index.html` - coluna "Sinal / Canal" com display condicional

### Migrações:
- `migrate_add_frequency_channel.sql` - adiciona colunas no DB existente

## Benefícios do Modo Monitor

1. **RSSI preciso** - força real do sinal (-30 dBm = perto, -90 dBm = longe)
2. **Análise de cobertura** - identifica áreas com sinal fraco
3. **Detecção de canal** - verifica interferência/congestionamento
4. **Troubleshooting** - diagnóstico de problemas de conectividade
5. **Mais dispositivos** - detecta mesmo sem tráfego ARP
