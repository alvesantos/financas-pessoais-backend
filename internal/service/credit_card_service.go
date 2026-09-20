package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/alvesantos/financas-backend/internal/domain"
)

const maxCardNameLength = 40

// CreditCardService reúne os casos de uso de cartões de crédito.
type CreditCardService struct {
	repository domain.CreditCardRepository
}

var _ domain.CreditCardService = (*CreditCardService)(nil)

func NewCreditCardService(repository domain.CreditCardRepository) *CreditCardService {
	return &CreditCardService{repository: repository}
}

func (s *CreditCardService) Create(ctx context.Context, input domain.NewCreditCard) (*domain.CreditCard, error) {
	if err := validateCard(input.Name, input.LimitCents, input.BestPurchaseDay, input.DueDay); err != nil {
		return nil, err
	}

	input.Name = strings.TrimSpace(input.Name)

	return s.repository.Create(ctx, input)
}

func (s *CreditCardService) Update(ctx context.Context, card domain.CreditCard) (*domain.CreditCard, error) {
	if err := validateCard(card.Name, card.LimitCents, card.BestPurchaseDay, card.DueDay); err != nil {
		return nil, err
	}

	card.Name = strings.TrimSpace(card.Name)

	return s.repository.Update(ctx, card)
}

func (s *CreditCardService) List(ctx context.Context, userID int64) ([]domain.CreditCard, error) {
	return s.repository.List(ctx, userID)
}

func (s *CreditCardService) Delete(ctx context.Context, userID, id int64) error {
	return s.repository.Delete(ctx, userID, id)
}

func validateCard(name string, limitCents int64, bestPurchaseDay, dueDay int) error {
	fields := map[string]string{}

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		fields["name"] = "informe o nome do cartão"
	}
	if utf8.RuneCountInString(trimmed) > maxCardNameLength {
		fields["name"] = "o nome passou de 40 caracteres"
	}
	if limitCents < 0 {
		fields["limit_cents"] = "o limite não pode ser negativo"
	}
	if bestPurchaseDay < 1 || bestPurchaseDay > 31 {
		fields["best_purchase_day"] = "informe um dia entre 1 e 31"
	}
	if dueDay < 1 || dueDay > 31 {
		fields["due_day"] = "informe um dia entre 1 e 31"
	}

	if len(fields) > 0 {
		return domain.ErrValidation.WithFields(fields)
	}

	return nil
}
