package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// TransactionRepository implementa domain.TransactionRepository.
type TransactionRepository struct {
	pool *pgxpool.Pool
}

var _ domain.TransactionRepository = (*TransactionRepository)(nil)

func NewTransactionRepository(pool *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{pool: pool}
}

func (r *TransactionRepository) Create(ctx context.Context, input domain.NewTransaction) (*domain.Transaction, error) {
	const query = `
		INSERT INTO transactions (user_id, description, amount, kind, occurred_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, description, amount, kind, occurred_at, created_at`

	var t domain.Transaction
	err := r.pool.QueryRow(ctx, query,
		input.UserID,
		strings.TrimSpace(input.Description),
		input.AmountCents,
		string(input.Kind),
		domain.Day(input.OccurredAt),
	).Scan(&t.ID, &t.UserID, &t.Description, &t.AmountCents, &t.Kind, &t.OccurredAt, &t.CreatedAt)
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return &t, nil
}

func (r *TransactionRepository) ListByPeriod(ctx context.Context, userID int64, period domain.Period) ([]domain.Transaction, error) {
	const query = `
		SELECT id, user_id, description, amount, kind, occurred_at, created_at
		FROM transactions
		WHERE user_id = $1 AND occurred_at BETWEEN $2 AND $3
		ORDER BY occurred_at, id`

	rows, err := r.pool.Query(ctx, query, userID, period.From, period.To)
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}
	defer rows.Close()

	var transactions []domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Description, &t.AmountCents, &t.Kind, &t.OccurredAt, &t.CreatedAt,
		); err != nil {
			return nil, domain.ErrInternal.Wrap(err)
		}
		transactions = append(transactions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return transactions, nil
}

func (r *TransactionRepository) Delete(ctx context.Context, userID, id int64) error {
	// O user_id na cláusula impede apagar o lançamento de outra pessoa.
	tag, err := r.pool.Exec(ctx, `DELETE FROM transactions WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return domain.ErrInternal.Wrap(err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrTransactionNotFound
	}

	return nil
}
