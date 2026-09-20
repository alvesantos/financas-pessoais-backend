//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"
)

func TestEditarLancamento(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criado := criarLancamento(t, token, map[string]any{
		"description": "Mercado", "amount_cents": 8550, "kind": "despesa", "occurred_at": "2026-09-10",
	})
	id, _ := criado.Body["id"].(float64)

	res := put(t, fmt.Sprintf("/api/transactions/%d", int64(id)), map[string]any{
		"description": "Feira", "amount_cents": 9000, "kind": "despesa", "occurred_at": "2026-09-12",
	}, token)

	assertStatus(t, res, http.StatusOK)
	if res.String("description") != "Feira" {
		t.Errorf("descrição = %q, esperava Feira", res.String("description"))
	}
	if got := number(t, res.Body, "amount_cents"); got != 9000 {
		t.Errorf("valor = %d, esperava 9000", got)
	}
	if res.String("occurred_at") != "2026-09-12" {
		t.Errorf("data = %q, esperava 2026-09-12", res.String("occurred_at"))
	}
}

func TestEditarLancamentoDeOutroUsuarioEhRecusado(t *testing.T) {
	resetDatabase(t)

	tokenA := contaComToken(t)
	tokenB := contaComToken(t)

	criado := criarLancamento(t, tokenA, map[string]any{
		"amount_cents": 8550, "kind": "despesa", "occurred_at": "2026-09-10",
	})
	id, _ := criado.Body["id"].(float64)

	res := put(t, fmt.Sprintf("/api/transactions/%d", int64(id)), map[string]any{
		"amount_cents": 1, "kind": "despesa", "occurred_at": "2026-09-10",
	}, tokenB)

	assertStatus(t, res, http.StatusNotFound)
}

func TestMarcarLancamentoComoPago(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	// Data passada, marcado como não pago.
	criado := criarLancamento(t, token, map[string]any{
		"description": "Conta de luz", "amount_cents": 5000, "kind": "despesa",
		"occurred_at": "2026-09-10", "paid": false,
	})
	assertStatus(t, criado, http.StatusCreated)

	if pago, _ := criado.Body["paid"].(bool); pago {
		t.Error("esperava o lançamento como não pago")
	}

	resumo := get(t, "/api/transactions/summary?year=2026&month=9", token)
	if got := number(t, resumo.Body, "saldo_atual_cents"); got != 0 {
		t.Errorf("saldo atual = %d, esperava 0 enquanto não for pago", got)
	}
	if got := number(t, resumo.Body, "saldo_previsto_cents"); got != -5000 {
		t.Errorf("saldo previsto = %d, esperava -5000", got)
	}

	// Depois de pago, entra no saldo atual.
	id, _ := criado.Body["id"].(float64)
	put(t, fmt.Sprintf("/api/transactions/%d", int64(id)), map[string]any{
		"description": "Conta de luz", "amount_cents": 5000, "kind": "despesa",
		"occurred_at": "2026-09-10", "paid": true,
	}, token)

	depois := get(t, "/api/transactions/summary?year=2026&month=9", token)
	if got := number(t, depois.Body, "saldo_atual_cents"); got != -5000 {
		t.Errorf("saldo atual = %d, esperava -5000 depois de pago", got)
	}
}

func TestEditarFixo(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criado := criarFixo(t, token, map[string]any{
		"description": "Academia", "amount_cents": 15990, "kind": "despesa",
		"frequency": "mensal", "start_date": "2026-01-20",
	})
	id, _ := criado.Body["id"].(float64)

	res := put(t, fmt.Sprintf("/api/recurring/%d", int64(id)), map[string]any{
		"description": "Academia nova", "amount_cents": 19990, "kind": "despesa",
		"frequency": "mensal", "start_date": "2026-01-25",
	}, token)

	assertStatus(t, res, http.StatusOK)
	if res.String("description") != "Academia nova" {
		t.Errorf("descrição = %q, esperava Academia nova", res.String("description"))
	}

	// A projeção acompanha o fixo editado.
	_, lista := getList(t, "/api/transactions?year=2026&month=9", token)
	if len(lista) != 1 {
		t.Fatalf("esperava 1 projeção, veio %d", len(lista))
	}
	if got := text(t, lista[0], "occurred_at"); got != "2026-09-25" {
		t.Errorf("data = %q, esperava 2026-09-25", got)
	}
}

func TestEditarCategoria(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)
	id := categoriaCriada(t, token, "Mercado", "despesa")

	res := put(t, fmt.Sprintf("/api/categories/%d", id), map[string]any{
		"name": "Supermercado", "kind": "despesa", "color": "#112233",
	}, token)

	assertStatus(t, res, http.StatusOK)
	if res.String("name") != "Supermercado" {
		t.Errorf("nome = %q, esperava Supermercado", res.String("name"))
	}
	if res.String("color") != "#112233" {
		t.Errorf("cor = %q, esperava #112233", res.String("color"))
	}
}

func TestEditarDivida(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criada := criarDivida(t, token, emprestimo())
	id, _ := criada.Body["id"].(float64)

	res := put(t, fmt.Sprintf("/api/debts/%d", int64(id)), map[string]any{
		"description": "Empréstimo renegociado", "installment_amount_cents": 50000,
		"installments": 12, "kind": "despesa", "frequency": "mensal",
		"first_due_date": "2026-11-07",
	}, token)

	assertStatus(t, res, http.StatusOK)

	progresso := progressoDe(t, res)
	if got := number(t, progresso, "total_cents"); got != 600000 {
		t.Errorf("total = %d, esperava 600000", got)
	}
}
