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

// TransactionRepository implementa domain.TransactionRepository.
type TransactionRepository struct {
	pool *pgxpool.Pool
}

var _ domain.TransactionRepository = (*TransactionRepository)(nil)

func NewTransactionRepository(pool *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{pool: pool}
}

func (r *TransactionRepository) Create(ctx context.Context, input domain.NewTransaction) (*domain.Transaction, error) {
	// O RETURNING não alcança a categoria, que está em outra tabela: o
	// INSERT grava o id e o SELECT seguinte traz nome e cor.
	const query = `
		INSERT INTO transactions (user_id, description, amount, kind, occurred_at, category_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	var id int64
	err := r.pool.QueryRow(ctx, query,
		input.UserID,
		strings.TrimSpace(input.Description),
		input.AmountCents,
		string(input.Kind),
		domain.Day(input.OccurredAt),
		input.CategoryID,
	).Scan(&id)
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return r.findByID(ctx, input.UserID, id)
}

// transactionColumns traz a categoria junto, para a listagem não precisar de
// uma consulta por linha.
const transactionColumns = `
	t.id, t.user_id, t.description, t.amount, t.kind, t.occurred_at, t.created_at,
	t.category_id, c.name, c.color`

func (r *TransactionRepository) findByID(ctx context.Context, userID, id int64) (*domain.Transaction, error) {
	const query = `
		SELECT ` + transactionColumns + `
		FROM transactions t
		LEFT JOIN categories c ON c.id = t.category_id
		WHERE t.id = $1 AND t.user_id = $2`

	var t domain.Transaction
	err := scanTransaction(r.pool.QueryRow(ctx, query, id, userID), &t)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTransactionNotFound
	}
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return &t, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTransaction(row rowScanner, t *domain.Transaction) error {
	return row.Scan(
		&t.ID, &t.UserID, &t.Description, &t.AmountCents, &t.Kind,
		&t.OccurredAt, &t.CreatedAt,
		&t.CategoryID, &t.CategoryName, &t.CategoryColor,
	)
}

func (r *TransactionRepository) ListByPeriod(ctx context.Context, userID int64, period domain.Period) ([]domain.Transaction, error) {
	const query = `
		SELECT ` + transactionColumns + `
		FROM transactions t
		LEFT JOIN categories c ON c.id = t.category_id
		WHERE t.user_id = $1 AND t.occurred_at BETWEEN $2 AND $3
		ORDER BY t.occurred_at, t.id`

	rows, err := r.pool.Query(ctx, query, userID, period.From, period.To)
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}
	defer rows.Close()

	var transactions []domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		if err := scanTransaction(rows, &t); err != nil {
			return nil, domain.ErrInternal.Wrap(err)
		}
		transactions = append(transactions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return transactions, nil
}

// SumUntil deixa a soma no banco: trazer o histórico inteiro para o Go só
// para somá-lo cresceria com o tempo de uso.
func (r *TransactionRepository) SumUntil(ctx context.Context, userID int64, until time.Time) (int64, error) {
	const query = `
		SELECT COALESCE(SUM(CASE WHEN kind = 'receita' THEN amount ELSE -amount END), 0)
		FROM transactions
		WHERE user_id = $1 AND occurred_at <= $2`

	var total int64
	if err := r.pool.QueryRow(ctx, query, userID, domain.Day(until)).Scan(&total); err != nil {
		return 0, domain.ErrInternal.Wrap(err)
	}

	return total, nil
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
