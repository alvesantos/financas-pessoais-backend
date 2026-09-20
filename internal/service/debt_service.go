package service

import (
	"context"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// maxInstallments é o teto que o banco também impõe.
const maxInstallments = 600

// DebtService reúne os casos de uso de dívidas parceladas.
type DebtService struct {
	repository   domain.DebtRepository
	categories   domain.CategoryRepository
	transactions domain.TransactionRepository
	clock        domain.Clock
}

var _ domain.DebtService = (*DebtService)(nil)

func NewDebtService(
	repository domain.DebtRepository,
	categories domain.CategoryRepository,
	transactions domain.TransactionRepository,
	clock domain.Clock,
) *DebtService {
	return &DebtService{
		repository:   repository,
		categories:   categories,
		transactions: transactions,
		clock:        clock,
	}
}

func (s *DebtService) Create(ctx context.Context, input domain.NewDebt) (*domain.Debt, error) {
	if err := s.validate(ctx, input); err != nil {
		return nil, err
	}

	input.Description = defaultDescription(input.Description, input.Kind)
	input.FirstDueDate = domain.Day(input.FirstDueDate)

	return s.repository.Create(ctx, input)
}

func (s *DebtService) validate(ctx context.Context, input domain.NewDebt) error {
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
		return domain.ErrValidation.WithFields(fields)
	}

	return ensureCategory(ctx, s.categories, input.UserID, input.CategoryID, input.Kind)
}

// Update reescreve a dívida, com as mesmas regras da criação. O que já foi
// amortizado e a quitação não se mexem por aqui.
func (s *DebtService) Update(ctx context.Context, input domain.UpdateDebt) (*domain.Debt, error) {
	if err := s.validate(ctx, input.NewDebt); err != nil {
		return nil, err
	}

	current, err := s.repository.FindByID(ctx, input.UserID, input.ID)
	if err != nil {
		return nil, err
	}

	updated := *current
	updated.Description = defaultDescription(input.Description, input.Kind)
	updated.InstallmentCents = input.InstallmentCents
	updated.Installments = input.Installments
	updated.Kind = input.Kind
	updated.Frequency = input.Frequency
	updated.FirstDueDate = domain.Day(input.FirstDueDate)
	updated.CategoryID = input.CategoryID

	return s.repository.Save(ctx, updated)
}

// Amortize abate o saldo devedor e refaz o parcelamento conforme escolhido.
func (s *DebtService) Amortize(
	ctx context.Context, userID, id int64, input domain.Amortization,
) (*domain.Debt, error) {
	if input.Mode == "" {
		input.Mode = domain.AmortizeKeepInstallment
	}
	if !input.Mode.Valid() {
		return nil, domain.ErrValidation.WithFields(map[string]string{
			"mode": "escolha o que fazer com as parcelas",
		})
	}

	debt, err := s.repository.FindByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	amortized, err := debt.Amortize(input, s.clock.Today())
	if err != nil {
		return nil, err
	}

	return s.repository.Save(ctx, amortized)
}

// Settle quita a dívida. Com subtractFromBalance, o que faltava vira um
// lançamento pago de verdade, para o saldo em carteira refletir a saída.
func (s *DebtService) Settle(
	ctx context.Context, userID, id int64, subtractFromBalance bool,
) (*domain.Debt, error) {
	debt, err := s.repository.FindByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	today := s.clock.Today()
	remaining := debt.Progress(today).RemainingCents

	settled, err := debt.Settle(today)
	if err != nil {
		return nil, err
	}

	saved, err := s.repository.Save(ctx, settled)
	if err != nil {
		return nil, err
	}

	if subtractFromBalance && remaining > 0 {
		_, err := s.transactions.Create(ctx, domain.NewTransaction{
			UserID:      userID,
			Description: "Quitação: " + debt.Description,
			AmountCents: remaining,
			Kind:        debt.Kind,
			OccurredAt:  today,
			CategoryID:  debt.CategoryID,
			Paid:        true,
		})
		if err != nil {
			return nil, err
		}
	}

	return saved, nil
}

func (s *DebtService) List(ctx context.Context, userID int64) ([]domain.Debt, error) {
	return s.repository.List(ctx, userID)
}

func (s *DebtService) Delete(ctx context.Context, userID, id int64) error {
	return s.repository.Delete(ctx, userID, id)
}
