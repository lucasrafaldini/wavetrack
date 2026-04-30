# WaveTrack - TODO List

## Prioridade Alta - v2.0

### Biblioteca Própria MAC Address
- [ ] **Criar módulo `internal/macdb`**
 - [ ] Interface `MacDatabase` com métodos `LookupMAC()`, `Update()`, `Cache()`
 - [ ] Implementação `IEEEDatabase` para consulta oficial IEEE OUI
 - [ ] Sistema de cache com TTL configurável
 - [ ] Fallback para base local quando offline

- [ ] **Sistema de Atualização Automática**
 - [ ] Scheduler para download periódico da base IEEE
 - [ ] Verificação de integridade dos dados
 - [ ] Rollback automático em caso de falha
 - [ ] Logs de auditoria das atualizações

- [ ] **Performance e Otimização**
 - [ ] Índices eficientes para consulta rápida
 - [ ] Compressão da base de dados
 - [ ] Lazy loading para economizar memória
 - [ ] Benchmark e testes de performance

- [ ] **Configuração e Flexibilidade**
 - [ ] Configuração de fontes de dados (IEEE, custom, local)
 - [ ] Sistema de prioridades para múltiplas fontes
 - [ ] API para adicionar OUIs customizados
 - [ ] Interface CLI para gerenciar a base

## Refatoração Técnica

### Código Atual
- [ ] **Remover base hardcoded** do `deviceid.go`
- [ ] **Migrar função `lookupVendorLocal()`** para novo módulo
- [ ] **Manter compatibilidade** com API atual durante transição
- [ ] **Testes abrangentes** para nova biblioteca

### Estrutura Proposta
```
internal/
├── macdb/
│ ├── interface.go # Interface MacDatabase
│ ├── ieee.go # Implementação IEEE OUI
│ ├── cache.go # Sistema de cache
│ ├── updater.go # Atualizador automático
│ └── fallback.go # Base local de fallback
├── deviceid/
│ └── deviceid.go # Refatorado para usar macdb
```

## Testes e Qualidade

### Cobertura de Testes
- [ ] **Unit tests** para módulo `macdb` (95%+ cobertura)
- [ ] **Integration tests** com base IEEE real
- [ ] **Performance tests** para consultas em massa
- [ ] **Stress tests** para concorrência

### Benchmarks
- [ ] Tempo de lookup < 1ms
- [ ] Memória < 50MB para base completa
- [ ] Startup time < 2s
- [ ] Cache hit ratio > 90%

## Configuração

### Arquivo de Config
```yaml
macdb:
 sources:
 - type: "ieee"
 url: "http://standards-oui.ieee.org/oui.txt"
 priority: 1
 update_interval: "24h"
 - type: "local"
 file: "data/custom_oui.json"
 priority: 2
 cache:
 max_size: 10000
 ttl: "1h"
 update:
 auto_update: true
 retry_count: 3
 timeout: "30s"
```

## Implementação Faseada

### Fase 1: Base Infrastructure
1. Criar interfaces e estruturas básicas
2. Implementar sistema de cache
3. Testes unitários básicos

### Fase 2: IEEE Integration
1. Implementar downloader da base IEEE
2. Parser para formato OUI.txt
3. Sistema de validação

### Fase 3: Advanced Features
1. Sistema de atualizações automáticas
2. Múltiplas fontes de dados
3. Otimizações de performance

### Fase 4: Migration
1. Refatorar código existente
2. Manter backward compatibility
3. Documentação completa

## Timeline Estimado

- **Fase 1**: 2 semanas
- **Fase 2**: 3 semanas
- **Fase 3**: 2 semanas
- **Fase 4**: 1 semana

**Total**: ~8 semanas para v2.0

## Ideias Futuras

### v2.1+
- [ ] Machine Learning para classificação automática
- [ ] Fingerprinting avançado de dispositivos
- [ ] API GraphQL para consultas complexas
- [ ] Dashboard de analytics da base MAC
- [ ] Integração com outras bases (WiGLE, etc.)

---

*Última atualização: 28 de Outubro de 2025*
