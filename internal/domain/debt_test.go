package domain_test

import (
	"testing"
	"time"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// emprestimo é o caso do enunciado: 21 parcelas de R$ 877,66, mensais, a
// partir de 7 de outubro de 2026.
func emprestimo() domain.Debt {
	return domain.Debt{
		ID:               1,
		UserID:           1,
		Description:      "Empréstimo",
		InstallmentCents: 87766,
		Installments:     21,
		Kind:             domain.KindDespesa,
		Frequency:        domain.FrequencyMensal,
		FirstDueDate:     data(2026, time.October, 7),
	}
}

func TestTotalDaDivida(t *testing.T) {
	// 877,66 x 21 = 18.430,86
	if got := emprestimo().TotalCents(); got != 1843086 {
		t.Errorf("total = %d, esperava 1843086", got)
	}
}

func TestUltimaParcelaCai21MesesDepois(t *testing.T) {
	// Primeira em outubro de 2026, vigésima primeira em junho de 2028.
	final := emprestimo().FinalDueDate()

	if final.Year() != 2028 || final.Month() != time.June || final.Day() != 7 {
		t.Errorf("última parcela = %s, esperava 2028-06-07", final.Format("2006-01-02"))
	}
}

func TestCronogramaTemUmaLinhaPorParcela(t *testing.T) {
	cronograma := emprestimo().Schedule(data(2026, time.October, 7))

	if len(cronograma) != 21 {
		t.Fatalf("esperava 21 parcelas, veio %d", len(cronograma))
	}
	if cronograma[0].Number != 1 || cronograma[20].Number != 21 {
		t.Errorf("parcelas numeradas de %d a %d, esperava de 1 a 21",
			cronograma[0].Number, cronograma[20].Number)
	}
}

func TestProgressoNoComecoDaDivida(t *testing.T) {
	// Um dia antes da primeira parcela: nada vencido.
	progresso := emprestimo().Progress(data(2026, time.October, 6))

	if progresso.PaidCount != 0 || progresso.PaidCents != 0 {
		t.Errorf("pago = %d parcelas / %d centavos, esperava zero",
			progresso.PaidCount, progresso.PaidCents)
	}
	if progresso.RemainingCount != 21 {
		t.Errorf("restam %d parcelas, esperava 21", progresso.RemainingCount)
	}
	if progresso.Percent != 0 {
		t.Errorf("percentual = %d, esperava 0", progresso.Percent)
	}
	if progresso.Settled {
		t.Error("a dívida não está quitada")
	}
}

func TestParcelaQueVenceHojeContaComoPaga(t *testing.T) {
	progresso := emprestimo().Progress(data(2026, time.October, 7))

	if progresso.PaidCount != 1 {
		t.Errorf("pago = %d, esperava 1", progresso.PaidCount)
	}
	if progresso.PaidCents != 87766 {
		t.Errorf("pago = %d centavos, esperava 87766", progresso.PaidCents)
	}
}

func TestProgressoNoMeioDaDivida(t *testing.T) {
	// Em 7 de julho de 2027 já venceram 10 parcelas.
	progresso := emprestimo().Progress(data(2027, time.July, 7))

	if progresso.PaidCount != 10 {
		t.Errorf("pago = %d parcelas, esperava 10", progresso.PaidCount)
	}
	if progresso.RemainingCount != 11 {
		t.Errorf("restam %d parcelas, esperava 11", progresso.RemainingCount)
	}
	if progresso.RemainingCents != 87766*11 {
		t.Errorf("resta %d, esperava %d", progresso.RemainingCents, 87766*11)
	}
	// 877660 de 1843086 é 47,6%, que trunca em 47.
	if progresso.Percent != 47 {
		t.Errorf("percentual = %d, esperava 47", progresso.Percent)
	}
}

func TestProximoVencimento(t *testing.T) {
	progresso := emprestimo().Progress(data(2026, time.October, 7))

	if progresso.NextDueDate == nil {
		t.Fatal("esperava um próximo vencimento")
	}
	if got := progresso.NextDueDate.Format("2006-01-02"); got != "2026-11-07" {
		t.Errorf("próximo vencimento = %s, esperava 2026-11-07", got)
	}
}

func TestDividaQuitada(t *testing.T) {
	progresso := emprestimo().Progress(data(2028, time.July, 1))

	if !progresso.Settled {
		t.Error("depois da última parcela a dívida está quitada")
	}
	if progresso.Percent != 100 {
		t.Errorf("percentual = %d, esperava 100", progresso.Percent)
	}
	if progresso.RemainingCents != 0 {
		t.Errorf("resta %d, esperava 0", progresso.RemainingCents)
	}
	if progresso.NextDueDate != nil {
		t.Errorf("dívida quitada não tem próximo vencimento, veio %v", progresso.NextDueDate)
	}
}

func TestParcelaDoDia31PrendeAoUltimoDiaDoMes(t *testing.T) {
	divida := emprestimo()
	divida.FirstDueDate = data(2026, time.January, 31)

	// Fevereiro de 2026 tem 28 dias: a parcela não pode vazar para março.
	if got := divida.DueDateOf(2); got.Day() != 28 || got.Month() != time.February {
		t.Errorf("segunda parcela = %s, esperava 2026-02-28", got.Format("2006-01-02"))
	}
}

func TestProjecaoDaParcelaNoMes(t *testing.T) {
	projetadas := emprestimo().ProjectInto(domain.MonthPeriod(2026, time.December))

	if len(projetadas) != 1 {
		t.Fatalf("esperava 1 parcela em dezembro, veio %d", len(projetadas))
	}

	parcela := projetadas[0]
	if parcela.OccurredAt.Day() != 7 {
		t.Errorf("dia = %d, esperava 7", parcela.OccurredAt.Day())
	}
	if *parcela.InstallmentNumber != 3 || *parcela.InstallmentsTotal != 21 {
		t.Errorf("parcela %d/%d, esperava 3/21", *parcela.InstallmentNumber, *parcela.InstallmentsTotal)
	}
	if !parcela.IsProjected() {
		t.Error("a parcela precisa vir marcada como projeção")
	}
	if parcela.SignedAmount() != -87766 {
		t.Errorf("efeito no saldo = %d, esperava -87766", parcela.SignedAmount())
	}
}

func TestNadaProjetadoAntesDaPrimeiraParcela(t *testing.T) {
	projetadas := emprestimo().ProjectInto(domain.MonthPeriod(2026, time.September))

	if len(projetadas) != 0 {
		t.Errorf("esperava nenhuma parcela antes do começo, veio %d", len(projetadas))
	}
}

func TestNadaProjetadoDepoisDaUltimaParcela(t *testing.T) {
	// Julho de 2028 é o mês seguinte ao da última parcela.
	projetadas := emprestimo().ProjectInto(domain.MonthPeriod(2028, time.July))

	if len(projetadas) != 0 {
		t.Errorf("a dívida acabou, esperava nenhuma parcela, veio %d", len(projetadas))
	}
}

func TestDividaSemanalProjetaVariasParcelasNoMes(t *testing.T) {
	divida := emprestimo()
	divida.Frequency = domain.FrequencySemanal
	divida.FirstDueDate = data(2026, time.October, 1)

	projetadas := divida.ProjectInto(domain.MonthPeriod(2026, time.October))

	// 1, 8, 15, 22 e 29 de outubro.
	if len(projetadas) != 5 {
		t.Errorf("esperava 5 parcelas em outubro, veio %d", len(projetadas))
	}
}
