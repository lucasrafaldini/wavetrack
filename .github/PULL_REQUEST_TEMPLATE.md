## Descrição

Explique a mudança proposta e o problema que resolve.

## Tipo de mudança

- [ ] Feature (nova funcionalidade)
- [ ] Bug fix (correção)
- [ ] Refactor (melhoria sem mudança de comportamento)
- [ ] Docs (documentação)
- [ ] Test (adicionar/corrigir testes)
- [ ] Infra/CI (workflows, Docker, deploy)

## Como testar

Passos para validar manualmente e/ou comandos relevantes.

```bash
# Exemplo
go build -o wavetrack cmd/wavetrack/main.go
sudo ./wavetrack
# Acesse http://localhost:8080
```

## Screenshots (se aplicável)

Adicione capturas de tela para mudanças visuais na dashboard.

## Checklist

- [ ] Adiciona/atualiza testes quando aplicável
- [ ] Atualiza documentação (README/QUICKSTART/API_EXAMPLES) quando aplicável
- [ ] Mantém escopo pequeno e commits claros (Conventional Commits)
- [ ] Sem quebras de compatibilidade não documentadas
- [ ] Lint/CI passando (`go vet ./...` e `go test ./...`)
- [ ] Respeita licenciamento MIT
- [ ] Considera impacto em privacidade (GDPR/LGPD) se aplicável

## Performance

- [ ] Testado com >100 dispositivos (se relevante)
- [ ] Sem memory leaks em scanner/tracker (se relevante)
- [ ] Logs não excessivos (se relevante)

## Relacionado

Relacione Issues/Milestones (ex.: Closes #123, Part of #456, Relates to v0.4)

