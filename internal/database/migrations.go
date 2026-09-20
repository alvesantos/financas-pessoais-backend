package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id            BIGSERIAL PRIMARY KEY,
	name          TEXT        NOT NULL,
	email         TEXT        NOT NULL UNIQUE,
	password_hash TEXT        NOT NULL,
	created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS accounts (
	id              BIGSERIAL PRIMARY KEY,
	user_id         BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name            TEXT        NOT NULL,
	type            TEXT        NOT NULL CHECK (type IN ('checking','savings','credit_card','cash','investment')),
	initial_balance BIGINT      NOT NULL DEFAULT 0,
	created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_accounts_user ON accounts(user_id);

CREATE TABLE IF NOT EXISTS categories (
	id      BIGSERIAL PRIMARY KEY,
	user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name    TEXT   NOT NULL,
	kind    TEXT   NOT NULL CHECK (kind IN ('income','expense')),
	color   TEXT   NOT NULL DEFAULT '#6366f1',
	UNIQUE (user_id, name, kind)
);

CREATE TABLE IF NOT EXISTS transactions (
	id          BIGSERIAL PRIMARY KEY,
	user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	account_id  BIGINT      NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
	category_id BIGINT      REFERENCES categories(id) ON DELETE SET NULL,
	description TEXT        NOT NULL,
	-- valor em centavos: evita erro de ponto flutuante em dinheiro
	amount      BIGINT      NOT NULL,
	kind        TEXT        NOT NULL CHECK (kind IN ('income','expense')),
	occurred_at DATE        NOT NULL,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_date ON transactions(user_id, occurred_at DESC);
`

// Migrate cria o schema caso ainda não exista.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, schema)
	return err
}
