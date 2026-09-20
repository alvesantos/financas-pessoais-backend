package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// DebtRepository implementa domain.DebtRepository.
type DebtRepository struct {
	pool *pgxpool.Pool
}

var _ domain.DebtRepository = (*DebtRepository)(nil)

func NewDebtRepository(pool *pgxpool.Pool) *DebtRepository {
	return &DebtRepository{pool: pool}
}

// debtColumns traz a categoria junto, para a listagem não precisar de uma
// consulta por linha.
const debtColumns = `
	d.id, d.user_id, d.description, d.installment_amount, d.installments,
	d.kind, d.frequency, d.first_due_date, d.created_at,
	d.amortized_cents, d.settled_at,
	d.category_id, c.name, c.color`

const debtFrom = `
	FROM debts d
	LEFT JOIN categories c ON c.id = d.category_id`

func (r *DebtRepository) Create(ctx context.Context, input domain.NewDebt) (*domain.Debt, error) {
	const query = `
		INSERT INTO debts
			(user_id, description, installment_amount, installments, kind, frequency, first_due_date, category_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	var id int64
	err := r.pool.QueryRow(ctx, query,
		input.UserID,
		strings.TrimSpace(input.Description),
		input.InstallmentCents,
		input.Installments,
		string(input.Kind),
		string(input.Frequency),
		domain.Day(input.FirstDueDate),
		input.CategoryID,
	).Scan(&id)
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	const byID = `SELECT ` + debtColumns + debtFrom + ` WHERE d.id = $1`

	var debt domain.Debt
	if err := scanDebt(r.pool.QueryRow(ctx, byID, id), &debt); err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return &debt, nil
}

// Save grava a dívida inteira: é o que a edição, a amortização e a quitação
// usam, já que todas devolvem uma dívida recalculada pelo domínio.
func (r *DebtRepository) Save(ctx context.Context, debt domain.Debt) (*domain.Debt, error) {
	const query = `
		UPDATE debts SET
			description = $3, installment_amount = $4, installments = $5,
			kind = $6, frequency = $7, first_due_date = $8, category_id = $9,
			amortized_cents = $10, settled_at = $11
		WHERE id = $1 AND user_id = $2`

	tag, err := r.pool.Exec(ctx, query,
		debt.ID, debt.UserID,
		strings.TrimSpace(debt.Description), debt.InstallmentCents, debt.Installments,
		string(debt.Kind), string(debt.Frequency), domain.Day(debt.FirstDueDate),
		debt.CategoryID, debt.AmortizedCents, debt.SettledAt,
	)
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	if tag.RowsAffected() == 0 {
		return nil, domain.ErrDebtNotFound
	}

	return r.FindByID(ctx, debt.UserID, debt.ID)
}

func (r *DebtRepository) FindByID(ctx context.Context, userID, id int64) (*domain.Debt, error) {
	const query = `SELECT ` + debtColumns + debtFrom + ` WHERE d.id = $1 AND d.user_id = $2`

	var debt domain.Debt
	err := scanDebt(r.pool.QueryRow(ctx, query, id, userID), &debt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrDebtNotFound
	}
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return &debt, nil
}

func (r *DebtRepository) List(ctx context.Context, userID int64) ([]domain.Debt, error) {
	const query = `SELECT ` + debtColumns + debtFrom + `
		WHERE d.user_id = $1 ORDER BY d.first_due_date, d.id`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}
	defer rows.Close()

	var debts []domain.Debt
	for rows.Next() {
		var debt domain.Debt
		if err := scanDebt(rows, &debt); err != nil {
			return nil, domain.ErrInternal.Wrap(err)
		}
		debts = append(debts, debt)
	}

	if err := rows.Err(); err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return debts, nil
}

func (r *DebtRepository) Delete(ctx context.Context, userID, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM debts WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return domain.ErrInternal.Wrap(err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrDebtNotFound
	}

	return nil
}

func scanDebt(row scanner, debt *domain.Debt) error {
	return row.Scan(
		&debt.ID, &debt.UserID, &debt.Description, &debt.InstallmentCents,
		&debt.Installments, &debt.Kind, &debt.Frequency, &debt.FirstDueDate,
		&debt.CreatedAt, &debt.AmortizedCents, &debt.SettledAt,
		&debt.CategoryID, &debt.CategoryName, &debt.CategoryColor,
	)
}
