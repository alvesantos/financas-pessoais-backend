package service

import (
	"context"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// RecurringService reúne os casos de uso de lançamentos fixos.
type RecurringService struct {
	repository domain.RecurringRepository
	categories domain.CategoryRepository
}

var _ domain.RecurringService = (*RecurringService)(nil)

func NewRecurringService(
	repository domain.RecurringRepository,
	categories domain.CategoryRepository,
) *RecurringService {
	return &RecurringService{repository: repository, categories: categories}
}

// Create grava um fixo. Vale a mesma regra de descrição dos lançamentos.
func (s *RecurringService) Create(ctx context.Context, input domain.NewRecurringEntry) (*domain.RecurringEntry, error) {
	if err := s.validate(ctx, input); err != nil {
		return nil, err
	}

	input.Description = defaultDescription(input.Description, input.Kind)
	input.StartDate = domain.Day(input.StartDate)

	return s.repository.Create(ctx, input)
}

// Update reescreve o fixo, com as mesmas regras da criação.
func (s *RecurringService) Update(
	ctx context.Context, input domain.UpdateRecurringEntry,
) (*domain.RecurringEntry, error) {
	if err := s.validate(ctx, input.NewRecurringEntry); err != nil {
		return nil, err
	}

	input.Description = defaultDescription(input.Description, input.Kind)
	input.StartDate = domain.Day(input.StartDate)

	return s.repository.Update(ctx, input)
}

func (s *RecurringService) List(ctx context.Context, userID int64) ([]domain.RecurringEntry, error) {
	return s.repository.List(ctx, userID)
}

func (s *RecurringService) Delete(ctx context.Context, userID, id int64) error {
	return s.repository.Delete(ctx, userID, id)
}

func (s *RecurringService) validate(ctx context.Context, input domain.NewRecurringEntry) error {
	if err := validateAmountAndKind(input.AmountCents, input.Kind); err != nil {
		return err
	}

	if !input.Frequency.Valid() {
		return domain.ErrValidation.WithFields(map[string]string{
			"frequency": "frequência inválida",
		})
	}

	if input.EndDate != nil && input.EndDate.Before(input.StartDate) {
		return domain.ErrValidation.WithFields(map[string]string{
			"end_date": "o fim não pode ser antes do início",
		})
	}

	return ensureCategory(ctx, s.categories, input.UserID, input.CategoryID, input.Kind)
}
