package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/alvesantos/financas-backend/internal/domain"
	"github.com/alvesantos/financas-backend/internal/service"
)

func novoPainel(hoje time.Time) (*service.DashboardService, *fakeTransactionRepo, *fakeRecurringRepo) {
	transacoes := &fakeTransactionRepo{}
	fixos := &fakeRecurringRepo{}

	return service.NewDashboardService(transacoes, fixos, relogioFixo{hoje: hoje}), transacoes, fixos
}

func gravar(t *testing.T, repo *fakeTransactionRepo, descricao string, centavos int64, tipo domain.Kind, data time.Time) {
	t.Helper()

	if _, err := repo.Create(context.Background(), domain.NewTransaction{
		UserID: usuario, Description: descricao, AmountCents: centavos, Kind: tipo, OccurredAt: data,
	}); err != nil {
		t.Fatalf("gravar lançamento: %v", err)
	}
}

func TestPainelSomaOAnoInteiro(t *testing.T) {
	painel, transacoes, _ := novoPainel(dia(2026, time.September, 30))

	gravar(t, transacoes, "Salário jan", 500000, domain.KindReceita, dia(2026, time.January, 5))
	gravar(t, transacoes, "Salário set", 500000, domain.KindReceita, dia(2026, time.September, 5))
	gravar(t, transacoes, "Mercado set", 80000, domain.KindDespesa, dia(2026, time.September, 6))
	// Ano diferente: não pode entrar na conta.
	gravar(t, transacoes, "Salário 2025", 999999, domain.KindReceita, dia(2025, time.September, 5))

	overview, err := painel.Overview(context.Background(), usuario, 2026, time.September)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}

	if overview.Year.Receitas != 1000000 {
		t.Errorf("receitas do ano = %d, esperava 1000000", overview.Year.Receitas)
	}
	if overview.Year.Despesas != 80000 {
		t.Errorf("despesas do ano = %d, esperava 80000", overview.Year.Despesas)
	}
	if overview.Year.Saldo != 920000 {
		t.Errorf("saldo do ano = %d, esperava 920000", overview.Year.Saldo)
	}
}

func TestPainelTemOsDozeMeses(t *testing.T) {
	painel, _, _ := novoPainel(dia(2026, time.September, 30))

	overview, _ := painel.Overview(context.Background(), usuario, 2026, time.September)

	if len(overview.PorMes) != 12 {
		t.Fatalf("esperava 12 meses na série, veio %d", len(overview.PorMes))
	}

	for i, mes := range overview.PorMes {
		if int(mes.Month) != i+1 {
			t.Errorf("posição %d: mês = %d, esperava %d", i, mes.Month, i+1)
		}
	}
}

func TestPainelProjetaOsFixosEmTodosOsMeses(t *testing.T) {
	painel, _, fixos := novoPainel(dia(2026, time.September, 30))

	_, _ = fixos.Create(context.Background(), domain.NewRecurringEntry{
		UserID: usuario, Description: "Academia", AmountCents: 15990,
		Kind: domain.KindDespesa, Frequency: domain.FrequencyMensal,
		StartDate: dia(2026, time.January, 20),
	})

	overview, _ := painel.Overview(context.Background(), usuario, 2026, time.September)

	// Doze meses de academia entram na despesa do ano.
	if esperado := int64(15990 * 12); overview.Year.Despesas != esperado {
		t.Errorf("despesas do ano = %d, esperava %d", overview.Year.Despesas, esperado)
	}
	if overview.PorMes[0].Despesas != 15990 {
		t.Errorf("janeiro = %d, esperava 15990", overview.PorMes[0].Despesas)
	}
}

func TestGastosPorTipoVemDoMaiorParaOMenor(t *testing.T) {
	painel, transacoes, _ := novoPainel(dia(2026, time.September, 30))

	gravar(t, transacoes, "Tesouro", 20000, domain.KindInvestimento, dia(2026, time.September, 2))
	gravar(t, transacoes, "Mercado", 90000, domain.KindDespesa, dia(2026, time.September, 3))
	gravar(t, transacoes, "Cartão", 50000, domain.KindCartaoCredito, dia(2026, time.September, 4))
	gravar(t, transacoes, "Salário", 500000, domain.KindReceita, dia(2026, time.September, 5))

	overview, _ := painel.Overview(context.Background(), usuario, 2026, time.September)

	esperado := []domain.Kind{domain.KindDespesa, domain.KindCartaoCredito, domain.KindInvestimento}
	if len(overview.GastosPorTipo) != len(esperado) {
		t.Fatalf("esperava %d tipos, veio %d", len(esperado), len(overview.GastosPorTipo))
	}

	for i, tipo := range esperado {
		if overview.GastosPorTipo[i].Kind != tipo {
			t.Errorf("posição %d: tipo = %s, esperava %s", i, overview.GastosPorTipo[i].Kind, tipo)
		}
	}

	// Receita não é gasto: fica fora do gráfico de composição.
	for _, gasto := range overview.GastosPorTipo {
		if gasto.Kind.IsIncome() {
			t.Errorf("receita não deveria aparecer nos gastos: %v", gasto)
		}
	}
}

func TestMaiorGastoDoMes(t *testing.T) {
	painel, transacoes, _ := novoPainel(dia(2026, time.September, 30))

	gravar(t, transacoes, "Mercado", 90000, domain.KindDespesa, dia(2026, time.September, 3))
	gravar(t, transacoes, "Aluguel", 200000, domain.KindDespesa, dia(2026, time.September, 4))
	gravar(t, transacoes, "Salário", 900000, domain.KindReceita, dia(2026, time.September, 5))

	overview, _ := painel.Overview(context.Background(), usuario, 2026, time.September)

	if overview.MaiorGasto == nil {
		t.Fatal("esperava um maior gasto")
	}
	if overview.MaiorGasto.Description != "Aluguel" {
		t.Errorf("maior gasto = %q, esperava %q (o salário é maior, mas é receita)",
			overview.MaiorGasto.Description, "Aluguel")
	}
}

func TestPainelSemLancamentosVemZerado(t *testing.T) {
	painel, _, _ := novoPainel(dia(2026, time.September, 30))

	overview, err := painel.Overview(context.Background(), usuario, 2026, time.September)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}

	if overview.Year.Saldo != 0 || overview.Month.SaldoPrevisto != 0 {
		t.Errorf("esperava tudo zerado, veio ano=%d mês=%d", overview.Year.Saldo, overview.Month.SaldoPrevisto)
	}
	if len(overview.GastosPorTipo) != 0 {
		t.Errorf("esperava nenhum gasto, veio %d", len(overview.GastosPorTipo))
	}
	if overview.MaiorGasto != nil {
		t.Error("esperava nenhum maior gasto")
	}
}
