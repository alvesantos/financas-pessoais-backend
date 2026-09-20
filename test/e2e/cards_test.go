//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"
)

func criarCartao(t *testing.T, token string, corpo map[string]any) apiResponse {
	t.Helper()
	return postAuth(t, "/api/cards", corpo, token)
}

func cartaoPadrao() map[string]any {
	return map[string]any{
		"name":              "Nubank",
		"limit_cents":       500000,
		"best_purchase_day": 5,
		"due_day":           15,
	}
}

func TestCadastrarCartao(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	res := criarCartao(t, token, cartaoPadrao())
	assertStatus(t, res, http.StatusCreated)

	if res.String("name") != "Nubank" {
		t.Errorf("nome = %q, esperava Nubank", res.String("name"))
	}
	if got := number(t, res.Body, "best_purchase_day"); got != 5 {
		t.Errorf("melhor dia = %d, esperava 5", got)
	}
}

func TestEditarCartao(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criado := criarCartao(t, token, cartaoPadrao())
	id, _ := criado.Body["id"].(float64)

	res := put(t, fmt.Sprintf("/api/cards/%d", int64(id)), map[string]any{
		"name": "Nubank Ultravioleta", "limit_cents": 900000,
		"best_purchase_day": 8, "due_day": 18,
	}, token)

	assertStatus(t, res, http.StatusOK)
	if res.String("name") != "Nubank Ultravioleta" {
		t.Errorf("nome = %q, esperava Nubank Ultravioleta", res.String("name"))
	}
}

func TestCartaoRecusaNomeRepetido(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarCartao(t, token, cartaoPadrao())
	repetido := criarCartao(t, token, cartaoPadrao())

	assertStatus(t, repetido, http.StatusConflict)
}

func TestValidacaoDoCartao(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	casos := []struct {
		nome      string
		ajuste    func(map[string]any)
		campoErro string
	}{
		{"sem nome", func(c map[string]any) { c["name"] = "  " }, "name"},
		{"melhor dia fora do mês", func(c map[string]any) { c["best_purchase_day"] = 40 }, "best_purchase_day"},
		{"vencimento fora do mês", func(c map[string]any) { c["due_day"] = 0 }, "due_day"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			corpo := cartaoPadrao()
			caso.ajuste(corpo)

			res := criarCartao(t, token, corpo)

			assertStatus(t, res, http.StatusUnprocessableEntity)
			if res.Field(caso.campoErro) == "" {
				t.Errorf("esperava erro no campo %q, veio %v", caso.campoErro, res.Body)
			}
		})
	}
}

func TestCompraAntesDoMelhorDiaCaiNaFaturaDoMes(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criado := criarCartao(t, token, cartaoPadrao())
	cartao, _ := criado.Body["id"].(float64)

	// Melhor dia é 5; comprar no dia 3 cai na fatura de setembro.
	res := criarLancamento(t, token, map[string]any{
		"description": "Compra", "amount_cents": 10000, "kind": "cartao_credito",
		"occurred_at": "2026-09-03", "credit_card_id": int64(cartao),
	})

	assertStatus(t, res, http.StatusCreated)
	if got := res.String("invoice_month"); got != "2026-09-01" {
		t.Errorf("fatura = %q, esperava 2026-09-01", got)
	}
	if res.String("credit_card_name") != "Nubank" {
		t.Errorf("cartão = %q, esperava Nubank", res.String("credit_card_name"))
	}
}

func TestCompraNoMelhorDiaCaiNaFaturaSeguinte(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criado := criarCartao(t, token, cartaoPadrao())
	cartao, _ := criado.Body["id"].(float64)

	// Comprar no melhor dia empurra para a fatura seguinte: é a vantagem dele.
	res := criarLancamento(t, token, map[string]any{
		"amount_cents": 10000, "kind": "cartao_credito",
		"occurred_at": "2026-09-05", "credit_card_id": int64(cartao),
	})

	assertStatus(t, res, http.StatusCreated)
	if got := res.String("invoice_month"); got != "2026-10-01" {
		t.Errorf("fatura = %q, esperava 2026-10-01", got)
	}
}

func TestEscolherAProximaFatura(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criado := criarCartao(t, token, cartaoPadrao())
	cartao, _ := criado.Body["id"].(float64)

	res := criarLancamento(t, token, map[string]any{
		"amount_cents": 10000, "kind": "cartao_credito",
		"occurred_at": "2026-09-03", "credit_card_id": int64(cartao), "invoice": "proxima",
	})

	assertStatus(t, res, http.StatusCreated)
	if got := res.String("invoice_month"); got != "2026-10-01" {
		t.Errorf("fatura = %q, esperava 2026-10-01", got)
	}
}

func TestCartaoSoValeEmGastoDeCartao(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criado := criarCartao(t, token, cartaoPadrao())
	cartao, _ := criado.Body["id"].(float64)

	res := criarLancamento(t, token, map[string]any{
		"amount_cents": 10000, "kind": "despesa",
		"occurred_at": "2026-09-03", "credit_card_id": int64(cartao),
	})

	assertStatus(t, res, http.StatusUnprocessableEntity)
	if res.Field("credit_card_id") == "" {
		t.Errorf("esperava erro no campo credit_card_id, veio %v", res.Body)
	}
}

func TestCartaoDeOutroUsuarioNaoExiste(t *testing.T) {
	resetDatabase(t)

	tokenA := contaComToken(t)
	tokenB := contaComToken(t)

	criado := criarCartao(t, tokenA, cartaoPadrao())
	cartao, _ := criado.Body["id"].(float64)

	res := criarLancamento(t, tokenB, map[string]any{
		"amount_cents": 10000, "kind": "cartao_credito",
		"occurred_at": "2026-09-03", "credit_card_id": int64(cartao),
	})

	assertStatus(t, res, http.StatusUnprocessableEntity)
}

func TestApagarCartaoDeixaOLancamentoSemEle(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criado := criarCartao(t, token, cartaoPadrao())
	cartao, _ := criado.Body["id"].(float64)

	criarLancamento(t, token, map[string]any{
		"amount_cents": 10000, "kind": "cartao_credito",
		"occurred_at": "2026-09-03", "credit_card_id": int64(cartao),
	})

	assertStatus(t, del(t, fmt.Sprintf("/api/cards/%d", int64(cartao)), token), http.StatusNoContent)

	// ON DELETE SET NULL: o lançamento fica, sem cartão.
	_, lista := getList(t, "/api/transactions?year=2026&month=9", token)
	if len(lista) != 1 {
		t.Fatalf("o lançamento não deveria sumir, veio %d", len(lista))
	}
	if lista[0]["credit_card_id"] != nil {
		t.Errorf("esperava o lançamento sem cartão, veio %v", lista[0]["credit_card_id"])
	}
}
