package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// RecurringRepository implementa domain.RecurringRepository.
type RecurringRepository struct {
	pool *pgxpool.Pool
}

var _ domain.RecurringRepository = (*RecurringRepository)(nil)

func NewRecurringRepository(pool *pgxpool.Pool) *RecurringRepository {
	return &RecurringRepository{pool: pool}
}

// recurringColumns traz a categoria junto, pelo mesmo motivo dos lançamentos.
const recurringColumns = `
	r.id, r.user_id, r.description, r.amount, r.kind, r.frequency,
	r.start_date, r.end_date, r.active, r.created_at,
	r.category_id, c.name, c.color`

const recurringFrom = `
	FROM recurring_entries r
	LEFT JOIN categories c ON c.id = r.category_id`

func (r *RecurringRepository) Create(ctx context.Context, input domain.NewRecurringEntry) (*domain.RecurringEntry, error) {
	const query = `
		INSERT INTO recurring_entries
			(user_id, description, amount, kind, frequency, start_date, end_date, category_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	var id int64
	err := r.pool.QueryRow(ctx, query,
		input.UserID,
		strings.TrimSpace(input.Description),
		input.AmountCents,
		string(input.Kind),
		string(input.Frequency),
		domain.Day(input.StartDate),
		normalizeEndDate(input.EndDate),
		input.CategoryID,
	).Scan(&id)
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	const byID = `SELECT ` + recurringColumns + recurringFrom + ` WHERE r.id = $1`

	var entry domain.RecurringEntry
	if err := scanRecurring(r.pool.QueryRow(ctx, byID, id), &entry); err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return &entry, nil
}

// Update reescreve o fixo. A cláusula com user_id impede editar o de outra
// pessoa.
func (r *RecurringRepository) Update(ctx context.Context, input domain.UpdateRecurringEntry) (*domain.RecurringEntry, error) {
	const query = `
		UPDATE recurring_entries SET
			description = $3, amount = $4, kind = $5, frequency = $6,
			start_date = $7, end_date = $8, category_id = $9, active = $10
		WHERE id = $1 AND user_id = $2`

	tag, err := r.pool.Exec(ctx, query,
		input.ID, input.UserID,
		strings.TrimSpace(input.Description), input.AmountCents,
		string(input.Kind), string(input.Frequency),
		domain.Day(input.StartDate), normalizeEndDate(input.EndDate),
		input.CategoryID, input.Active,
	)
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	if tag.RowsAffected() == 0 {
		return nil, domain.ErrRecurringNotFound
	}

	const byID = `SELECT ` + recurringColumns + recurringFrom + ` WHERE r.id = $1`

	var entry domain.RecurringEntry
	if err := scanRecurring(r.pool.QueryRow(ctx, byID, input.ID), &entry); err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return &entry, nil
}

func (r *RecurringRepository) List(ctx context.Context, userID int64) ([]domain.RecurringEntry, error) {
	return r.list(ctx, `SELECT `+recurringColumns+recurringFrom+`
		WHERE r.user_id = $1 ORDER BY r.description`, userID)
}

func (r *RecurringRepository) ListActive(ctx context.Context, userID int64) ([]domain.RecurringEntry, error) {
	return r.list(ctx, `SELECT `+recurringColumns+recurringFrom+`
		WHERE r.user_id = $1 AND r.active ORDER BY r.description`, userID)
}

func (r *RecurringRepository) FindByID(ctx context.Context, userID, id int64) (*domain.RecurringEntry, error) {
	const query = `SELECT ` + recurringColumns + recurringFrom + ` WHERE r.id = $1 AND r.user_id = $2`

	var entry domain.RecurringEntry
	err := scanRecurring(r.pool.QueryRow(ctx, query, id, userID), &entry)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrRecurringNotFound
	}
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return &entry, nil
}

func (r *RecurringRepository) Delete(ctx context.Context, userID, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM recurring_entries WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return domain.ErrInternal.Wrap(err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrRecurringNotFound
	}

	return nil
}

func (r *RecurringRepository) list(ctx context.Context, query string, userID int64) ([]domain.RecurringEntry, error) {
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}
	defer rows.Close()

	var entries []domain.RecurringEntry
	for rows.Next() {
		var entry domain.RecurringEntry
		if err := scanRecurring(rows, &entry); err != nil {
			return nil, domain.ErrInternal.Wrap(err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return entries, nil
}

// scanner cobre tanto pgx.Row quanto pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanRecurring(row scanner, entry *domain.RecurringEntry) error {
	return row.Scan(
		&entry.ID, &entry.UserID, &entry.Description, &entry.AmountCents,
		&entry.Kind, &entry.Frequency, &entry.StartDate, &entry.EndDate,
		&entry.Active, &entry.CreatedAt,
		&entry.CategoryID, &entry.CategoryName, &entry.CategoryColor,
	)
}

func normalizeEndDate(date *time.Time) *time.Time {
	if date == nil {
		return nil
	}

	normalized := domain.Day(*date)
	return &normalized
}
