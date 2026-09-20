package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/alvesantos/financas-backend/internal/domain"
	"github.com/alvesantos/financas-backend/internal/service"
)

func novoServicoDeDividas() (*service.DebtService, *fakeCategoryRepo) {
	categorias := &fakeCategoryRepo{}
	return service.NewDebtService(&fakeDebtRepo{}, categorias), categorias
}

// emprestimoDoEnunciado: 21 parcelas de R$ 877,66 a partir de 7 de outubro.
func emprestimoDoEnunciado() domain.NewDebt {
	return domain.NewDebt{
		UserID:           usuario,
		Description:      "Empréstimo",
		InstallmentCents: 87766,
		Installments:     21,
		Kind:             domain.KindDespesa,
		Frequency:        domain.FrequencyMensal,
		FirstDueDate:     dia(2026, time.October, 7),
	}
}

func TestDividaEhRegistrada(t *testing.T) {
	svc, _ := novoServicoDeDividas()

	criada, err := svc.Create(context.Background(), emprestimoDoEnunciado())
	if err != nil {
		t.Fatalf("registrar dívida: %v", err)
	}

	if criada.TotalCents() != 1843086 {
		t.Errorf("total = %d, esperava 1843086", criada.TotalCents())
	}
}

func TestDividaSemDescricaoUsaONomeDoTipo(t *testing.T) {
	svc, _ := novoServicoDeDividas()

	entrada := emprestimoDoEnunciado()
	entrada.Description = "  "

	criada, err := svc.Create(context.Background(), entrada)
	if err != nil {
		t.Fatalf("registrar dívida: %v", err)
	}

	if criada.Description != "Despesa" {
		t.Errorf("descrição = %q, esperava Despesa", criada.Description)
	}
}

func TestDividaRecusaReceita(t *testing.T) {
	svc, _ := novoServicoDeDividas()

	entrada := emprestimoDoEnunciado()
	entrada.Kind = domain.KindReceita

	_, err := svc.Create(context.Background(), entrada)

	appErr, ok := domain.AsError(err)
	if !ok || appErr.Fields["kind"] == "" {
		t.Errorf("dívida é saída, esperava erro no campo kind, veio %v", err)
	}
}

func TestDividaRecusaParcelasInvalidas(t *testing.T) {
	svc, _ := novoServicoDeDividas()

	casos := map[string]int{"zero": 0, "negativo": -3, "acima do limite": 601}

	for nome, parcelas := range casos {
		t.Run(nome, func(t *testing.T) {
			entrada := emprestimoDoEnunciado()
			entrada.Installments = parcelas

			_, err := svc.Create(context.Background(), entrada)

			appErr, ok := domain.AsError(err)
			if !ok || appErr.Fields["installments"] == "" {
				t.Errorf("esperava erro no campo installments, veio %v", err)
			}
		})
	}
}

func TestDividaRecusaParcelaSemValor(t *testing.T) {
	svc, _ := novoServicoDeDividas()

	entrada := emprestimoDoEnunciado()
	entrada.InstallmentCents = 0

	_, err := svc.Create(context.Background(), entrada)

	appErr, ok := domain.AsError(err)
	if !ok || appErr.Fields["installment_amount_cents"] == "" {
		t.Errorf("esperava erro no campo installment_amount_cents, veio %v", err)
	}
}

func TestDividaRecusaCategoriaDeOutroTipo(t *testing.T) {
	svc, categorias := novoServicoDeDividas()
	categoria := categoriaDe(t, categorias, "Salário", domain.KindReceita)

	entrada := emprestimoDoEnunciado()
	entrada.CategoryID = &categoria.ID

	_, err := svc.Create(context.Background(), entrada)

	appErr, ok := domain.AsError(err)
	if !ok || appErr.Fields["category_id"] == "" {
		t.Errorf("esperava erro no campo category_id, veio %v", err)
	}
}

func TestParcelaApareceNaListaDoMes(t *testing.T) {
	svc, _, _, dividas := novoServicoComDividas(dia(2026, time.December, 31))

	if _, err := dividas.Create(context.Background(), emprestimoDoEnunciado()); err != nil {
		t.Fatalf("registrar dívida: %v", err)
	}

	entradas, err := svc.ListMonth(context.Background(), usuario, 2026, time.December)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}

	if len(entradas) != 1 {
		t.Fatalf("esperava 1 parcela em dezembro, veio %d", len(entradas))
	}

	parcela := entradas[0]
	if *parcela.InstallmentNumber != 3 || *parcela.InstallmentsTotal != 21 {
		t.Errorf("parcela %d/%d, esperava 3/21", *parcela.InstallmentNumber, *parcela.InstallmentsTotal)
	}
}

func TestParcelaPesaNoSaldoDoMes(t *testing.T) {
	svc, _, _, dividas := novoServicoComDividas(dia(2026, time.October, 31))

	_, _ = dividas.Create(context.Background(), emprestimoDoEnunciado())

	resumo, err := svc.Summary(context.Background(), usuario, 2026, time.October)
	if err != nil {
		t.Fatalf("resumo: %v", err)
	}

	if resumo.SaldoPrevisto != -87766 {
		t.Errorf("saldo previsto = %d, esperava -87766", resumo.SaldoPrevisto)
	}
	if resumo.Despesas != 87766 {
		t.Errorf("despesas = %d, esperava 87766", resumo.Despesas)
	}
}

func TestMesDepoisDaUltimaParcelaFicaLimpo(t *testing.T) {
	svc, _, _, dividas := novoServicoComDividas(dia(2028, time.July, 31))

	_, _ = dividas.Create(context.Background(), emprestimoDoEnunciado())

	// A última parcela vence em junho de 2028.
	entradas, _ := svc.ListMonth(context.Background(), usuario, 2028, time.July)

	if len(entradas) != 0 {
		t.Errorf("a dívida acabou, esperava nenhuma parcela, veio %d", len(entradas))
	}
}
