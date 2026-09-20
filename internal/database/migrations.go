package database

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// migration é um passo versionado do schema. Uma vez aplicada e commitada,
// uma migração nunca é editada: corrige-se com uma nova.
type migration struct {
	version int
	name    string
	sql     string
}

var migrations = []migration{
	{1, "schema inicial", schema001},
	{2, "lancamentos e fixos", schema002},
}

const schema001 = `
CREATE TABLE IF NOT EXISTS users (
	id            BIGSERIAL PRIMARY KEY,
	name          TEXT        NOT NULL,
	email         TEXT        NOT NULL UNIQUE,
	password_hash TEXT        NOT NULL,
	created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS categories (
	id      BIGSERIAL PRIMARY KEY,
	user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name    TEXT   NOT NULL,
	kind    TEXT   NOT NULL,
	color   TEXT   NOT NULL DEFAULT '#6366f1',
	UNIQUE (user_id, name, kind)
);
`

// schema002 substitui a tabela de lançamentos do rascunho inicial, que nunca
// chegou a receber dados: os tipos passam a ser quatro e a conta deixa de ser
// obrigatória, já que contas ainda não existem na aplicação.
const schema002 = `
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS accounts;

CREATE TABLE transactions (
	id          BIGSERIAL PRIMARY KEY,
	user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	category_id BIGINT      REFERENCES categories(id) ON DELETE SET NULL,
	description TEXT        NOT NULL,
	-- valor em centavos, sempre positivo: o sinal vem do tipo
	amount      BIGINT      NOT NULL CHECK (amount > 0),
	kind        TEXT        NOT NULL CHECK (kind IN ('receita','despesa','cartao_credito','investimento')),
	occurred_at DATE        NOT NULL,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_transactions_user_date ON transactions(user_id, occurred_at);

-- Lançamentos fixos. Não geram linhas em transactions: as ocorrências são
-- projetadas na leitura, o que evita o schema divergir quando a regra muda.
CREATE TABLE recurring_entries (
	id          BIGSERIAL PRIMARY KEY,
	user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	category_id BIGINT      REFERENCES categories(id) ON DELETE SET NULL,
	description TEXT        NOT NULL,
	amount      BIGINT      NOT NULL CHECK (amount > 0),
	kind        TEXT        NOT NULL CHECK (kind IN ('receita','despesa','cartao_credito','investimento')),
	frequency   TEXT        NOT NULL CHECK (frequency IN ('diario','semanal','quinzenal','mensal','semestral','anual')),
	start_date  DATE        NOT NULL,
	end_date    DATE,
	active      BOOLEAN     NOT NULL DEFAULT true,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
	CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX idx_recurring_user ON recurring_entries(user_id) WHERE active;
`

// Migrate aplica, em ordem, as migrações que ainda faltam.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if err := ensureMigrationsTable(ctx, pool); err != nil {
		return err
	}

	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.version] {
			continue
		}

		if err := apply(ctx, pool, m); err != nil {
			return fmt.Errorf("migração %d (%s): %w", m.version, m.name, err)
		}

		slog.Info("migração aplicada", "versao", m.version, "nome", m.name)
	}

	return nil
}

func ensureMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER     PRIMARY KEY,
			name       TEXT        NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	if err != nil {
		return fmt.Errorf("criar schema_migrations: %w", err)
	}

	return baselineExistingSchema(ctx, pool)
}

// baselineExistingSchema marca a versão 1 como aplicada em bancos criados
// antes do controle de versão existir, para não recriar o que já está lá.
func baselineExistingSchema(ctx context.Context, pool *pgxpool.Pool) error {
	var registered bool
	err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations)`).Scan(&registered)
	if err != nil {
		return fmt.Errorf("ler schema_migrations: %w", err)
	}
	if registered {
		return nil
	}

	var usersExists bool
	err = pool.QueryRow(ctx, `SELECT to_regclass('public.users') IS NOT NULL`).Scan(&usersExists)
	if err != nil {
		return fmt.Errorf("verificar schema existente: %w", err)
	}
	if !usersExists {
		return nil
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO schema_migrations (version, name) VALUES (1, 'schema inicial (baseline)')`)
	if err != nil {
		return fmt.Errorf("registrar baseline: %w", err)
	}

	return nil
}

func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[int]bool, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("listar migrações aplicadas: %w", err)
	}
	defer rows.Close()

	applied := map[int]bool{}
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

// apply roda a migração e o registro dela na mesma transação: ou as duas
// coisas acontecem, ou nenhuma.
func apply(ctx context.Context, pool *pgxpool.Pool, m migration) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, m.sql); err != nil {
			return err
		}

		_, err := tx.Exec(ctx,
			`INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, m.version, m.name)
		return err
	})
}
