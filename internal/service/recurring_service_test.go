package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/alvesantos/financas-backend/internal/domain"
	"github.com/alvesantos/financas-backend/internal/service"
)

func TestFixoSemDescricaoUsaONomeDoTipo(t *testing.T) {
	svc := service.NewRecurringService(&fakeRecurringRepo{}, &fakeCategoryRepo{})

	criado, err := svc.Create(context.Background(), domain.NewRecurringEntry{
		UserID: usuario, AmountCents: 15990, Kind: domain.KindCartaoCredito,
		Frequency: domain.FrequencyMensal, StartDate: dia(2026, time.January, 20),
	})
	if err != nil {
		t.Fatalf("criar fixo: %v", err)
	}

	if criado.Description != "Gasto no cartão de crédito" {
		t.Errorf("descrição = %q, esperava %q", criado.Description, "Gasto no cartão de crédito")
	}
}

func TestFixoRecusaFrequenciaInvalida(t *testing.T) {
	svc := service.NewRecurringService(&fakeRecurringRepo{}, &fakeCategoryRepo{})

	_, err := svc.Create(context.Background(), domain.NewRecurringEntry{
		UserID: usuario, AmountCents: 15990, Kind: domain.KindDespesa,
		Frequency: "bimestral", StartDate: dia(2026, time.January, 20),
	})

	appErr, ok := domain.AsError(err)
	if !ok || appErr.Fields["frequency"] == "" {
		t.Errorf("esperava erro de validação no campo frequency, veio %v", err)
	}
}

func TestFixoRecusaFimAntesDoInicio(t *testing.T) {
	svc := service.NewRecurringService(&fakeRecurringRepo{}, &fakeCategoryRepo{})
	fim := dia(2026, time.January, 10)

	_, err := svc.Create(context.Background(), domain.NewRecurringEntry{
		UserID: usuario, AmountCents: 15990, Kind: domain.KindDespesa,
		Frequency: domain.FrequencyMensal, StartDate: dia(2026, time.January, 20), EndDate: &fim,
	})

	appErr, ok := domain.AsError(err)
	if !ok || appErr.Fields["end_date"] == "" {
		t.Errorf("esperava erro de validação no campo end_date, veio %v", err)
	}
}

func TestFixoAceitaTodasAsFrequencias(t *testing.T) {
	for _, frequencia := range domain.AllFrequencies {
		svc := service.NewRecurringService(&fakeRecurringRepo{}, &fakeCategoryRepo{})

		criado, err := svc.Create(context.Background(), domain.NewRecurringEntry{
			UserID: usuario, Description: "Academia", AmountCents: 15990,
			Kind: domain.KindDespesa, Frequency: frequencia, StartDate: dia(2026, time.January, 20),
		})
		if err != nil {
			t.Errorf("%s: %v", frequencia, err)
			continue
		}

		if criado.Frequency != frequencia {
			t.Errorf("frequência = %q, esperava %q", criado.Frequency, frequencia)
		}
	}
}
