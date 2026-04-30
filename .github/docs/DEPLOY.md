# Guia de Deploy - WaveTrack

Este guia explica como fazer o deploy do WaveTrack em diferentes ambientes.

## Deploy em Servidor Linux

### 1. Preparação do Servidor

```bash
# Atualizar sistema
sudo apt update && sudo apt upgrade -y

# Instalar dependências
sudo apt install -y git golang-go libpcap-dev

# Criar usuário para o WaveTrack (opcional mas recomendado)
sudo useradd -r -s /bin/false wavetrack
```

### 2. Clonar e Compilar

```bash
# Clonar repositório
cd /opt
sudo git clone https://github.com/lucasrafaldini/wavetrack.git
cd wavetrack

# Compilar
sudo go build -o wavetrack cmd/wavetrack/main.go

# Dar permissão ao binário para capturar pacotes
sudo setcap cap_net_raw,cap_net_admin=eip ./wavetrack
```

### 3. Configurar

```bash
# Copiar e editar configuração
sudo cp config.yaml config.production.yaml
sudo nano config.production.yaml

# Ajustar interface de rede (ex: wlan0)
# Ajustar diretórios de logs e dados
```

### 4. Criar Serviço Systemd

Crie o arquivo `/etc/systemd/system/wavetrack.service`:

```ini
[Unit]
Description=WaveTrack - Sistema de Monitoramento de Presença
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/wavetrack
ExecStart=/opt/wavetrack/wavetrack -config config.production.yaml -port 8080
Restart=always
RestartSec=10

# Logs
StandardOutput=journal
StandardError=journal

# Segurança
PrivateTmp=yes
NoNewPrivileges=yes

[Install]
WantedBy=multi-user.target
```

### 5. Iniciar o Serviço

```bash
# Recarregar systemd
sudo systemctl daemon-reload

# Habilitar início automático
sudo systemctl enable wavetrack

# Iniciar serviço
sudo systemctl start wavetrack

# Ver status
sudo systemctl status wavetrack

# Ver logs
sudo journalctl -u wavetrack -f
```

## Deploy com Docker

### 1. Criar Dockerfile

Crie o arquivo `Dockerfile`:

```dockerfile
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git libpcap-dev gcc musl-dev

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -o wavetrack cmd/wavetrack/main.go

FROM alpine:latest

RUN apk add --no-cache libpcap

WORKDIR /app
COPY --from=builder /app/wavetrack .
COPY config.yaml .
COPY web/ web/

EXPOSE 8080

# Nota: Requer privilégios para captura de pacotes
CMD ["./wavetrack", "-port", "8080"]
```

### 2. Criar docker-compose.yml

```yaml
version: '3.8'

services:
 wavetrack:
 build: .
 container_name: wavetrack
 network_mode: host
 privileged: true # Necessário para captura de pacotes
 volumes:
 - ./config.yaml:/app/config.yaml
 - ./logs:/app/logs
 - ./data:/app/data
 environment:
 - TZ=America/Sao_Paulo
 restart: unless-stopped
```

### 3. Executar

```bash
# Construir imagem
docker-compose build

# Iniciar container
docker-compose up -d

# Ver logs
docker-compose logs -f

# Parar
docker-compose down
```

## Configurar Reverse Proxy (Nginx)

### 1. Instalar Nginx

```bash
sudo apt install nginx
```

### 2. Configurar Site

Crie `/etc/nginx/sites-available/wavetrack`:

```nginx
server {
 listen 80;
 server_name wavetrack.seudominio.com;

 location / {
 proxy_pass http://localhost:8080;
 proxy_http_version 1.1;
 proxy_set_header Upgrade $http_upgrade;
 proxy_set_header Connection 'upgrade';
 proxy_set_header Host $host;
 proxy_cache_bypass $http_upgrade;
 proxy_set_header X-Real-IP $remote_addr;
 proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
 proxy_set_header X-Forwarded-Proto $scheme;
 }
}
```

### 3. Habilitar Site

```bash
sudo ln -s /etc/nginx/sites-available/wavetrack /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

### 4. SSL com Let's Encrypt (Opcional)

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d wavetrack.seudominio.com
```

## Adicionar Autenticação Básica

### Opção 1: Nginx Basic Auth

```bash
# Instalar utilitário
sudo apt install apache2-utils

# Criar arquivo de senhas
sudo htpasswd -c /etc/nginx/.htpasswd admin

# Adicionar ao nginx config
# location / {
# auth_basic "WaveTrack";
# auth_basic_user_file /etc/nginx/.htpasswd;
# ...
# }
```

## Monitoramento

### 1. Verificar Saúde do Serviço

```bash
# Status do systemd
sudo systemctl status wavetrack

# Uso de CPU e memória
ps aux | grep wavetrack

# Logs em tempo real
sudo journalctl -u wavetrack -f

# Verificar API
curl http://localhost:8080/api/stats
```

### 2. Rotação de Logs

Crie `/etc/logrotate.d/wavetrack`:

```
/opt/wavetrack/logs/*.log {
 daily
 missingok
 rotate 30
 compress
 delaycompress
 notifempty
 create 0640 root root
}
```

## Troubleshooting

### Erro: "Permission denied" ao capturar pacotes

```bash
# Dar permissões ao binário
sudo setcap cap_net_raw,cap_net_admin=eip ./wavetrack

# OU executar como root
sudo ./wavetrack
```

### Interface não encontrada

```bash
# Listar interfaces disponíveis
ip link show

# No macOS
ifconfig

# Atualizar config.yaml com a interface correta
```

### Porta já em uso

```bash
# Verificar processo usando a porta
sudo lsof -i :8080

# Usar porta diferente
./wavetrack -port 8081
```

## Acesso Remoto Seguro

### Túnel SSH

```bash
# No cliente (seu computador)
ssh -L 8080:localhost:8080 usuario@servidor

# Acesse http://localhost:8080 no navegador
```

### VPN

Configure uma VPN para acesso seguro à rede interna onde o WaveTrack está rodando.

## Atualização

```bash
# Parar serviço
sudo systemctl stop wavetrack

# Atualizar código
cd /opt/wavetrack
sudo git pull

# Recompilar
sudo go build -o wavetrack cmd/wavetrack/main.go

# Reiniciar
sudo systemctl start wavetrack
```

## Performance

Para ambientes com muitos dispositivos:

1. **Aumentar buffer de pacotes** no código
2. **Usar SSD** para logs
3. **Configurar log level** para "warn" ou "error"
4. **Considerar Redis** para cache de dispositivos
5. **Usar banco de dados** para histórico longo

## Segurança

- Execute com usuário não-privilegiado quando possível
- Use HTTPS (SSL/TLS) em produção
- Configure firewall para permitir apenas portas necessárias
- Faça backup regular dos dados
- Mantenha o sistema atualizado
- Implemente autenticação na interface web
- Monitore logs de acesso

## Suporte

Para problemas ou dúvidas, abra uma issue no GitHub ou entre em contato com o time de desenvolvimento.
