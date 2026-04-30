# Changelog

Todas as mudanças notáveis do projeto WaveTrack serão documentadas neste arquivo.

O formato é baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.0.0/),
e este projeto adere ao [Semantic Versioning](https://semver.org/lang/pt-BR/).

## [1.0.0] - 2025-10-28

### Release Inicial - Production Ready

Esta é a primeira versão completa e estável do WaveTrack, pronta para uso em produção.

### Adicionado

#### Interface Web Completa
- **Sistema de Abas**: Organização em 3 abas principais
 - **Tempo Real**: Monitoramento de dispositivos ativos
 - **Histórico (7 dias)**: Visualização de presença dos últimos 7 dias
 - **Colaboradores**: Gerenciamento completo (CRUD)
- **Organização de Código**: HTML, CSS e JavaScript em arquivos separados
 - `web/index.html` - Estrutura
 - `web/css/style.css` - Estilos
 - `web/js/app.js` - Lógica

#### Gerenciamento de Colaboradores
- **CRUD Completo**: Criar, visualizar, editar e deletar colaboradores
- **Edição Inteligente**:
 - Alteração de MAC com migração automática de histórico
 - Atualização de nome refletida em todos os eventos
 - Edição de departamento e tipo de dispositivo
- **Exclusão com Preservação**: Remover colaborador mantém histórico completo
- **Validações**: Campos obrigatórios e confirmação de deleção

#### Histórico e Relatórios
- **Histórico de 7 Dias**: Visualização completa de presença
 - Horários de chegada e saída por dia
 - Total de horas trabalhadas
 - Percentual de presença (8h = 100%)
 - Média semanal por colaborador
- **Preservação Permanente**: Histórico nunca é deletado
- **Migração Automática**: Eventos seguem o colaborador ao alterar MAC

#### API REST
- `GET /api/employees` - Lista todos os colaboradores
- `POST /api/associate` - Cria/atualiza colaborador (suporta `old_mac_address`)
- `DELETE /api/employees/{mac}` - Remove colaborador (preserva histórico)
- `GET /api/history/7days` - Retorna histórico dos últimos 7 dias
- `GET /api/devices` - Lista dispositivos detectados
- `GET /api/stats` - Estatísticas do sistema

#### Banco de Dados
- **Schema Atualizado**: Sem `ON DELETE CASCADE` para preservar histórico
- **Migração**: `migrate_preserve_history.sql` aplicada
- **Funções de Storage**:
 - `UpdateHistoryMAC()` - Migra histórico ao alterar MAC
 - `UpdateHistoryEmployee()` - Atualiza nome em eventos
 - `DeleteEmployee()` - Remove colaborador preservando eventos

#### Ferramentas de Desenvolvimento
- **Seed History**: Gerador de dados históricos para testes
 - 12 colaboradores fake com padrões realistas
 - 7 dias de eventos de presença
 - Variação de horários e durações
- **Documentação Completa**:
 - `HISTORY_PRESERVATION.md` - Preservação de histórico
 - `SEED_HISTORY.md` - Gerador de dados
 - `TABS_HISTORICO.md` - Sistema de abas
 - `CHANGELOG.md` - Este arquivo

### Alterado

#### Interface
- Dashboard reorganizado em sistema de abas para melhor UX
- Cálculo de percentual ajustado (8 horas = 100%, sem cap)
- Estilos modernizados com melhor responsividade
- Ordenação e filtros aprimorados

#### Backend
- Refatoração de handlers para suportar edição com histórico
- Logs mais detalhados de operações críticas
- Validações aprimoradas em todas as operações

### Removido

#### Arquivos Obsoletos
- `data/employees.json` - Substituído por SQLite
- `web/index_old.html` - Backup não mais necessário
- `internal/api/handlers_old.go.bak` - Backup de desenvolvimento
- `internal/storage/storage_json.go.bak` - Sistema JSON antigo
- `migrate_add_custom_fields.sql` - Migração já aplicada
- `migrate_add_frequency_channel.sql` - Migração já aplicada

#### Funcionalidades
- Sistema de storage JSON (substituído por SQLite)
- Configuração de employees em `config.yaml` (agora via interface)

### Corrigido

- **Percentual de Presença**: Corrigido para usar 8 horas como base (não mais 9h)
- **Limite de 100%**: Removido cap para mostrar horas extras corretamente
- **MAC Readonly**: Campo MAC agora editável em modo de edição
- **Duplicação de Eventos**: Event listener duplicado removido
- **Cascade Delete**: Histórico não é mais deletado ao remover colaborador

### Segurança

- Confirmação obrigatória antes de deletar colaborador
- Validação de campos obrigatórios em todas as operações
- Logs de auditoria para alterações de MAC e nome

### Estatísticas da Release

- **Linhas de Código**: ~3.500 linhas (Go + JS + CSS + HTML)
- **Arquivos**: 25+ arquivos de código
- **Endpoints API**: 6 endpoints REST
- **Cobertura de Testes**: Storage layer testado
- **Documentação**: 8 arquivos markdown (~2.500 linhas)

### Próximos Passos (v1.1)

- Relatórios em PDF/CSV
- Exportação de dados históricos
- Gráficos e dashboards analíticos
- Notificações por webhook/email
- Autenticação e controle de acesso

---

## [0.x.x] - Desenvolvimento

Versões anteriores foram de desenvolvimento interno e não foram documentadas formalmente.

---

**Formato do Changelog**: [Keep a Changelog](https://keepachangelog.com/pt-BR/1.0.0/)
**Versionamento**: [Semantic Versioning](https://semver.org/lang/pt-BR/)
