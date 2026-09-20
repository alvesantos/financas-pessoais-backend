# Mnemio: API (Go)

API REST do Mnemio Finanças. Arquitetura em camadas, domínio no
centro.

## Regras obrigatórias

### Commits

- **Nunca se adicione como co-autor.** Não inclua `Co-Authored-By: Claude`
  nem nenhuma outra linha de atribuição a assistente.
- Também não adicione "Generated with Claude Code" em descrições de PR.
- Mensagem no formato `tipo: descrição` (`feat`, `fix`, `refactor`, `test`,
  `chore`, `docs`), em português, no imperativo.
- O corpo explica **por quê**, não o que o diff já mostra.

### Testes

Toda funcionalidade entra com **testes unitários e testes e2e**. Não
considere uma funcionalidade pronta sem os dois.

- **Unitários**: nos casos de uso (`internal/service/`), com as portas do
  domínio substituídas por fakes. Sem banco, sem servidor. Veja
  `internal/service/auth_service_test.go` como referência.
- **E2e**: em `test/e2e/`, atrás da tag de build `e2e`. Exercitam a rota HTTP
  de ponta a ponta (requisição real, roteador real, Postgres real) e
  verificam status, corpo e o efeito no banco. Cubra o caminho feliz e as
  falhas (validação, não autorizado, conflito). Rodam contra o banco
  `financas_test`, criado e limpo pela própria suíte.
- Testes de erro são parte da funcionalidade, não um extra.
- Rode `make test-all` antes de commitar. Não commite com teste vermelho.

## Comandos

```bash
make run        # sobe a API em :8080
make test       # unitários, go test ./... -race -cover
make test-e2e   # e2e contra o Postgres (exige make db-up)
make test-all   # unitários + e2e
make vet        # go vet
make build      # compila em bin/api
make db-up      # Postgres em localhost:5434
make db-down    # para o Postgres (mantém os dados)
make db-reset   # apaga o volume e recria
make psql       # psql dentro do container
```

## Arquitetura

```
cmd/api/                  monta as dependências e sobe o servidor
internal/
  domain/                 entidades, erros e as INTERFACES (portas)
  service/                casos de uso: sem HTTP, sem SQL
  repository/postgres/    implementa as portas de persistência (pgx/v5)
  auth/                   adaptadores de bcrypt e JWT
  database/               pool e schema
  config/                 ambiente, .env e slog
  api/
    controller/  dto/  request/  response/  middleware/  router/
```

Regras que não se quebram:

- **Dependências apontam para dentro.** `service/` importa só `domain/`.
  Quem conhece pgx é `repository/postgres/`; quem conhece HTTP é `api/`.
- **As interfaces moram com quem as consome** (`domain/ports.go`), não com
  quem as implementa.
- Toda implementação de porta declara `var _ domain.X = (*Y)(nil)`.
- Controller não tem regra de negócio: decodifica, delega, serializa.

### Como adicionar uma funcionalidade

Na ordem: porta em `domain/ports.go` → repositório em
`repository/postgres/` → caso de uso em `service/` (+ teste unitário) →
DTO em `api/dto/` → controller em `api/controller/` → rota em
`api/router/router.go` (grupo livre ou autenticado) → teste e2e.

## Convenções

- Código, comentários, mensagens de erro e commits em **português**.
- **Nunca use travessão (—) em nada**: nem em texto de interface, nem em
  comentário, nem em commit, nem em documentação. Use vírgula, dois-pontos,
  ponto ou parênteses.
- Comentário explica o porquê de uma decisão, não o que a linha faz.
- **Dinheiro em centavos** (`BIGINT`), sempre positivo. O sinal vem do tipo
  (`domain.Kind.Signed`). Nenhum lugar inverte valor na mão.
- **Só `receita` soma saldo.** Despesa, cartão e investimento subtraem.
- **Lançamento sem descrição recebe o nome do tipo**, e a regra mora no
  serviço, nunca no controller.
- **Fixos não viram linhas em `transactions`**: as ocorrências são projetadas
  na leitura. Ao mexer nisso, o painel e a tela de lançamentos têm que
  continuar contando igual: os dois passam pelo mesmo `summarize`.
- **Schema muda por migração nova** em `internal/database/migrations.go`.
  Migração já commitada nunca é editada.
- Erros usam `domain.Error` com código de negócio. A tradução para status
  HTTP acontece só em `api/response/error.go`.
- Causa interna de erro vai para o log, nunca para o cliente. 5xx responde
  mensagem genérica.
- Resposta de autenticação não revela se um e-mail existe.
- Duplicidade é detectada pela constraint `UNIQUE`, não por um `SELECT`
  antes do `INSERT`.
- `.env` nunca é versionado; mudanças de configuração vão para `.env.example`.
