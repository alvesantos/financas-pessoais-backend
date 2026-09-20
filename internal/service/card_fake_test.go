package service_test

import (
	"context"

	"github.com/alvesantos/financas-backend/internal/domain"
)

type fakeCardRepo struct {
	items  []domain.CreditCard
	nextID int64
}

func (f *fakeCardRepo) Create(_ context.Context, input domain.NewCreditCard) (*domain.CreditCard, error) {
	f.nextID++

	card := domain.CreditCard{
		ID: f.nextID, UserID: input.UserID, Name: input.Name,
		LimitCents: input.LimitCents, BestPurchaseDay: input.BestPurchaseDay, DueDay: input.DueDay,
	}
	f.items = append(f.items, card)

	return &card, nil
}

func (f *fakeCardRepo) Update(_ context.Context, card domain.CreditCard) (*domain.CreditCard, error) {
	for i := range f.items {
		if f.items[i].ID == card.ID {
			f.items[i] = card
			return &f.items[i], nil
		}
	}
	return nil, domain.ErrCreditCardNotFound
}

func (f *fakeCardRepo) List(_ context.Context, _ int64) ([]domain.CreditCard, error) {
	return f.items, nil
}

func (f *fakeCardRepo) FindByID(_ context.Context, _, id int64) (*domain.CreditCard, error) {
	for i := range f.items {
		if f.items[i].ID == id {
			return &f.items[i], nil
		}
	}
	return nil, domain.ErrCreditCardNotFound
}

func (f *fakeCardRepo) Delete(_ context.Context, _, id int64) error {
	for i, item := range f.items {
		if item.ID == id {
			f.items = append(f.items[:i], f.items[i+1:]...)
			return nil
		}
	}
	return domain.ErrCreditCardNotFound
}
