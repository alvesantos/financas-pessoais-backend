//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"
)

func amortizar(t *testing.T, token string, id int64, corpo map[string]any) apiResponse {
	t.Helper()
	return postAuth(t, fmt.Sprintf("/api/debts/%d/amortize", id), corpo, token)
}

func dividaRegistrada(t *testing.T, token string) int64 {
	t.Helper()

	criada := criarDivida(t, token, emprestimo())
	if criada.StatusCode != http.StatusCreated {
		t.Fatalf("registrar dívida: %d %v", criada.StatusCode, criada.Body)
	}

	id, _ := criada.Body["id"].(float64)
	return int64(id)
}

func TestAmortizarAbateEMantemOTotal(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)
	id := dividaRegistrada(t, token)

	res := amortizar(t, token, id, map[string]any{"amount_cents": 100000})
	assertStatus(t, res, http.StatusOK)

	progresso := progressoDe(t, res)
	total := number(t, progresso, "total_cents")
	pago := number(t, progresso, "paid_cents")
	resta := number(t, progresso, "remaining_cents")

	if total != 1843086 {
		t.Errorf("total = %d, esperava 1843086", total)
	}
	if pago+resta != total {
		t.Errorf("pago (%d) mais restante (%d) precisa fechar o total (%d)", pago, resta, total)
	}
	if pago < 100000 {
		t.Errorf("pago = %d, esperava ao menos o valor amortizado", pago)
	}
}

func TestAmortizarMantendoAParcela(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)
	id := dividaRegistrada(t, token)

	res := amortizar(t, token, id, map[string]any{
		"amount_cents": 87766 * 2, "mode": "manter_parcela",
	})
	assertStatus(t, res, http.StatusOK)

	if got := number(t, res.Body, "installment_amount_cents"); got != 87766 {
		t.Errorf("parcela = %d, esperava 87766", got)
	}
	if got := number(t, res.Body, "installments"); got != 19 {
		t.Errorf("parcelas = %d, esperava 19", got)
	}
}

func TestAmortizarRecalculandoAParcela(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)
	id := dividaRegistrada(t, token)

	res := amortizar(t, token, id, map[string]any{
		"amount_cents": 87766 * 2, "mode": "recalcular_parcela",
	})
	assertStatus(t, res, http.StatusOK)

	if got := number(t, res.Body, "installments"); got != 21 {
		t.Errorf("parcelas = %d, esperava as 21 que faltavam", got)
	}
	if got := number(t, res.Body, "installment_amount_cents"); got >= 87766 {
		t.Errorf("parcela = %d, esperava um valor menor", got)
	}
}

func TestAmortizarEscolhendoAsParcelas(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)
	id := dividaRegistrada(t, token)

	res := amortizar(t, token, id, map[string]any{
		"amount_cents": 100000, "mode": "recalcular_parcelas", "installments": 10,
	})
	assertStatus(t, res, http.StatusOK)

	if got := number(t, res.Body, "installments"); got != 10 {
		t.Errorf("parcelas = %d, esperava 10", got)
	}
}

func TestAmortizarMaisQueODevidoEhRecusadoNaAPI(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)
	id := dividaRegistrada(t, token)

	res := amortizar(t, token, id, map[string]any{"amount_cents": 99999999})

	assertStatus(t, res, http.StatusUnprocessableEntity)
	if res.Field("amount_cents") == "" {
		t.Errorf("esperava erro no campo amount_cents, veio %v", res.Body)
	}
}

func TestQuitarSemSubtrairDoSaldo(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)
	id := dividaRegistrada(t, token)

	res := postAuth(t, fmt.Sprintf("/api/debts/%d/settle", id),
		map[string]any{"subtract_from_balance": false}, token)
	assertStatus(t, res, http.StatusOK)

	progresso := progressoDe(t, res)
	if quitada, _ := progresso["settled"].(bool); !quitada {
		t.Error("esperava a dívida quitada")
	}

	// Nenhum lançamento novo no mês da primeira parcela.
	_, lista := getList(t, "/api/transactions?year=2026&month=10", token)
	if len(lista) != 0 {
		t.Errorf("esperava nenhum lançamento, veio %d", len(lista))
	}
}

func TestQuitarSubtraindoDoSaldoCriaLancamentoPago(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)
	id := dividaRegistrada(t, token)

	res := postAuth(t, fmt.Sprintf("/api/debts/%d/settle", id),
		map[string]any{"subtract_from_balance": true}, token)
	assertStatus(t, res, http.StatusOK)

	painel := get(t, "/api/dashboard?year=2026&month=10", token)
	if got := number(t, painel.Body, "saldo_atual_cents"); got != -1843086 {
		t.Errorf("saldo atual = %d, esperava -1843086", got)
	}
}

func TestDividaQuitadaSomeDosLancamentos(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)
	id := dividaRegistrada(t, token)

	postAuth(t, fmt.Sprintf("/api/debts/%d/settle", id), map[string]any{}, token)

	_, lista := getList(t, "/api/transactions?year=2026&month=12", token)
	if len(lista) != 0 {
		t.Errorf("dívida quitada não projeta parcela, veio %d", len(lista))
	}
}
