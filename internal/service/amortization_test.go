package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// dividaDeTeste registra o empréstimo do enunciado e devolve o id.
func dividaDeTeste(t *testing.T, dividas *fakeDebtRepo) int64 {
	t.Helper()

	criada, err := dividas.Create(context.Background(), emprestimoDoEnunciado())
	if err != nil {
		t.Fatalf("registrar dívida: %v", err)
	}

	return criada.ID
}

func TestAmortizarAbateOSaldoDevedor(t *testing.T) {
	// Uma parcela vencida: faltam 20 de 21.
	svc, _, dividas, _ := servicoDeDividasCompleto(dia(2026, time.October, 7))
	id := dividaDeTeste(t, dividas)

	atualizada, err := svc.Amortize(context.Background(), usuario, id, domain.Amortization{
		AmountCents: 100000,
	})
	if err != nil {
		t.Fatalf("amortizar: %v", err)
	}

	progresso := atualizada.Progress(dia(2026, time.October, 7))
	esperado := int64(87766*20) - 100000

	// A sobra da divisão entre as parcelas vira valor amortizado, então o
	// restante pode ficar alguns centavos abaixo do esperado na conta redonda.
	if progresso.RemainingCents > esperado || esperado-progresso.RemainingCents >= int64(atualizada.Installments) {
		t.Errorf("resta %d, esperava algo logo abaixo de %d", progresso.RemainingCents, esperado)
	}

	// O total não muda: o abatimento só troca de lado, de a pagar para pago.
	if progresso.TotalCents != 1843086 {
		t.Errorf("total = %d, esperava 1843086", progresso.TotalCents)
	}
	if progresso.PaidCents+progresso.RemainingCents != progresso.TotalCents {
		t.Errorf("pago (%d) mais restante (%d) precisa fechar o total (%d)",
			progresso.PaidCents, progresso.RemainingCents, progresso.TotalCents)
	}
}

func TestAmortizarMantendoAParcelaEncurtaOParcelamento(t *testing.T) {
	svc, _, dividas, _ := servicoDeDividasCompleto(dia(2026, time.October, 7))
	id := dividaDeTeste(t, dividas)

	// Duas parcelas de abatimento, mantendo o valor de cada uma.
	atualizada, err := svc.Amortize(context.Background(), usuario, id, domain.Amortization{
		AmountCents: 87766 * 2,
		Mode:        domain.AmortizeKeepInstallment,
	})
	if err != nil {
		t.Fatalf("amortizar: %v", err)
	}

	if atualizada.InstallmentCents != 87766 {
		t.Errorf("parcela = %d, esperava 87766 (o modo mantém o valor)", atualizada.InstallmentCents)
	}
	if atualizada.Installments != 18 {
		t.Errorf("parcelas = %d, esperava 18 (20 menos as 2 amortizadas)", atualizada.Installments)
	}
}

func TestAmortizarRecalculandoAParcelaMantemAQuantidade(t *testing.T) {
	svc, _, dividas, _ := servicoDeDividasCompleto(dia(2026, time.October, 7))
	id := dividaDeTeste(t, dividas)

	atualizada, err := svc.Amortize(context.Background(), usuario, id, domain.Amortization{
		AmountCents: 87766 * 2,
		Mode:        domain.AmortizeRecalcInstallment,
	})
	if err != nil {
		t.Fatalf("amortizar: %v", err)
	}

	if atualizada.Installments != 20 {
		t.Errorf("parcelas = %d, esperava 20 (o modo mantém a quantidade)", atualizada.Installments)
	}
	if atualizada.InstallmentCents >= 87766 {
		t.Errorf("parcela = %d, esperava um valor menor que 87766", atualizada.InstallmentCents)
	}
}

func TestAmortizarEscolhendoQuantasParcelasFicam(t *testing.T) {
	svc, _, dividas, _ := servicoDeDividasCompleto(dia(2026, time.October, 7))
	id := dividaDeTeste(t, dividas)

	dez := 10
	atualizada, err := svc.Amortize(context.Background(), usuario, id, domain.Amortization{
		AmountCents:  100000,
		Mode:         domain.AmortizeRecalcCount,
		Installments: &dez,
	})
	if err != nil {
		t.Fatalf("amortizar: %v", err)
	}

	if atualizada.Installments != 10 {
		t.Errorf("parcelas = %d, esperava 10", atualizada.Installments)
	}
}

func TestAmortizarComNovoSaldoFinal(t *testing.T) {
	svc, _, dividas, _ := servicoDeDividasCompleto(dia(2026, time.October, 7))
	id := dividaDeTeste(t, dividas)

	novoSaldo := int64(500000)
	atualizada, err := svc.Amortize(context.Background(), usuario, id, domain.Amortization{
		AmountCents:       100000,
		NewRemainingCents: &novoSaldo,
	})
	if err != nil {
		t.Fatalf("amortizar: %v", err)
	}

	// O saldo informado manda. A sobra da divisão entre as parcelas vira
	// valor amortizado, então o restante pode ficar alguns centavos abaixo.
	got := atualizada.Progress(dia(2026, time.October, 7)).RemainingCents
	if got > novoSaldo || novoSaldo-got >= int64(atualizada.Installments) {
		t.Errorf("resta %d, esperava algo logo abaixo de %d", got, novoSaldo)
	}
}

func TestAmortizarTudoQuitaADivida(t *testing.T) {
	hoje := dia(2026, time.October, 7)
	svc, _, dividas, _ := servicoDeDividasCompleto(hoje)
	id := dividaDeTeste(t, dividas)

	atualizada, err := svc.Amortize(context.Background(), usuario, id, domain.Amortization{
		AmountCents: 87766 * 20,
	})
	if err != nil {
		t.Fatalf("amortizar: %v", err)
	}

	progresso := atualizada.Progress(hoje)
	if !progresso.Settled || progresso.RemainingCents != 0 {
		t.Errorf("esperava a dívida quitada, veio settled=%v resta=%d",
			progresso.Settled, progresso.RemainingCents)
	}
}

func TestAmortizarMaisQueODevidoEhRecusado(t *testing.T) {
	svc, _, dividas, _ := servicoDeDividasCompleto(dia(2026, time.October, 7))
	id := dividaDeTeste(t, dividas)

	_, err := svc.Amortize(context.Background(), usuario, id, domain.Amortization{
		AmountCents: 99999999,
	})

	appErr, ok := domain.AsError(err)
	if !ok || appErr.Fields["amount_cents"] == "" {
		t.Errorf("esperava erro no campo amount_cents, veio %v", err)
	}
}

func TestQuitarSemMexerNoSaldo(t *testing.T) {
	hoje := dia(2026, time.October, 7)
	svc, _, dividas, transacoes := servicoDeDividasCompleto(hoje)
	id := dividaDeTeste(t, dividas)

	quitada, err := svc.Settle(context.Background(), usuario, id, false)
	if err != nil {
		t.Fatalf("quitar: %v", err)
	}

	if !quitada.IsSettled() {
		t.Error("esperava a dívida quitada")
	}
	if len(transacoes.items) != 0 {
		t.Errorf("sem subtrair do saldo, nenhum lançamento deveria surgir, veio %d", len(transacoes.items))
	}
}

func TestQuitarSubtraindoDoSaldoCriaOLancamento(t *testing.T) {
	hoje := dia(2026, time.October, 7)
	svc, _, dividas, transacoes := servicoDeDividasCompleto(hoje)
	id := dividaDeTeste(t, dividas)

	if _, err := svc.Settle(context.Background(), usuario, id, true); err != nil {
		t.Fatalf("quitar: %v", err)
	}

	if len(transacoes.items) != 1 {
		t.Fatalf("esperava 1 lançamento de quitação, veio %d", len(transacoes.items))
	}

	lancamento := transacoes.items[0]
	if lancamento.AmountCents != 87766*20 {
		t.Errorf("valor = %d, esperava o que faltava (%d)", lancamento.AmountCents, 87766*20)
	}
	if !lancamento.Paid {
		t.Error("a quitação já saiu do bolso: o lançamento precisa vir pago")
	}
}

func TestQuitarDuasVezesEhRecusado(t *testing.T) {
	svc, _, dividas, _ := servicoDeDividasCompleto(dia(2026, time.October, 7))
	id := dividaDeTeste(t, dividas)

	if _, err := svc.Settle(context.Background(), usuario, id, false); err != nil {
		t.Fatalf("quitar: %v", err)
	}

	_, err := svc.Settle(context.Background(), usuario, id, false)
	if err == nil {
		t.Error("esperava recusa ao quitar uma dívida já quitada")
	}
}

func TestDividaQuitadaNaoProjetaMaisParcelas(t *testing.T) {
	hoje := dia(2026, time.October, 7)
	svc, _, dividas, _ := servicoDeDividasCompleto(hoje)
	id := dividaDeTeste(t, dividas)

	if _, err := svc.Settle(context.Background(), usuario, id, false); err != nil {
		t.Fatalf("quitar: %v", err)
	}

	quitada, _ := dividas.FindByID(context.Background(), usuario, id)
	if projetadas := quitada.ProjectInto(domain.MonthPeriod(2026, time.December)); len(projetadas) != 0 {
		t.Errorf("dívida quitada não projeta parcela, veio %d", len(projetadas))
	}
}
