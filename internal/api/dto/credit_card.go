package dto

import (
	"strings"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// CreditCardRequest é o corpo de POST e PUT /api/cards.
type CreditCardRequest struct {
	Name            string `json:"name"`
	LimitCents      int64  `json:"limit_cents"`
	BestPurchaseDay int    `json:"best_purchase_day"`
	DueDay          int    `json:"due_day"`
}

func (r CreditCardRequest) Validate() error {
	fields := map[string]string{}

	if strings.TrimSpace(r.Name) == "" {
		fields["name"] = "informe o nome do cartão"
	}
	if r.BestPurchaseDay < 1 || r.BestPurchaseDay > 31 {
		fields["best_purchase_day"] = "informe um dia entre 1 e 31"
	}
	if r.DueDay < 1 || r.DueDay > 31 {
		fields["due_day"] = "informe um dia entre 1 e 31"
	}

	if len(fields) > 0 {
		return domain.ErrValidation.WithFields(fields)
	}
	return nil
}

func (r CreditCardRequest) ToDomain(userID int64) domain.NewCreditCard {
	return domain.NewCreditCard{
		UserID:          userID,
		Name:            strings.TrimSpace(r.Name),
		LimitCents:      r.LimitCents,
		BestPurchaseDay: r.BestPurchaseDay,
		DueDay:          r.DueDay,
	}
}

// CreditCardResponse é um cartão como o cliente o vê.
type CreditCardResponse struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	LimitCents      int64  `json:"limit_cents"`
	BestPurchaseDay int    `json:"best_purchase_day"`
	DueDay          int    `json:"due_day"`
}

func NewCreditCardResponse(card domain.CreditCard) CreditCardResponse {
	return CreditCardResponse{
		ID:              card.ID,
		Name:            card.Name,
		LimitCents:      card.LimitCents,
		BestPurchaseDay: card.BestPurchaseDay,
		DueDay:          card.DueDay,
	}
}

func NewCreditCardListResponse(cards []domain.CreditCard) []CreditCardResponse {
	list := make([]CreditCardResponse, 0, len(cards))
	for _, card := range cards {
		list = append(list, NewCreditCardResponse(card))
	}
	return list
}

// AmortizeRequest é o corpo de POST /api/debts/{id}/amortize.
type AmortizeRequest struct {
	AmountCents       int64  `json:"amount_cents"`
	NewRemainingCents *int64 `json:"new_remaining_cents"`
	Mode              string `json:"mode"`
	Installments      *int   `json:"installments"`
}

func (r AmortizeRequest) Validate() error {
	if r.AmountCents <= 0 {
		return domain.ErrValidation.WithFields(map[string]string{
			"amount_cents": "informe um valor maior que zero",
		})
	}
	return nil
}

func (r AmortizeRequest) ToDomain() domain.Amortization {
	return domain.Amortization{
		AmountCents:       r.AmountCents,
		NewRemainingCents: r.NewRemainingCents,
		Mode:              domain.AmortizationMode(r.Mode),
		Installments:      r.Installments,
	}
}

// SettleRequest é o corpo de POST /api/debts/{id}/settle.
type SettleRequest struct {
	// SubtractFromBalance cria um lançamento pago com o que faltava, para o
	// saldo em carteira refletir a saída.
	SubtractFromBalance bool `json:"subtract_from_balance"`
}

func (r SettleRequest) Validate() error { return nil }
