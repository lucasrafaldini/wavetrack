# Guia de Atualização - Edição de Funcionários e Campos Customizados

## Mudanças Implementadas

### 1. Banco de Dados
- **Schema atualizado** (`internal/storage/schema.sql`):
 - Adicionadas colunas `custom_device_type` e `custom_vendor` na tabela `employees`

- **Migração** (`migrate_add_custom_fields.sql`):
 - Script SQL para adicionar as novas colunas em bancos existentes

**Como aplicar a migração em banco existente:**
```bash
sqlite3 data/wavetrack.db < migrate_add_custom_fields.sql
```

### 2. Backend (Go)

#### Models (`internal/models/models.go`)
- `Employee` agora inclui:
 - `CustomDeviceType string` - Tipo de dispositivo definido pelo usuário
 - `CustomVendor string` - Fabricante definido pelo usuário

#### Storage (`internal/storage/storage.go`)
- `SaveEmployee`: salva campos customizados
- `GetEmployeeByMAC`: carrega campos customizados
- `GetAllEmployees`: carrega campos customizados

#### API (`internal/api/handlers.go`)
- `AssociateDeviceRequest`: aceita `custom_device_type` e `custom_vendor`
- `handleDevices`: retorna tipo/vendor customizado quando disponível (sobrescreve detecção automática)

### 3. Frontend (web/index.html)

#### Modal de Cadastro/Edição
- **Novo campo: Tipo de Dispositivo** (dropdown):
 - Notebooks: Windows, Mac, Linux
 - Smartphones: iPhone (15, 14, 13, 12), Samsung (S24, S23, A54), Xiaomi, Motorola
 - Tablets: iPad (Pro, Air, padrão), Samsung Tab
 - Opção "Detectado automaticamente" (valor vazio)

- **Novo campo: Fabricante** (dropdown):
 - Apple, Samsung, Xiaomi, Motorola, Dell, HP, Lenovo, Asus, Acer, LG, Huawei, OnePlus, Google, Microsoft, Sony
 - Opção "Detectado automaticamente" (valor vazio)

#### Botão Editar
- **Antes**: Dispositivos cadastrados mostravam apenas " Cadastrado"
- **Agora**: Botão amarelo " Editar" permite modificar dados do funcionário

#### Funções Novas
- `openEditModal(mac)`: abre modal preenchido com dados atuais do funcionário
- `getDeviceTypeName()`: reconhece tipos customizados e exibe nome amigável
- `getDeviceTypeIcon()`: identifica ícone correto para tipos customizados

## Como Usar

### 1. Cadastrar Novo Funcionário
1. Clique em "Cadastrar" ao lado de um dispositivo não associado
2. Preencha nome e departamento
3. (Opcional) Selecione tipo de dispositivo específico
4. (Opcional) Selecione fabricante
5. Clique em "Salvar"

### 2. Editar Funcionário Existente
1. Clique no botão amarelo " Editar" ao lado do funcionário
2. Modifique qualquer campo (nome, departamento, tipo, fabricante)
3. Clique em "Salvar"

### 3. Comportamento dos Campos Customizados
- Se **não preenchidos**: sistema usa detecção automática por OUI do MAC
- Se **preenchidos**: valor customizado sobrescreve detecção automática
- Útil quando:
 - Detecção automática falha
 - Quer especificar modelo exato (ex: "iPhone 15" em vez de genérico "smartphone")
 - Dispositivo não tem vendor conhecido no banco OUI

## Exemplo de Uso

### Cenário 1: Detecção Automática Funcionou
- MAC detectado como "Apple" e "smartphone"
- Cadastra funcionário sem preencher tipo/fabricante
- Sistema mantém "Apple" e "smartphone"

### Cenário 2: Quer Ser Mais Específico
- MAC detectado como "Apple" e "smartphone"
- Cadastra e escolhe "iPhone 15" no dropdown de tipo
- Dashboard agora mostra " iPhone 15" em vez de genérico " Smartphone"

### Cenário 3: Detecção Falhou
- MAC desconhecido, aparece como "unknown" e "Desconhecido"
- Edita funcionário e escolhe "Notebook Windows" e "Dell"
- Dashboard agora mostra " Notebook Windows" e "Dell" corretamente

## Validação

- Build sem erros
- Schema SQL atualizado
- API aceita novos campos
- Frontend envia e exibe campos customizados
- Migração SQL pronta para bancos existentes

## Próximos Passos (Opcional)

- [ ] Adicionar validação de duplicatas (mesmo funcionário com múltiplos MACs)
- [ ] Histórico de edições
- [ ] Exportação de relatório com tipos customizados
- [ ] Bulk edit (editar múltiplos funcionários de uma vez)
