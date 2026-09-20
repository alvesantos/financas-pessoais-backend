package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

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
	invoiceMonth := invoiceMonthOf(input)

	// O RETURNING não alcança a categoria nem o cartão, que estão em outras
	// tabelas: o INSERT grava os ids e o SELECT seguinte traz nome e cor.
	const query = `
		INSERT INTO transactions
			(user_id, description, amount, kind, occurred_at, category_id, paid,
			 credit_card_id, invoice_month, recurring_id, debt_id, installment_number)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`

	var id int64
	err := r.pool.QueryRow(ctx, query,
		input.UserID,
		strings.TrimSpace(input.Description),
		input.AmountCents,
		string(input.Kind),
		domain.Day(input.OccurredAt),
		input.CategoryID,
		input.Paid,
		input.CreditCardID,
		invoiceMonth,
		input.RecurringID,
		input.DebtID,
		input.InstallmentNumber,
	).Scan(&id)

	// O índice único impede a mesma ocorrência virar linha duas vezes.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return nil, domain.ErrOccurrenceAlreadyPaid.Wrap(err)
	}
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return r.findByID(ctx, input.UserID, id)
}

// transactionColumns traz a categoria junto, para a listagem não precisar de
// uma consulta por linha.
const transactionColumns = `
	t.id, t.user_id, t.description, t.amount, t.kind, t.occurred_at, t.created_at,
	t.paid, t.credit_card_id, cc.name, t.invoice_month,
	t.recurring_id, t.debt_id, t.installment_number,
	t.category_id, c.name, c.color`

func (r *TransactionRepository) findByID(ctx context.Context, userID, id int64) (*domain.Transaction, error) {
	const query = `
		SELECT ` + transactionColumns + `
		FROM transactions t
		LEFT JOIN categories c ON c.id = t.category_id
		LEFT JOIN credit_cards cc ON cc.id = t.credit_card_id
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
		&t.Paid, &t.CreditCardID, &t.CreditCardName, &t.InvoiceMonth,
		&t.RecurringID, &t.DebtID, &t.InstallmentNumber,
		&t.CategoryID, &t.CategoryName, &t.CategoryColor,
	)
}

func (r *TransactionRepository) ListByPeriod(ctx context.Context, userID int64, period domain.Period) ([]domain.Transaction, error) {
	const query = `
		SELECT ` + transactionColumns + `
		FROM transactions t
		LEFT JOIN categories c ON c.id = t.category_id
		LEFT JOIN credit_cards cc ON cc.id = t.credit_card_id
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

// Update reescreve o lançamento. A cláusula com user_id impede editar o
// lançamento de outra pessoa.
func (r *TransactionRepository) Update(ctx context.Context, input domain.UpdateTransaction) (*domain.Transaction, error) {
	invoiceMonth := invoiceMonthOf(input.NewTransaction)

	const query = `
		UPDATE transactions SET
			description = $3, amount = $4, kind = $5, occurred_at = $6,
			category_id = $7, paid = $8, credit_card_id = $9, invoice_month = $10
		WHERE id = $1 AND user_id = $2`

	tag, err := r.pool.Exec(ctx, query,
		input.ID,
		input.UserID,
		strings.TrimSpace(input.Description),
		input.AmountCents,
		string(input.Kind),
		domain.Day(input.OccurredAt),
		input.CategoryID,
		input.Paid,
		input.CreditCardID,
		invoiceMonth,
	)
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	if tag.RowsAffected() == 0 {
		return nil, domain.ErrTransactionNotFound
	}

	return r.findByID(ctx, input.UserID, input.ID)
}

// invoiceMonthOf só faz sentido quando o lançamento é de cartão.
func invoiceMonthOf(input domain.NewTransaction) *time.Time {
	return input.InvoiceMonth
}

// SumPaid deixa a soma no banco: trazer o histórico inteiro para o Go só
// para somá-lo cresceria com o tempo de uso.
func (r *TransactionRepository) SumPaid(ctx context.Context, userID int64) (int64, error) {
	const query = `
		SELECT COALESCE(SUM(CASE WHEN kind = 'receita' THEN amount ELSE -amount END), 0)
		FROM transactions
		WHERE user_id = $1 AND paid`

	var total int64
	if err := r.pool.QueryRow(ctx, query, userID).Scan(&total); err != nil {
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
