# WaveTrack - Roadmap de Desenvolvimento

## Versões Lançadas

### v1.0 - Sistema Base (Outubro 2025)
- [x] Captura Wi-Fi com detecção de dispositivos
- [x] Interface web com sistema de abas (Tempo Real, Histórico, Colaboradores)
- [x] CRUD completo de colaboradores via interface
- [x] API REST (6 endpoints)
- [x] Banco de dados SQLite com histórico permanente
- [x] Edição com migração automática de histórico
- [x] Preservação de dados ao deletar colaboradores

### v2.0 - Sistema de Identificação Avançado (Abril 2026)
- [x] **Integração com OUIja** - Biblioteca própria para consulta MAC Address
  - Base IEEE oficial via Wireshark (38k+ OUIs vs ~50 hardcoded)
  - Cache inteligente em memória
  - Atualização automática (TTL 7 dias)
- [x] **API de vendor lookup** - 4 novos endpoints
  - `POST /api/vendor/details` - Detalhes de um MAC
  - `POST /api/vendor/search` - Busca fabricantes por padrão
  - `GET /api/vendor/top` - Top fabricantes por OUIs registrados
  - `GET /api/vendor/stats` - Estatísticas da base OUI
- [x] **Remoção da base OUI hardcoded** - Código reduzido de 738 para 507 linhas
- [x] **CI/CD com GitHub Actions**
  - Lint: go vet, gofmt, go mod tidy, golangci-lint
  - Testes com race detector e coverage report
  - Benchmarks com benchmem
  - Build matrix (linux/amd64, darwin/arm64, darwin/amd64)

## Próximas Versões

### v2.1 - Interface e UX
- [ ] Interface web responsiva (mobile-first)
- [ ] Dashboard com gráficos interativos
- [ ] Sistema de notificações push
- [ ] Interface web para configurações
- [ ] Profiles de detecção personalizáveis

### v2.2 - Recursos Empresariais
- [ ] Relatórios de presença (PDF/CSV/Excel)
- [ ] Notificações por webhook (email/Slack)
- [ ] Suporte a múltiplos dispositivos por colaborador
- [ ] Autenticação e controle de acesso
- [ ] Integração com sistemas de RH

### v3.0 - Escalabilidade
- [ ] Suporte a múltiplos scanners
- [ ] Arquitetura distribuída
- [ ] Métricas Prometheus/Grafana
- [ ] Docker e Docker Compose

## Melhorias Técnicas Contínuas

### Identificação de Dispositivos
- [x] Biblioteca MAC Address própria (OUIja)
- [ ] Fingerprinting avançado baseado em comportamento
- [ ] Detecção de dispositivos virtualizados/emulados
- [ ] Identificação de IoT devices por padrões de tráfego

### Performance e Confiabilidade
- [ ] Otimização de memória para ambientes limitados
- [ ] Sistema de health checks
- [ ] Métricas Prometheus/Grafana

### Segurança
- [ ] Autenticação e autorização
- [ ] Criptografia de dados sensíveis
- [ ] Auditoria de ações
- [ ] LGPD/GDPR compliance

## Metas de Qualidade

- **Cobertura de Testes**: 80%+
- **Performance**: < 100ms tempo de resposta API
- **Uptime**: 99.9%
- **Compatibilidade**: Linux, macOS
- **Documentação**: Completa e atualizada

## Cronograma Estimado

- **v2.0**: Q2 2026 - Concluído
- **v2.1**: Q3 2026 (Interface Responsiva)
- **v2.2**: Q4 2026 (Recursos Empresariais)
- **v3.0**: Q1 2027 (Escalabilidade)

---

*Última atualização: Abril 2026*
