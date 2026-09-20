# Finn — API

API REST em Go do **Finn — Finanças Pessoais**. Arquitetura em camadas: o
domínio no centro, adaptadores na borda, dependências sempre apontando para
dentro.

## Stack

| Item | Escolha |
|---|---|
| Linguagem | Go 1.26 |
| HTTP | `net/http` (roteador da stdlib, com padrões de método) |
| Banco | PostgreSQL 16 via `pgx/v5` |
| Auth | JWT HS256 (`golang-jwt/jwt/v5`) + bcrypt |
| Logs | `log/slog` (texto em dev, JSON em produção) |

## Rodando

```bash
cp .env.example .env     # preencha JWT_SECRET
make db-up               # Postgres em localhost:5434
make run                 # http://localhost:8080
```

As migrações rodam sozinhas ao subir: `internal/database/migrations.go`
mantém a lista versionada e aplica o que falta, registrando cada passo em
`schema_migrations`. Bancos criados antes do controle de versão são marcados
com a versão 1 automaticamente.

## Comandos

```bash
make run        # sobe a API
make build      # compila em bin/api
make test       # testes unitários, com -race -cover
make test-e2e   # testes e2e contra o Postgres
make test-all   # unitários + e2e
make vet        # go vet
make db-up      # sobe o Postgres e espera ficar saudável
make db-down    # para o Postgres (mantém os dados)
make db-reset   # apaga o volume e recria o banco
make psql       # abre o psql no container
```

## Arquitetura

```
cmd/api/                  monta as dependências e sobe o servidor
internal/
  domain/                 entidades, erros e INTERFACES (as portas)
    user.go               User, Session, Credentials, Registration
    errors.go             Error, ErrorCode e as sentinelas
    ports.go              UserRepository, AuthService, TokenIssuer, PasswordHasher
  service/                CASOS DE USO — regras de negócio, sem HTTP nem SQL
  repository/postgres/    REPOSITORIES — implementam as portas de persistência
  auth/                   adaptadores de bcrypt e JWT
  database/               pool do Postgres e schema
  config/                 ambiente, .env e logger
  api/
    controller/           CONTROLLERS — decodificam, delegam, serializam
    dto/                  corpos de requisição/resposta e sua validação
    request/              decodificação de JSON com validação
    response/             respostas JSON e a tradução de erro → status
    middleware/           Recover, Logger, CORS, Authenticate
    router/               ROTAS livres e autenticadas
```

O fluxo de uma requisição:

```
router → middleware → controller → service → repository → Postgres
                          ↑            ↑
                         dto        domain (portas e erros)
```

Regras que valem a pena manter:

- **O domínio não importa nada de fora.** `service/` conhece só `domain/`;
  quem conhece pgx é `repository/postgres/`, quem conhece HTTP é `api/`.
- **As interfaces moram com quem as consome** (`domain/ports.go`), não com
  quem as implementa. Trocar Postgres por outro banco é escrever outro
  `UserRepository`.
- **Cada implementação declara `var _ domain.X = (*Y)(nil)`**: o compilador
  avisa se a porta deixar de ser satisfeita.

## Erros

Um único tipo, `domain.Error`, carrega um código de negócio, a mensagem
segura para o cliente, os erros por campo e a causa interna:

```go
return nil, domain.ErrEmailTaken.WithFields(map[string]string{
    "email": "e-mail já cadastrado",
})
```

`api/response.Fail` é o único lugar que traduz código em status HTTP:

| Código | Status |
|---|---|
| `validation` | 422 |
| `invalid_payload` | 400 |
| `unauthorized` | 401 |
| `not_found` | 404 |
| `conflict` | 409 |
| `unavailable` | 503 |
| `internal` | 500 |

O corpo é sempre o mesmo:

```json
{ "error": "dados inválidos", "code": "validation", "fields": { "password": "a senha precisa de ao menos 8 caracteres" } }
```

Detalhes internos nunca chegam ao cliente: a causa vai só para o log, e 5xx
sempre responde uma mensagem genérica.

## Endpoints

### Livres de autenticação

| Método | Rota | Descrição |
|---|---|---|
| GET | `/api/health` | O processo está no ar |
| GET | `/api/health/ready` | O banco responde |
| POST | `/api/auth/register` | Cria conta, devolve sessão |
| POST | `/api/auth/login` | Autentica, devolve sessão |

### Autenticadas (`Authorization: Bearer <token>`)

| Método | Rota | Descrição |
|---|---|---|
| GET | `/api/auth/me` | Usuário da sessão |
| GET | `/api/transactions?year=&month=` | Lançamentos do mês, com os fixos projetados |
| POST | `/api/transactions` | Cria um lançamento avulso |
| DELETE | `/api/transactions/{id}` | Apaga um lançamento |
| GET | `/api/transactions/summary?year=&month=` | Saldo atual e previsto do mês |
| GET | `/api/recurring` | Lista os lançamentos fixos |
| POST | `/api/recurring` | Cria um lançamento fixo |
| DELETE | `/api/recurring/{id}` | Apaga um fixo, e com ele as projeções |
| GET | `/api/categories` | Lista as categorias |
| POST | `/api/categories` | Cria uma categoria |
| DELETE | `/api/categories/{id}` | Apaga uma categoria |
| GET | `/api/dashboard?year=&month=` | Métricas do painel |

Sem `year` e `month`, as rotas assumem o mês corrente.

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"voce@exemplo.com","password":"senha12345"}'
```

## Lançamentos

Quatro tipos: `receita`, `despesa`, `cartao_credito` e `investimento`.
**Só receita soma saldo**; os outros três subtraem. O valor é sempre gravado
positivo e o sinal vem do tipo (`domain.Kind.Signed`), então nenhum lugar do
sistema precisa lembrar de inverter nada.

Sem descrição, o lançamento recebe o nome do próprio tipo — "Despesa",
"Gasto no cartão de crédito". A regra mora no serviço, não no controller.

**Saldo atual** conta só o que já aconteceu, até hoje. **Saldo previsto**
conta o mês fechado, incluindo o que ainda vai cair e as projeções dos fixos.

## Lançamentos fixos

Um fixo é uma regra — "Academia, todo dia 20, R$ 159,90" — com uma das seis
frequências: diário, semanal, quinzenal, mensal, semestral e anual.

**Fixos não geram linhas em `transactions`.** As ocorrências são projetadas
na leitura, por `domain.RecurringEntry.OccurrencesIn`. Isso evita o banco
divergir quando a regra muda, e faz apagar um fixo sumir com as projeções
dele em todos os meses de uma vez. Em troca, uma ocorrência isolada não pode
ser editada — o que é o próximo passo natural, com uma tabela de exceções.

O ciclo conta desde a data de início, não do começo do mês: um semanal que
começa em 01/01 cai nos dias 3, 10, 17 e 24 de setembro. Um mensal do dia 31
cai no último dia dos meses mais curtos, em vez de vazar para o mês seguinte.

## Painel

Dois números respondem "como estou hoje":

- **Saldo atual** acumula tudo que já foi pago e recebido, desde a primeira
  movimentação — sem recorte de mês ou ano. Os lançamentos gravados são
  somados no banco (`SumUntil`); os fixos são projetados do início de cada
  regra até hoje.
- **Despesas fixas** é o custo de vida do mês: o que os fixos de saída somam
  no período. Como a conta usa as ocorrências projetadas, um fixo semanal
  pesa quatro ou cinco vezes sem nenhuma conversão de frequência à mão.

## Decisões

- **Valores em centavos** (`BIGINT`). Ponto flutuante não é usado para dinheiro.
- **Login não revela se o e-mail existe**: senha errada e e-mail inexistente
  devolvem a mesma resposta.
- **E-mail duplicado é detectado pela constraint `UNIQUE`**, não por um
  `SELECT` antes do `INSERT` — dois cadastros simultâneos não passam os dois.
- **O `.env` não sobrescreve o ambiente**: em produção as variáveis reais vencem.
- **`JWT_SECRET` é obrigatório**; a aplicação não sobe sem ele, em nenhum ambiente.

## Testes

**Unitários** (`internal/service/`): exercitam os casos de uso com
repositório, hasher e emissor falsos — sem banco e sem servidor. É o retorno
prático de depender de interfaces.

**E2e** (`test/e2e/`, atrás da tag de build `e2e`): sobem o roteador real em
um `httptest.Server` e batem nas rotas com um cliente HTTP de verdade, contra
um Postgres de verdade. Verificam status, corpo e o efeito no banco — que a
senha foi hasheada, que o hash nunca aparece na resposta, que o e-mail é
normalizado, que senha errada e e-mail inexistente são indistinguíveis.

A suíte usa o banco `financas_test`, separado do de desenvolvimento. Ela o
cria na primeira execução e limpa as tabelas entre os testes; nenhum dado de
desenvolvimento é tocado. Para apontar para outro banco, defina
`TEST_DATABASE_URL`.

```bash
make db-up      # o Postgres precisa estar no ar
make test-all   # unitários + e2e
```

## Próximos passos

As categorias já têm CRUD, e `transactions` e `recurring_entries` já têm a
coluna `category_id` — **falta ligar uma coisa à outra**: escolher a
categoria ao lançar, exibi-la na lista e agrupar por ela no painel, que hoje
agrupa por tipo.

Cada funcionalidade segue o mesmo caminho: porta em `domain/ports.go`,
repositório em `repository/postgres/`, caso de uso em `service/`, DTO,
controller e rota em `api/`, com teste unitário e e2e.
