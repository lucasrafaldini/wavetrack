# 🌱 Seed de Dados Históricos

## 📋 Resumo

Agora os dados do **histórico de 7 dias** são armazenados e lidos **diretamente do banco de dados SQLite**, não mais gerados dinamicamente.

## ✅ O que foi implementado

### 1. Script de Seed (`cmd/seed_history/main.go`)
Popula o banco com dados fake realistas:
- **12 funcionários** com nomes, departamentos e dispositivos variados
- **Eventos de arrival/departure** dos últimos 7 dias
- **Variação de presença**: Mais pessoas em dias úteis, menos em finais de semana
- **Horários variados**: Chegadas entre 7h-10h, durações 4h-10h

### 2. Funções no Storage (`internal/storage/storage.go`)
Novas funções para buscar histórico:
- `GetHistory7Days()` - Busca dados dos últimos 7 dias
- `getFirstArrivalForDate()` - Primeiro horário de chegada do dia
- `getLastDepartureForDate()` - Último horário de saída do dia  
- `getOnlineDurationForDate()` - Calcula duração total online no dia

### 3. Handler Atualizado (`internal/api/handlers.go`)
O endpoint `/api/history/7days` agora:
- ✅ Busca dados **do banco de dados**
- ✅ Calcula percentuais de presença
- ✅ Formata datas e horários
- ❌ Não gera mais dados mockados em memória

## 🚀 Como Usar

### 1. Popular o banco com dados fake

```bash
# Compilar o seed
go build -o seed_history ./cmd/seed_history

# Executar
./seed_history
```

**Output esperado:**
```
🌱 Iniciando seed de dados históricos...
📝 Cadastrando funcionários fake...
   ✓ Lucas Rafaldini cadastrado
   ✓ Ana Silva cadastrado
   ...
📊 Gerando histórico dos últimos 7 dias...
📅 Dia: 28/10/2025
   ✓ Patricia Souza: 10:24 - 19:35 (9h11m)
   ...
✅ Seed concluído com sucesso!
```

### 2. Iniciar o servidor

```bash
sudo ./wavetrack
```

### 3. Visualizar no navegador

Acesse `http://localhost:8080` e clique na aba **"📊 Histórico (7 dias)"**

## 📊 Dados Gerados

### Funcionários Fake (12 no total)

| Nome               | Departamento | Dispositivo       | MAC Address         |
|--------------------|--------------|-------------------|---------------------|
| Lucas Rafaldini    | TI           | iPhone 16         | 00:11:22:33:44:01   |
| Ana Silva          | RH           | Samsung S24       | 00:11:22:33:44:02   |
| Carlos Mendes      | Vendas       | Notebook Mac      | 00:11:22:33:44:03   |
| Maria Santos       | Marketing    | iPhone 15         | 00:11:22:33:44:04   |
| João Oliveira      | TI           | Notebook Windows  | 00:11:22:33:44:05   |
| Fernanda Costa     | Financeiro   | Samsung S23       | 00:11:22:33:44:06   |
| Ricardo Lima       | TI           | Notebook Linux    | 00:11:22:33:44:07   |
| Patricia Souza     | RH           | iPhone 14         | 00:11:22:33:44:08   |
| Bruno Almeida      | Vendas       | Xiaomi 13         | 00:11:22:33:44:09   |
| Juliana Ferreira   | Marketing    | Motorola Edge     | 00:11:22:33:44:10   |
| Rafael Barbosa     | Design       | iPad Pro          | 00:11:22:33:44:11   |
| Camila Rocha       | Design       | Galaxy Tab S9     | 00:11:22:33:44:12   |

### Padrões de Presença

- **Dias úteis**: 5-9 funcionários presentes
- **Finais de semana**: 2-3 funcionários presentes
- **Horários**: Chegadas variadas entre 7h-10h
- **Duração**: Entre 4h e 10h por dia
- **Variação**: Dados aleatórios mas realistas

## 🔄 Executar Novamente

Se quiser **regenerar** os dados:

```bash
# Remove dados antigos (opcional)
sqlite3 data/wavetrack.db "DELETE FROM events WHERE mac_address LIKE '00:11:22:33:44:%'"
sqlite3 data/wavetrack.db "DELETE FROM employees WHERE mac_address LIKE '00:11:22:33:44:%'"
sqlite3 data/wavetrack.db "DELETE FROM devices WHERE mac_address LIKE '00:11:22:33:44:%'"

# Executa seed novamente
./seed_history
```

## 🗄️ Estrutura do Banco

### Tabelas Utilizadas

**employees**
- Armazena funcionários fake com custom_device_type e custom_vendor

**devices**
- Registra dispositivos com MAC addresses fake

**events**
- Guarda eventos de arrival/departure com timestamps dos últimos 7 dias

## 🎯 Diferenças da Implementação Anterior

| Aspecto              | Antes (Mockado)       | Agora (Banco)            |
|----------------------|-----------------------|--------------------------|
| Fonte dos dados      | Gerado em memória     | ✅ Banco de dados SQLite |
| Persistência         | ❌ Não persiste       | ✅ Persiste              |
| Performance          | Rápido mas efêmero    | ✅ Rápido e persistente  |
| Dados consistentes   | ❌ Mudam a cada req   | ✅ Consistentes          |
| Testável             | Parcial               | ✅ Totalmente testável   |
| Produção-ready       | ❌ Apenas demo        | ✅ Sim                   |

## 📝 Notas Técnicas

- **MAC Addresses fake**: Usam padrão `00:11:22:33:44:XX` para facilitar identificação
- **Timestamps**: Salvos no formato SQLite com timezone local
- **Cálculos**: Duração calculada pela diferença entre arrival e departure
- **Percentual**: Baseado em jornada de 8h (28.800 segundos)
- **Queries**: Otimizadas com índices em mac_address e timestamp

## 🚀 Próximos Passos

Para **dados reais** (não fake):

1. ✅ **Já está pronto!** O sistema já coleta dados reais em tempo real
2. Os dados fake são apenas para **testar a interface** do histórico
3. Conforme o sistema roda, os **dados reais** vão acumulando nos mesmos 7 dias
4. Eventualmente, os dados fake serão **substituídos naturalmente** pelos reais

Ou você pode **limpar dados fake** e começar do zero:
```bash
sqlite3 data/wavetrack.db "DELETE FROM events WHERE mac_address LIKE '00:11:22:33:44:%'"
```

## ✨ Resultado

Agora você tem:
- ✅ Dados fake **persistentes** no banco
- ✅ Interface testável com dados variados
- ✅ Sistema pronto para **migrar para dados reais**
- ✅ Histórico de 7 dias totalmente funcional
