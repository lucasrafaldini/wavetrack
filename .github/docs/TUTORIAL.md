# Tutorial Visual - WaveTrack

Este documento simula um tutorial passo a passo de como usar o WaveTrack.

## Cena 1: Compilação e Execução

```bash
# Terminal mostra:
$ cd wavetrack
$ go build -o wavetrack cmd/wavetrack/main.go
# [Compilando...]

$ sudo ./wavetrack
Password: ********

=== WaveTrack - Sistema de Monitoramento de Presença ===
Configuração carregada: interface=en0, intervalo=10s
Sistema de armazenamento iniciado
Sistema de logs iniciado: ./logs/presence_2025-10-28.log
Scanner iniciado na interface en0
Monitorando 0 funcionários cadastrados
 Servidor web iniciado em http://localhost:8080
 Acesse o dashboard no navegador!
Sistema iniciado! Pressione Ctrl+C para parar...
```

## Cena 2: Acessando o Dashboard

```
[Navegador abre em http://localhost:8080]

┌─────────────────────────────────────────────────────────────┐
│ WaveTrack │
│ Sistema de Monitoramento de Presença Wi-Fi │
└─────────────────────────────────────────────────────────────┘

 ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
 │ 5 │ │ 3 │ │ 0 │ │ 0 │
 │Dispos. │ │ Ativos │ │Funcion.│ │Present.│
 └────────┘ └────────┘ └────────┘ └────────┘

[Animação: Números aparecem com fade-in]
```

## Cena 3: Primeiro Dispositivo Detectado

```
[Console do servidor:]
Dispositivo detectado: A4:83:E7:1B:2C:3D (Sinal: -52 dBm)

[Dashboard atualiza automaticamente:]

┌──────────────────────────────────────────────────────────────┐
│ Dispositivos Detectados │
├────────┬─────────────────┬──────────────┬────────┬───────────┤
│Status. │ MAC Address │ Funcionário │ Sinal │ Ação │
├────────┼─────────────────┼──────────────┼────────┼───────────┤
│ Ativo│A4:83:E7:1B:2C:3D│ Não cadast. │▓▓▓▓▓░░ │[Cadastrar]│
│ │ │ │-52 dBm │ │
└────────┴─────────────────┴──────────────┴────────┴───────────┘

[Botão [Cadastrar] pulsa suavemente para chamar atenção]
```

## Cena 4: Cadastrando Primeiro Funcionário

```
[Usuário clica no botão [Cadastrar]]

[Modal aparece com animação slide-in:]

┌─────────────────────────────────────────────┐
│ │
│ Cadastrar Funcionário │
│ │
│ ┌───────────────────────────────────────┐ │
│ │ MAC Address │ │
│ │ A4:83:E7:1B:2C:3D │ │ [Readonly]
│ └───────────────────────────────────────┘ │
│ │
│ ┌───────────────────────────────────────┐ │
│ │ Nome do Funcionário * │ │
│ │ João Silva█ │ │ [Usuário digitando]
│ └───────────────────────────────────────┘ │
│ │
│ ┌───────────────────────────────────────┐ │
│ │ Departamento │ │
│ │ TI█ │ │ [Usuário digitando]
│ └───────────────────────────────────────┘ │
│ │
│ ┌──────────────┐ ┌──────────────┐ │
│ │ Salvar │ │ Cancelar │ │
│ └──────────────┘ └──────────────┘ │
└─────────────────────────────────────────────┘

[Usuário clica em "Salvar"]
[Modal fecha com animação fade-out]
```

## Cena 5: Funcionário Cadastrado

```
[Notificação aparece no topo:]
┌─────────────────────────────────────┐
│ Funcionário cadastrado com sucesso!│
└─────────────────────────────────────┘

[Console do servidor:]
Dispositivo A4:83:E7:1B:2C:3D associado a João Silva
 João Silva chegou (Departamento: TI)

[Dashboard atualiza:]

 ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
 │ 5 │ │ 3 │ │ 1 │ │ 1 │
 │Dispos. │ │ Ativos │ │Funcion.│ │Present.│
 └────────┘ └────────┘ └────────┘ └────────┘
 ↑ ↑
 [Verde] [Verde]

┌─────────────────────────────────────────────────────────────┐
│ Dispositivos Detectados │
├──────┬─────────────────┬──────────────┬────────┬──────────┤
│Status│ MAC Address │ Funcionário │ Sinal │ Ação │
├──────┼─────────────────┼──────────────┼────────┼──────────┤
│ Ativo│A4:83:E7:1B:2C:3D│ João Silva │▓▓▓▓▓░░│ Cadastr.│
│ │ │ │-52 dBm │ │
└──────┴─────────────────┴──────────────┴────────┴──────────┘
 ↑
 [Destaque em azul]
```

## Cena 6: Mais Dispositivos Chegando

```
[2 minutos depois...]

[Console:]
Dispositivo detectado: 5C:F9:DD:4A:8E:12 (Sinal: -48 dBm)
Dispositivo detectado: B8:27:EB:F3:11:AA (Sinal: -65 dBm)

[Dashboard mostra 3 dispositivos:]

┌────────────────────────────────────────────────────────────────┐
│ Status │ MAC Address │ Funcionário │ Sinal │ Ação │
├────────┼──────────────────┼──────────────┼──────────┼──────────┤
│ Ativo │A4:83:E7:1B:2C:3D │ João Silva │▓▓▓▓▓░░ │ Cadastr.│
│ │ │ │ -52 dBm │ │
├────────┼──────────────────┼──────────────┼──────────┼──────────┤
│ Ativo │5C:F9:DD:4A:8E:12 │ Não cadast. │▓▓▓▓▓▓░ │[Cadastrar]│
│ │ │ │ -48 dBm │ │ [Novo!]
├────────┼──────────────────┼──────────────┼──────────┼──────────┤
│ Ativo │B8:27:EB:F3:11:AA │ Não cadast. │▓▓▓░░░░ │[Cadastrar]│
│ │ │ │ -65 dBm │ │ [Novo!]
└────────┴──────────────────┴──────────────┴──────────┴──────────┘
```

## Cena 7: Cadastrando Mais Funcionários

```
[Usuário cadastra rapidamente:]

1. Clica [Cadastrar] no 2º dispositivo
 → Nome: "Maria Santos"
 → Depto: "RH"
 → [Salvar]

[Console:]
 Maria Santos chegou (Departamento: RH)

2. Clica [Cadastrar] no 3º dispositivo
 → Nome: "Carlos Souza"
 → Depto: "Vendas"
 → [Salvar]

[Console:]
 Carlos Souza chegou (Departamento: Vendas)

[Dashboard atualizado:]

 ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
 │ 5 │ │ 3 │ │ 3 │ │ 3 │
 │Dispos. │ │ Ativos │ │Funcion.│ │Present.│
 └────────┘ └────────┘ └────────┘ └────────┘

Todos os 3 dispositivos agora mostram nomes!
```

## Cena 8: Funcionário Saindo

```
[5 minutos depois...]
[João Silva sai do escritório / desliga Wi-Fi]

[Console:]
 João Silva saiu

[Dashboard atualiza automaticamente:]

 ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
 │ 5 │ │ 2 │ │ 3 │ │ 2 │
 │Dispos. │ │ Ativos │ │Funcion.│ │Present.│
 └────────┘ └────────┘ └────────┘ └────────┘
 ↓ ↓
 [Mudou] [Mudou]

┌────────────────────────────────────────────────────────────────┐
│ Status │ MAC Address │ Funcionário │ Sinal │ Ação │
├────────┼──────────────────┼───────────────┼─────────┼──────────┤
│ Inativo│A4:83:E7:1B:2C:3D│ João Silva │▓░░░░░░ │ Cadastr.│
│ │ │ │ -95 dBm │ │
│ │ │ │5 min atrás│ │ [Ficou vermelho!]
├────────┼──────────────────┼───────────────┼─────────┼──────────┤
│ Ativo │5C:F9:DD:4A:8E:12 │ Maria Santos │▓▓▓▓▓▓░ │ Cadastr.│
│ Ativo │B8:27:EB:F3:11:AA │ Carlos Souza │▓▓▓░░░░ │ Cadastr.│
└────────┴──────────────────┴───────────────┴─────────┴──────────┘
```

## Cena 9: Novo Dispositivo Desconhecido

```
[Um visitante chega...]

[Console:]
Dispositivo detectado: 3A:FF:1C:8B:9D:45 (Sinal: -55 dBm)

[Dashboard:]

┌────────────────────────────────────────────────────────────────┐
│ Status │ MAC Address │ Funcionário │ Sinal │ Ação │
├────────┼──────────────────┼───────────────┼─────────┼──────────┤
│ Ativo │3A:FF:1C:8B:9D:45 │ Não cadastrado│▓▓▓▓░░░ │[Cadastrar]│
│ │ │ │ -55 dBm │ │ [Novo visitante]
│ Inativo│A4:83:E7:1B:2C:3D│ João Silva │▓░░░░░░ │ Cadastr.│
│ Ativo │5C:F9:DD:4A:8E:12 │ Maria Santos │▓▓▓▓▓▓░ │ Cadastr.│
│ Ativo │B8:27:EB:F3:11:AA │ Carlos Souza │▓▓▓░░░░ │ Cadastr.│
└────────┴──────────────────┴───────────────┴─────────┴──────────┘

[Gerente decide não cadastrar - é apenas um visitante temporário]
```

## Cena 10: Verificando Logs

```
[Terminal 2:]

$ tail -f logs/presence_2025-10-28.log

{"timestamp":"2025-10-28T14:30:15Z","employee_id":"001","employee_name":"João Silva","mac_address":"A4:83:E7:1B:2C:3D","event_type":"arrival","signal":-52}

{"timestamp":"2025-10-28T14:32:20Z","employee_id":"002","employee_name":"Maria Santos","mac_address":"5C:F9:DD:4A:8E:12","event_type":"arrival","signal":-48}

{"timestamp":"2025-10-28T14:33:45Z","employee_id":"003","employee_name":"Carlos Souza","mac_address":"B8:27:EB:F3:11:AA","event_type":"arrival","signal":-65}

{"timestamp":"2025-10-28T14:40:00Z","employee_id":"001","employee_name":"João Silva","mac_address":"A4:83:E7:1B:2C:3D","event_type":"departure","signal":0}

{"timestamp":"2025-10-28T14:45:30Z","mac_address":"3A:FF:1C:8B:9D:45","event_type":"unknown_device","signal":-55}

[Logs estruturados e fáceis de processar!]
```

## Cena 11: Usando a API

```
[Terminal 3:]

$ curl http://localhost:8080/api/stats

{
 "total_devices": 5,
 "active_devices": 3,
 "total_employees": 3,
 "present_employees": 2,
 "unknown_devices": 1
}

$ curl http://localhost:8080/api/devices | jq '.[] | select(.employee_name != null)'

[
 {
 "mac_address": "A4:83:E7:1B:2C:3D",
 "employee_name": "João Silva",
 "employee_id": "001",
 "is_active": false,
 ...
 },
 {
 "mac_address": "5C:F9:DD:4A:8E:12",
 "employee_name": "Maria Santos",
 "employee_id": "002",
 "is_active": true,
 ...
 }
]
```

## Cena 12: João Retorna

```
[15 minutos depois, João volta ao escritório]

[Console:]
 João Silva chegou (Departamento: TI)

[Dashboard:]

 ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
 │ 5 │ │ 4 │ │ 3 │ │ 3 │
 │Dispos. │ │ Ativos │ │Funcion.│ │Present.│
 └────────┘ └────────┘ └────────┘ └────────┘
 ↑ ↑
 [Aumentou] [Aumentou]

[Status de João volta a Ativo]
[Sistema detectou automaticamente o retorno!]
```

## Cena 13: Final do Dia

```
[18h00 - Todos saem]

[Console ao longo do tempo:]
 Carlos Souza saiu
 Maria Santos saiu
 João Silva saiu

[Dashboard vazio:]

 ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
 │ 5 │ │ 0 │ │ 3 │ │ 0 │
 │Dispos. │ │ Ativos │ │Funcion.│ │Present.│
 └────────┘ └────────┘ └────────┘ └────────┘

[Todos os dispositivos marcados como Inativo]
[Ninguém presente]
```

## Resumo do Tutorial

 **Compilou e executou** o WaveTrack
 **Acessou o dashboard** no navegador
 **Viu dispositivos sendo detectados** em tempo real
 **Cadastrou funcionários** pela interface
 **Monitorou chegadas e saídas** automaticamente
 **Identificou visitantes** não cadastrados
 **Consultou logs** estruturados
 **Usou a API REST** para integração

## Recursos Visuais Destacados

- **Auto-refresh**: Dashboard atualiza sozinho a cada 5s
- **Feedback visual**: Cores indicam status (verde/vermelho)
- **Barra de sinal**: Mostra força Wi-Fi visualmente
- **Modal intuitivo**: Cadastro simples e rápido
- **Tempo relativo**: "Agora mesmo", "5 min atrás"
- **Badges de status**: Ativo/Inativo, Cadastrado
- **Estatísticas em tempo real**: Cards no topo
- **Notificações**: Console mostra eventos

## Fim do Tutorial

**Parabéns!**

Você agora sabe como usar o WaveTrack para monitorar presença de funcionários através de seus dispositivos Wi-Fi!

---

 Para mais informações:
- [Quick Start](QUICKSTART.md)
- [Documentação Completa](SUMMARY.md)
- [Exemplos de API](API_EXAMPLES.md)
