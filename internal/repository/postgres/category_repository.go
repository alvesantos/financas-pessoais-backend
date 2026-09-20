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

// CategoryRepository implementa domain.CategoryRepository.
type CategoryRepository struct {
	pool *pgxpool.Pool
}

var _ domain.CategoryRepository = (*CategoryRepository)(nil)

func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

const categoryColumns = `id, user_id, name, kind, color, created_at`

func (r *CategoryRepository) Create(ctx context.Context, input domain.NewCategory) (*domain.Category, error) {
	const query = `
		INSERT INTO categories (user_id, name, kind, color)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + categoryColumns

	var category domain.Category
	err := r.pool.QueryRow(ctx, query,
		input.UserID, strings.TrimSpace(input.Name), string(input.Kind), input.Color,
	).Scan(&category.ID, &category.UserID, &category.Name, &category.Kind, &category.Color, &category.CreatedAt)

	// A restrição UNIQUE(user_id, name, kind) resolve a corrida entre dois
	// cadastros iguais.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return nil, domain.ErrCategoryTaken.Wrap(err)
	}
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return &category, nil
}

func (r *CategoryRepository) List(ctx context.Context, userID int64) ([]domain.Category, error) {
	const query = `SELECT ` + categoryColumns + `
		FROM categories WHERE user_id = $1 ORDER BY kind, name`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		var category domain.Category
		if err := rows.Scan(
			&category.ID, &category.UserID, &category.Name,
			&category.Kind, &category.Color, &category.CreatedAt,
		); err != nil {
			return nil, domain.ErrInternal.Wrap(err)
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return categories, nil
}

// FindByID busca no escopo do usuário: categoria de outra pessoa não existe.
func (r *CategoryRepository) FindByID(ctx context.Context, userID, id int64) (*domain.Category, error) {
	const query = `SELECT ` + categoryColumns + `
		FROM categories WHERE id = $1 AND user_id = $2`

	var category domain.Category
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&category.ID, &category.UserID, &category.Name,
		&category.Kind, &category.Color, &category.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCategoryNotFound
	}
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return &category, nil
}

func (r *CategoryRepository) Delete(ctx context.Context, userID, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM categories WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return domain.ErrInternal.Wrap(err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrCategoryNotFound
	}

	return nil
}
