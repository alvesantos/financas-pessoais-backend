package service_test

import (
	"context"

	"github.com/alvesantos/financas-backend/internal/domain"
)

type fakeDebtRepo struct {
	items  []domain.Debt
	nextID int64
}

func (f *fakeDebtRepo) Create(_ context.Context, input domain.NewDebt) (*domain.Debt, error) {
	f.nextID++

	debt := domain.Debt{
		ID:               f.nextID,
		UserID:           input.UserID,
		Description:      input.Description,
		InstallmentCents: input.InstallmentCents,
		Installments:     input.Installments,
		Kind:             input.Kind,
		Frequency:        input.Frequency,
		FirstDueDate:     input.FirstDueDate,
		CategoryID:       input.CategoryID,
	}
	f.items = append(f.items, debt)

	return &debt, nil
}

func (f *fakeDebtRepo) List(_ context.Context, _ int64) ([]domain.Debt, error) {
	return f.items, nil
}

func (f *fakeDebtRepo) Delete(_ context.Context, _, id int64) error {
	for i, item := range f.items {
		if item.ID == id {
			f.items = append(f.items[:i], f.items[i+1:]...)
			return nil
		}
	}
	return domain.ErrDebtNotFound
}
