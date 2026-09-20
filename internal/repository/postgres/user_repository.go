package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// uniqueViolation é o SQLSTATE do Postgres para violação de índice único.
const uniqueViolation = "23505"

// UserRepository implementa domain.UserRepository sobre o Postgres.
type UserRepository struct {
	pool *pgxpool.Pool
}

// Garantia em tempo de compilação de que a porta está satisfeita.
var _ domain.UserRepository = (*UserRepository)(nil)

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, input domain.NewUser) (*domain.User, error) {
	const query = `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, password_hash, created_at, updated_at`

	user, err := r.queryOne(ctx, query,
		strings.TrimSpace(input.Name),
		normalizeEmail(input.Email),
		input.PasswordHash,
	)

	// A restrição UNIQUE decide quem ganha a corrida entre dois cadastros iguais.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return nil, domain.ErrEmailTaken.Wrap(err)
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	const query = `
		SELECT id, name, email, password_hash, created_at, updated_at
		FROM users WHERE email = $1`

	return r.queryOne(ctx, query, normalizeEmail(email))
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	const query = `
		SELECT id, name, email, password_hash, created_at, updated_at
		FROM users WHERE id = $1`

	return r.queryOne(ctx, query, id)
}

func (r *UserRepository) queryOne(ctx context.Context, query string, args ...any) (*domain.User, error) {
	var u domain.User

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return &u, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
