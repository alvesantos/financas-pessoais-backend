package service

import (
	"context"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// maxInstallments é o teto que o banco também impõe.
const maxInstallments = 600

// DebtService reúne os casos de uso de dívidas parceladas.
type DebtService struct {
	repository domain.DebtRepository
	categories domain.CategoryRepository
}

var _ domain.DebtService = (*DebtService)(nil)

func NewDebtService(
	repository domain.DebtRepository,
	categories domain.CategoryRepository,
) *DebtService {
	return &DebtService{repository: repository, categories: categories}
}

func (s *DebtService) Create(ctx context.Context, input domain.NewDebt) (*domain.Debt, error) {
	fields := map[string]string{}

	if input.InstallmentCents <= 0 {
		fields["installment_amount_cents"] = "informe o valor da parcela"
	}
	if input.Installments <= 0 {
		fields["installments"] = "informe quantas parcelas são"
	}
	if input.Installments > maxInstallments {
		fields["installments"] = "o limite é de 600 parcelas"
	}
	// Dívida é saída: receita e investimento não cabem aqui.
	if input.Kind != domain.KindDespesa && input.Kind != domain.KindCartaoCredito {
		fields["kind"] = "a dívida é uma despesa ou um gasto no cartão"
	}
	if !input.Frequency.Valid() {
		fields["frequency"] = "escolha uma frequência"
	}

	if len(fields) > 0 {
		return nil, domain.ErrValidation.WithFields(fields)
	}

	if err := ensureCategory(ctx, s.categories, input.UserID, input.CategoryID, input.Kind); err != nil {
		return nil, err
	}

	input.Description = defaultDescription(input.Description, input.Kind)
	input.FirstDueDate = domain.Day(input.FirstDueDate)

	return s.repository.Create(ctx, input)
}

func (s *DebtService) List(ctx context.Context, userID int64) ([]domain.Debt, error) {
	return s.repository.List(ctx, userID)
}

func (s *DebtService) Delete(ctx context.Context, userID, id int64) error {
	return s.repository.Delete(ctx, userID, id)
}
