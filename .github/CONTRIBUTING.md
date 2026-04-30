# Contribuindo para o WaveTrack

Obrigado por contribuir! Este documento descreve como participar do projeto.

## Requisitos
- Go 1.21+
- libpcap-dev (ou WinPcap/Npcap no Windows)
- Git
- (Opcional) Docker e Docker Compose

## Como começar
1. Faça um fork e clone o repositório.
2. Crie uma branch: `git checkout -b feat/minha-feature`.
3. Instale as dependências:
 ```bash
 # Linux (Debian/Ubuntu)
 sudo apt-get install libpcap-dev

 # macOS
 brew install libpcap

 # Windows
 # Instale Npcap: https://npcap.com/
 ```
4. Compile o projeto:
 ```bash
 go build -o wavetrack cmd/wavetrack/main.go
 ```
5. Rode os testes:
 ```bash
 go test ./...
 ```

## Padrão de commits
Use o formato do Conventional Commits:

```
type(scope): descrição curta
```

Exemplos:

```text
feat(scanner): add signal strength filter
fix(tracker): correct timeout detection
docs(readme): update installation steps
refactor(storage): simplify JSON serialization
test(api): add integration tests for devices endpoint
```

**Tipos comuns:**
- `feat`: Nova funcionalidade
- `fix`: Correção de bug
- `docs`: Documentação
- `refactor`: Refatoração
- `test`: Testes
- `perf`: Performance
- `chore`: Manutenção

## Código e estilo
- Go: Siga [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` para formatação automática
- Use `go vet` para detecção de erros
- (Opcional) Use `golangci-lint` para análise completa
- JavaScript: Vanilla JS, sem frameworks
- Nomes claros e concisos
- Comentários quando necessário, não quando óbvio

## Testes
- Toda função não-trivial deve possuir ao menos um teste.
- Testes de integração para APIs.
- Mock para scanner em testes (não dependa de hardware).
- Coverage mínimo desejado: 70%.
- PRs sem testes podem receber solicitação para adicioná-los.

## Estrutura do Código
```
cmd/wavetrack/ → Main application
internal/
 ├── api/ → REST API handlers
 ├── config/ → Configuration
 ├── logger/ → Event logging
 ├── models/ → Data structures
 ├── storage/ → Persistence
 ├── tracker/ → Presence logic
 └── wifi/ → Packet capture
web/ → Frontend (HTML/CSS/JS)
```

## Discussões e issues
- Use Issues para bugs e propostas de features.
- Para bugs: descreva o contexto, passos para reproduzir, logs relevantes.
- Para features: explique o caso de uso e benefícios.
- Verifique se não há issue duplicada antes de criar.

## Pull Requests
1. Certifique-se de que os testes passam.
2. Atualize documentação se necessário.
3. Adicione screenshot se mudança visual.
4. Descreva o que foi feito e por quê.
5. Referencie issues relacionadas.

**Checklist:**
- [ ] Testes passando
- [ ] Código formatado (`gofmt`)
- [ ] Sem erros de lint (`go vet`)
- [ ] Documentação atualizada
- [ ] CHANGELOG.md atualizado (se aplicável)

## Licença
- Projeto sob licença MIT.
- Ao contribuir, você concorda que seu código será licenciado sob MIT.
- Não submeter código com licenças incompatíveis.
- Verificar licenças de dependências adicionadas.

## Privacidade e Ética
- Este projeto lida com dados de presença de pessoas.
- **IMPORTANTE**: Implemente anonimização de MAC addresses em produção.
- Documente políticas de retenção de dados.
- Respeite GDPR, LGPD e regulamentações locais.
- Obtenha consentimento dos funcionários antes do deploy.

## Código de Conduta
- Seja respeitoso e acolhedor.
- Sem toxicidade, assédio ou discriminação.
- Críticas construtivas são bem-vindas.
- Foque no código, não nas pessoas.
- Eduque quem está aprendendo.

## Comunidade
- GitHub Discussions: Para dúvidas gerais e ideias.
- Issues: Para bugs e feature requests.
- Pull Requests: Para contribuições de código.

Obrigado por ajudar a construir uma ferramenta de presença simples e eficaz!
