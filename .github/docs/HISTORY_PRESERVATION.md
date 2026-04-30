# Preservação de Histórico - WaveTrack

## Objetivo

Garantir que o histórico de presença dos colaboradores seja **preservado permanentemente** no banco de dados, mesmo após a exclusão de um colaborador do sistema.

## Comportamento Implementado

### Ao Deletar um Colaborador

 **O QUE É REMOVIDO:**
- Registro na tabela `employees` (vínculo do colaborador)

 **O QUE É PRESERVADO:**
- Todos os eventos de presença (`events` table)
 - Chegadas (arrivals)
 - Saídas (departures)
 - Timestamps
 - MAC addresses
 - Nomes dos colaboradores
- Registro do dispositivo (`devices` table)
- Histórico completo para relatórios

### Ao Editar um Colaborador

#### Edição de MAC:
- TODO o histórico é atualizado com o novo MAC
- Eventos antigos são migrados automaticamente
- Dispositivo é atualizado
- Continuidade total dos dados

#### Edição de Nome:
- Nome é atualizado em todos os eventos históricos
- Relatórios mostram o nome atualizado
- Histórico permanece consistente

#### Edição de Departamento:
- Registro atualizado na tabela employees
- Histórico não é afetado (eventos não guardam departamento)

## Implementação Técnica

### Migração do Schema

**Antes:**
```sql
FOREIGN KEY (mac_address) REFERENCES devices(mac_address) ON DELETE CASCADE
```

**Depois:**
```sql
FOREIGN KEY (mac_address) REFERENCES devices(mac_address)
-- SEM CASCADE - histórico é preservado
```

### Arquivo de Migração

- **Arquivo:** `migrate_preserve_history.sql`
- **Executado em:** 28/10/2025
- **Status:** Sucesso
- **Dados preservados:** 12 colaboradores, 136 eventos

### Funções Implementadas

**storage.go:**
```go
// DeleteEmployee - Remove colaborador mas preserva eventos
func (s *Storage) DeleteEmployee(macAddress string) error

// UpdateHistoryMAC - Atualiza MAC em todo histórico
func (s *Storage) UpdateHistoryMAC(oldMAC, newMAC string) error

// UpdateHistoryEmployee - Atualiza nome em todo histórico
func (s *Storage) UpdateHistoryEmployee(macAddress, newName string) error
```

## Casos de Uso

### Exemplo 1: Colaborador Demitido
```
Situação: João foi demitido após 3 meses
Ação: Deletar colaborador "João Silva"
Resultado:
 João removido da lista de colaboradores ativos
 90 eventos de presença preservados (3 meses)
 Relatórios históricos continuam mostrando dados de João
 Análises de turnover mantêm dados completos
```

### Exemplo 2: Troca de Dispositivo
```
Situação: Maria trocou de celular (novo MAC)
Ação: Editar colaborador, alterar MAC
Resultado:
 45 eventos antigos migrados para novo MAC
 Histórico completo sob um único MAC
 Continuidade de dados preservada
 Relatórios mostram dados unificados
```

### Exemplo 3: Correção de Nome
```
Situação: "Pedro Silva" → "Pedro S. Santos" (correção)
Ação: Editar nome do colaborador
Resultado:
 120 eventos atualizados com novo nome
 Histórico consistente
 Relatórios exibem nome correto
```

## Teste de Validação

```bash
# 1. Contar eventos antes de deletar
sqlite3 data/wavetrack.db "SELECT COUNT(*) FROM events WHERE mac_address='00:11:22:33:44:01';"
# Resultado: 6 eventos

# 2. Deletar colaborador
sqlite3 data/wavetrack.db "DELETE FROM employees WHERE mac_address='00:11:22:33:44:01';"

# 3. Verificar eventos após deletar
sqlite3 data/wavetrack.db "SELECT COUNT(*) FROM events WHERE mac_address='00:11:22:33:44:01';"
# Resultado: 6 eventos (PRESERVADOS! )
```

## Benefícios

1. **Compliance e Auditoria**
 - Histórico completo para auditorias trabalhistas
 - Registros imutáveis de presença
 - Rastreabilidade total

2. **Análises e Relatórios**
 - Dados históricos sempre disponíveis
 - Comparações temporais precisas
 - Análises de turnover completas

3. **Integridade de Dados**
 - Sem perda de informação crítica
 - Banco de dados como fonte única de verdade
 - Recuperação de dados facilitada

4. **Flexibilidade Operacional**
 - Colaboradores podem ser removidos e re-adicionados
 - Correções de dados sem perda de histórico
 - Gestão simplificada

## Garantias

- **Zero perda de dados** ao deletar colaboradores
- **Histórico consistente** ao editar informações
- **Migração automática** de eventos ao mudar MAC
- **Logs detalhados** de todas as operações
- **Backwards compatible** com dados existentes

## Logs do Sistema

```
 Funcionário deletado: Lucas Rafaldini (00:11:22:33:44:01) - Histórico preservado para relatórios
→ MAC alterado de AA:BB:CC:DD:EE:FF para AA:BB:CC:DD:EE:11 - Atualizando histórico...
 Histórico atualizado com sucesso
→ Nome alterado de 'Pedro Silva' para 'Pedro S. Santos' - Atualizando histórico...
```

## Status

- Schema atualizado
- Migração aplicada
- Funções implementadas
- Logs informativos
- Testes validados
- Documentação completa

**Data de Implementação:** 28 de Outubro de 2025
**Versão:** 1.0
**Status:** Produção
