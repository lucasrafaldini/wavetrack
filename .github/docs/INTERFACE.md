# WaveTrack - Visão Geral da Interface

## Dashboard Principal

### Estatísticas em Tempo Real
```
┌─────────────────────────────────────────────────────────────────┐
│ WaveTrack │
│ Sistema de Monitoramento de Presença Wi-Fi │
└─────────────────────────────────────────────────────────────────┘

┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌───────────────┐
│ 15 │ │ 8 │ │ 5 │ │ 3 │
│ Dispositivos │ │ Dispositivos │ │ Funcionários │ │ Funcionários │
│ Detectados │ │ Ativos │ │ Cadastrados │ │ Presentes │
└───────────────┘ └───────────────┘ └───────────────┘ └───────────────┘
```

### Tabela de Dispositivos
```
┌─────────────────────────────────────────────────────────────────────────────────┐
│ Dispositivos Detectados │
├────────┬─────────────────────┬─────────────────┬──────────┬─────────┬──────────┤
│ Status │ MAC Address │ Funcionário │ Sinal │ Visto │ Ação │
├────────┼─────────────────────┼─────────────────┼──────────┼─────────┼──────────┤
│ Ativo│ 00:11:22:33:44:55 │ João Silva │ ▓▓▓▓▓░░ │ Agora │ Cadast.│
│ │ │ │ -45 dBm │ mesmo │ │
├────────┼─────────────────────┼─────────────────┼──────────┼─────────┼──────────┤
│ Ativo│ AA:BB:CC:DD:EE:FF │ Maria Santos │ ▓▓▓░░░░ │ 2 min │ Cadast.│
│ │ │ │ -60 dBm │ atrás │ │
├────────┼─────────────────────┼─────────────────┼──────────┼─────────┼──────────┤
│ Ativo│ 11:22:33:44:55:66 │ Não cadastrado │ ▓▓▓▓░░░ │ 1 min │[Cadastrar]│
│ │ │ │ -50 dBm │ atrás │ │
├────────┼─────────────────────┼─────────────────┼──────────┼─────────┼──────────┤
│ Inativo│ FF:EE:DD:CC:BB:AA│ Carlos Souza │ ▓░░░░░░ │ 8 min │ Cadast.│
│ │ │ │ -85 dBm │ atrás │ │
└────────┴─────────────────────┴─────────────────┴──────────┴─────────┴──────────┘
```

## Modal de Cadastro

Quando o gerente clica em "Cadastrar" em um dispositivo desconhecido:

```
┌─────────────────────────────────────────────┐
│ │
│ Cadastrar Funcionário │
│ │
│ ┌───────────────────────────────────────┐ │
│ │ MAC Address │ │
│ │ 11:22:33:44:55:66 │ │
│ └───────────────────────────────────────┘ │
│ │
│ ┌───────────────────────────────────────┐ │
│ │ Nome do Funcionário * │ │
│ │ [Digite o nome completo________] │ │
│ └───────────────────────────────────────┘ │
│ │
│ ┌───────────────────────────────────────┐ │
│ │ Departamento │ │
│ │ [Ex: TI, RH, Vendas_________] │ │
│ └───────────────────────────────────────┘ │
│ │
│ ┌──────────────┐ ┌──────────────┐ │
│ │ Salvar │ │ Cancelar │ │
│ └──────────────┘ └──────────────┘ │
│ │
└─────────────────────────────────────────────┘
```

## Responsividade

A interface se adapta a diferentes tamanhos de tela:

### Desktop (> 1200px)
- 4 cards de estatísticas lado a lado
- Tabela completa com todas as colunas
- Modal centralizado

### Tablet (768px - 1200px)
- 2 cards de estatísticas por linha
- Tabela com scroll horizontal
- Fonte ajustada

### Mobile (< 768px)
- 1 card por linha
- Tabela em formato de cards
- Menu responsivo

## Paleta de Cores

```css
/* Primária */
--primary: #667eea (Roxo/Azul)
--secondary: #764ba2 (Roxo escuro)

/* Status */
--success: #28a745 (Verde - Ativo)
--danger: #dc3545 (Vermelho - Inativo)
--warning: #ffc107 (Amarelo - Alerta)

/* Neutros */
--gray-100: #f8f9fa
--gray-600: #666
--white: #ffffff
```

## Fluxo de Uso

### Para o Gerente:

1. **Acessa o Dashboard**
 ```
 http://localhost:8080
 ```

2. **Visualiza Dispositivos em Tempo Real**
 - Vê todos os dispositivos detectados
 - Status (ativo/inativo)
 - Força do sinal
 - Último visto

3. **Identifica Dispositivos Novos**
 - Dispositivos sem nome aparecem como "Não cadastrado"
 - Botão "Cadastrar" disponível

4. **Cadastra Funcionário**
 - Clica em "Cadastrar"
 - Preenche nome e departamento
 - Salva

5. **Monitora Presença**
 - Dashboard atualiza automaticamente a cada 5 segundos
 - Vê quem está presente
 - Recebe notificações de chegada/saída (nos logs)

## Indicadores Visuais

### Sinal Wi-Fi
```
Excelente: ▓▓▓▓▓▓▓ (-30 a -50 dBm)
Bom: ▓▓▓▓▓░░ (-50 a -60 dBm)
Regular: ▓▓▓░░░░ (-60 a -70 dBm)
Fraco: ▓▓░░░░░ (-70 a -80 dBm)
Muito Fraco: ▓░░░░░░░ (-80 a -100 dBm)
```

### Status de Atividade
```
 Ativo - Visto nos últimos 2 minutos
 Inativo - Não visto há mais de 2 minutos
```

### Badges de Cadastro
```
 Cadastrado - Funcionário já associado
[Cadastrar] - Dispositivo sem funcionário
```

## Animações

- **Fade in**: Cards e tabela ao carregar
- **Pulse**: Stats ao atualizar valores
- **Hover**: Efeito suave em botões e linhas da tabela
- **Spin**: Ícone de refresh ao clicar
- **Slide in**: Modal de cadastro

## Notificações (Futuro)

```
┌─────────────────────────────────────────┐
│ João Silva chegou │
│ Departamento: TI │
│ Há 2 minutos │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│ Maria Santos saiu │
│ Departamento: RH │
│ Há 1 minuto │
└─────────────────────────────────────────┘
```

## Features Visuais Especiais

### Auto-refresh
- Indicador visual no botão de refresh
- Contador de última atualização
- Animação sutil ao atualizar

### Filtros (Futuro)
```
┌─────────────────────────────────────────────────────────────────┐
│ Filtros: [Todos] [Ativos] [Inativos] [Cadastrados] [Não Cad.] │
└─────────────────────────────────────────────────────────────────┘
```

### Busca (Futuro)
```
┌─────────────────────────────────────────────────────────────────┐
│ [Buscar por nome, MAC ou departamento...] │
└─────────────────────────────────────────────────────────────────┘
```

## UX/UI Highlights

 **Simplicidade**: Interface limpa e intuitiva
 **Feedback Imediato**: Ações confirmadas visualmente
 **Tempo Real**: Dados atualizados automaticamente
 **Responsivo**: Funciona em qualquer dispositivo
 **Cores Consistentes**: Paleta coerente
 **Tipografia Clara**: Fácil leitura
 **Espaçamento Adequado**: Organização visual
 **Ações Óbvias**: Botões claros e acessíveis

## Métricas de Performance

- **Tempo de Carregamento**: < 500ms
- **Atualização de Dados**: 5 segundos
- **Responsividade da API**: < 100ms
- **Tamanho do Bundle HTML**: ~15KB
- **Sem Dependências JS**: 0 libs externas

## Próximas Melhorias de UI

- [ ] Dark mode
- [ ] Gráficos de presença ao longo do dia
- [ ] Timeline de eventos
- [ ] Notificações push
- [ ] Exportar relatórios (PDF/Excel)
- [ ] Multi-idioma (i18n)
- [ ] Customização de tema
- [ ] Atalhos de teclado
