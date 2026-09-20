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

// CreditCardRepository implementa domain.CreditCardRepository.
type CreditCardRepository struct {
	pool *pgxpool.Pool
}

var _ domain.CreditCardRepository = (*CreditCardRepository)(nil)

func NewCreditCardRepository(pool *pgxpool.Pool) *CreditCardRepository {
	return &CreditCardRepository{pool: pool}
}

const creditCardColumns = `id, user_id, name, limit_cents, best_purchase_day, due_day, created_at`

func (r *CreditCardRepository) Create(ctx context.Context, input domain.NewCreditCard) (*domain.CreditCard, error) {
	const query = `
		INSERT INTO credit_cards (user_id, name, limit_cents, best_purchase_day, due_day)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + creditCardColumns

	var card domain.CreditCard
	err := scanCreditCard(r.pool.QueryRow(ctx, query,
		input.UserID, strings.TrimSpace(input.Name), input.LimitCents,
		input.BestPurchaseDay, input.DueDay,
	), &card)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return nil, domain.ErrCreditCardTaken.Wrap(err)
	}
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return &card, nil
}

func (r *CreditCardRepository) Update(ctx context.Context, card domain.CreditCard) (*domain.CreditCard, error) {
	const query = `
		UPDATE credit_cards
		SET name = $3, limit_cents = $4, best_purchase_day = $5, due_day = $6
		WHERE id = $1 AND user_id = $2
		RETURNING ` + creditCardColumns

	var updated domain.CreditCard
	err := scanCreditCard(r.pool.QueryRow(ctx, query,
		card.ID, card.UserID, strings.TrimSpace(card.Name), card.LimitCents,
		card.BestPurchaseDay, card.DueDay,
	), &updated)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCreditCardNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return nil, domain.ErrCreditCardTaken.Wrap(err)
	}
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return &updated, nil
}

func (r *CreditCardRepository) List(ctx context.Context, userID int64) ([]domain.CreditCard, error) {
	const query = `SELECT ` + creditCardColumns + ` FROM credit_cards WHERE user_id = $1 ORDER BY name`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}
	defer rows.Close()

	var cards []domain.CreditCard
	for rows.Next() {
		var card domain.CreditCard
		if err := scanCreditCard(rows, &card); err != nil {
			return nil, domain.ErrInternal.Wrap(err)
		}
		cards = append(cards, card)
	}

	if err := rows.Err(); err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return cards, nil
}

func (r *CreditCardRepository) FindByID(ctx context.Context, userID, id int64) (*domain.CreditCard, error) {
	const query = `SELECT ` + creditCardColumns + ` FROM credit_cards WHERE id = $1 AND user_id = $2`

	var card domain.CreditCard
	err := scanCreditCard(r.pool.QueryRow(ctx, query, id, userID), &card)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCreditCardNotFound
	}
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return &card, nil
}

func (r *CreditCardRepository) Delete(ctx context.Context, userID, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM credit_cards WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return domain.ErrInternal.Wrap(err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrCreditCardNotFound
	}

	return nil
}

func scanCreditCard(row scanner, card *domain.CreditCard) error {
	return row.Scan(
		&card.ID, &card.UserID, &card.Name, &card.LimitCents,
		&card.BestPurchaseDay, &card.DueDay, &card.CreatedAt,
	)
}
