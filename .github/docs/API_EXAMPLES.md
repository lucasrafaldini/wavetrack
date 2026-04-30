# Exemplos de uso da API WaveTrack

Este arquivo contém exemplos de como interagir com a API REST do WaveTrack usando curl.

## Listar todos os dispositivos detectados

```bash
curl http://localhost:8080/api/devices
```

## Listar funcionários cadastrados

```bash
curl http://localhost:8080/api/employees
```

## Obter estatísticas do sistema

```bash
curl http://localhost:8080/api/stats
```

## Associar um dispositivo a um funcionário

```bash
curl -X POST http://localhost:8080/api/associate \
 -H "Content-Type: application/json" \
 -d '{
 "mac_address": "AA:BB:CC:DD:EE:FF",
 "employee_name": "Maria Santos",
 "department": "RH"
 }'
```

## Exemplos com jq (formatação JSON)

### Ver apenas dispositivos ativos

```bash
curl -s http://localhost:8080/api/devices | jq '.[] | select(.is_active == true)'
```

### Ver apenas dispositivos sem funcionário cadastrado

```bash
curl -s http://localhost:8080/api/devices | jq '.[] | select(.employee_name == null or .employee_name == "")'
```

### Contar total de dispositivos

```bash
curl -s http://localhost:8080/api/devices | jq '. | length'
```

### Listar apenas os nomes dos funcionários presentes

```bash
curl -s http://localhost:8080/api/devices | \
 jq -r '.[] | select(.is_active == true and .employee_name != null) | .employee_name' | \
 sort -u
```

## Integração com Python

```python
import requests

BASE_URL = "http://localhost:8080/api"

# Listar dispositivos
response = requests.get(f"{BASE_URL}/devices")
devices = response.json()

for device in devices:
 if device['is_active'] and not device.get('employee_name'):
 print(f"Dispositivo não cadastrado: {device['mac_address']}")

# Cadastrar funcionário
data = {
 "mac_address": "AA:BB:CC:DD:EE:FF",
 "employee_name": "Carlos Souza",
 "department": "Vendas"
}

response = requests.post(f"{BASE_URL}/associate", json=data)
if response.ok:
 print("Funcionário cadastrado com sucesso!")

# Ver estatísticas
stats = requests.get(f"{BASE_URL}/stats").json()
print(f"Funcionários presentes: {stats['present_employees']}/{stats['total_employees']}")
```

## Integração com JavaScript/Node.js

```javascript
const axios = require('axios');

const BASE_URL = 'http://localhost:8080/api';

// Listar dispositivos ativos
async function getActiveDevices() {
 const response = await axios.get(`${BASE_URL}/devices`);
 return response.data.filter(d => d.is_active);
}

// Cadastrar funcionário
async function registerEmployee(macAddress, name, department) {
 const response = await axios.post(`${BASE_URL}/associate`, {
 mac_address: macAddress,
 employee_name: name,
 department: department
 });
 return response.data;
}

// Monitorar em tempo real
async function monitor() {
 setInterval(async () => {
 const stats = await axios.get(`${BASE_URL}/stats`);
 console.log(`Presentes: ${stats.data.present_employees}/${stats.data.total_employees}`);
 }, 5000);
}

monitor();
```

## Webhook para notificações (exemplo conceitual)

Você pode criar um script que verifica periodicamente e envia notificações:

```bash
#!/bin/bash

while true; do
 PRESENT=$(curl -s http://localhost:8080/api/stats | jq '.present_employees')

 if [ $PRESENT -lt 3 ]; then
 # Enviar notificação (exemplo com Slack)
 curl -X POST https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
 -H 'Content-Type: application/json' \
 -d "{\"text\": \" Apenas $PRESENT funcionários presentes!\"}"
 fi

 sleep 300 # Verifica a cada 5 minutos
done
```
