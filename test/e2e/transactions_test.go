//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// contaComToken cria um usuário novo e devolve o token dele.
func contaComToken(t *testing.T) string {
	t.Helper()

	email := fmt.Sprintf("lancamentos-%d@teste.com", time.Now().UnixNano())
	return registerUser(t, "Gabe", email, "senha12345")
}

func criarLancamento(t *testing.T, token string, corpo map[string]any) apiResponse {
	t.Helper()
	return postAuth(t, "/api/transactions", corpo, token)
}

func TestCriarLancamentoComDescricao(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	res := criarLancamento(t, token, map[string]any{
		"description":  "Mercado",
		"amount_cents": 8550,
		"kind":         "despesa",
		"occurred_at":  "2026-09-10",
	})

	assertStatus(t, res, http.StatusCreated)

	if res.String("description") != "Mercado" {
		t.Errorf("descrição = %q, esperava %q", res.String("description"), "Mercado")
	}
	// Despesa subtrai saldo: o valor assinado vem negativo.
	if valor, _ := res.Body["signed_cents"].(float64); int64(valor) != -8550 {
		t.Errorf("signed_cents = %v, esperava -8550", res.Body["signed_cents"])
	}
}

func TestLancamentoSemDescricaoRecebeONomeDoTipo(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	casos := map[string]string{
		"receita":        "Receita",
		"despesa":        "Despesa",
		"cartao_credito": "Gasto no cartão de crédito",
		"investimento":   "Investimento",
	}

	for tipo, esperado := range casos {
		t.Run(tipo, func(t *testing.T) {
			res := criarLancamento(t, token, map[string]any{
				"description":  "",
				"amount_cents": 10000,
				"kind":         tipo,
				"occurred_at":  "2026-09-10",
			})

			assertStatus(t, res, http.StatusCreated)
			if res.String("description") != esperado {
				t.Errorf("descrição = %q, esperava %q", res.String("description"), esperado)
			}
		})
	}
}

func TestSomenteReceitaSomaSaldo(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarLancamento(t, token, map[string]any{"amount_cents": 500000, "kind": "receita", "occurred_at": "2026-09-05"})
	criarLancamento(t, token, map[string]any{"amount_cents": 50000, "kind": "despesa", "occurred_at": "2026-09-06"})
	criarLancamento(t, token, map[string]any{"amount_cents": 30000, "kind": "cartao_credito", "occurred_at": "2026-09-07"})
	criarLancamento(t, token, map[string]any{"amount_cents": 20000, "kind": "investimento", "occurred_at": "2026-09-08"})

	res := get(t, "/api/transactions/summary?year=2026&month=9", token)
	assertStatus(t, res, http.StatusOK)

	previsto, _ := res.Body["saldo_previsto_cents"].(float64)
	if int64(previsto) != 400000 {
		t.Errorf("saldo previsto = %v, esperava 400000", previsto)
	}

	receitas, _ := res.Body["receitas_cents"].(float64)
	despesas, _ := res.Body["despesas_cents"].(float64)
	if int64(receitas) != 500000 || int64(despesas) != 100000 {
		t.Errorf("receitas = %v, despesas = %v; esperava 500000 e 100000", receitas, despesas)
	}
}

func TestListaSeparaPorMesEAno(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarLancamento(t, token, map[string]any{"description": "Agosto", "amount_cents": 1000, "kind": "despesa", "occurred_at": "2026-08-31"})
	criarLancamento(t, token, map[string]any{"description": "Setembro", "amount_cents": 1000, "kind": "despesa", "occurred_at": "2026-09-15"})
	criarLancamento(t, token, map[string]any{"description": "Outubro", "amount_cents": 1000, "kind": "despesa", "occurred_at": "2026-10-01"})
	criarLancamento(t, token, map[string]any{"description": "Ano passado", "amount_cents": 1000, "kind": "despesa", "occurred_at": "2025-09-15"})

	status, lista := getList(t, "/api/transactions?year=2026&month=9", token)
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}

	if len(lista) != 1 {
		t.Fatalf("esperava 1 lançamento em setembro/2026, veio %d", len(lista))
	}
	if got := text(t, lista[0], "description"); got != "Setembro" {
		t.Errorf("descrição = %q, esperava %q", got, "Setembro")
	}
}

func TestListaVemOrdenadaPorData(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarLancamento(t, token, map[string]any{"description": "Dia 25", "amount_cents": 1000, "kind": "despesa", "occurred_at": "2026-09-25"})
	criarLancamento(t, token, map[string]any{"description": "Dia 5", "amount_cents": 1000, "kind": "despesa", "occurred_at": "2026-09-05"})
	criarLancamento(t, token, map[string]any{"description": "Dia 15", "amount_cents": 1000, "kind": "despesa", "occurred_at": "2026-09-15"})

	_, lista := getList(t, "/api/transactions?year=2026&month=9", token)

	esperado := []string{"2026-09-05", "2026-09-15", "2026-09-25"}
	if len(lista) != len(esperado) {
		t.Fatalf("esperava %d lançamentos, veio %d", len(esperado), len(lista))
	}

	for i, data := range esperado {
		if got := text(t, lista[i], "occurred_at"); got != data {
			t.Errorf("posição %d: data = %q, esperava %q", i, got, data)
		}
	}
}

func TestValidacaoDoLancamento(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	casos := []struct {
		nome      string
		corpo     map[string]any
		campoErro string
	}{
		{"valor zero", map[string]any{"amount_cents": 0, "kind": "despesa", "occurred_at": "2026-09-10"}, "amount_cents"},
		{"valor negativo", map[string]any{"amount_cents": -500, "kind": "despesa", "occurred_at": "2026-09-10"}, "amount_cents"},
		{"tipo desconhecido", map[string]any{"amount_cents": 500, "kind": "pix", "occurred_at": "2026-09-10"}, "kind"},
		{"data inválida", map[string]any{"amount_cents": 500, "kind": "despesa", "occurred_at": "10/09/2026"}, "occurred_at"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			res := criarLancamento(t, token, caso.corpo)

			assertStatus(t, res, http.StatusUnprocessableEntity)
			if res.Field(caso.campoErro) == "" {
				t.Errorf("esperava erro no campo %q, veio %v", caso.campoErro, res.Body)
			}
		})
	}
}

func TestApagarLancamento(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criado := criarLancamento(t, token, map[string]any{
		"description": "Mercado", "amount_cents": 8550, "kind": "despesa", "occurred_at": "2026-09-10",
	})
	id, _ := criado.Body["id"].(float64)

	apagar := del(t, fmt.Sprintf("/api/transactions/%d", int64(id)), token)
	assertStatus(t, apagar, http.StatusNoContent)

	_, lista := getList(t, "/api/transactions?year=2026&month=9", token)
	if len(lista) != 0 {
		t.Errorf("esperava a lista vazia depois de apagar, veio %d", len(lista))
	}
}

func TestUmUsuarioNaoVeNemApagaOLancamentoDeOutro(t *testing.T) {
	resetDatabase(t)

	tokenA := contaComToken(t)
	tokenB := contaComToken(t)

	criado := criarLancamento(t, tokenA, map[string]any{
		"description": "Segredo", "amount_cents": 8550, "kind": "despesa", "occurred_at": "2026-09-10",
	})
	id, _ := criado.Body["id"].(float64)

	_, listaDeB := getList(t, "/api/transactions?year=2026&month=9", tokenB)
	if len(listaDeB) != 0 {
		t.Errorf("o outro usuário não deveria ver nada, veio %d", len(listaDeB))
	}

	apagar := del(t, fmt.Sprintf("/api/transactions/%d", int64(id)), tokenB)
	assertStatus(t, apagar, http.StatusNotFound)

	// E o dono continua com o lançamento dele.
	_, listaDeA := getList(t, "/api/transactions?year=2026&month=9", tokenA)
	if len(listaDeA) != 1 {
		t.Errorf("o dono deveria continuar com 1 lançamento, veio %d", len(listaDeA))
	}
}

func TestLancamentosExigemAutenticacao(t *testing.T) {
	rotas := []struct {
		metodo string
		rota   string
	}{
		{http.MethodGet, "/api/transactions"},
		{http.MethodGet, "/api/transactions/summary"},
		{http.MethodPost, "/api/transactions"},
		{http.MethodGet, "/api/recurring"},
		{http.MethodPost, "/api/recurring"},
		{http.MethodGet, "/api/dashboard"},
	}

	for _, caso := range rotas {
		t.Run(caso.metodo+" "+caso.rota, func(t *testing.T) {
			res := do(t, caso.metodo, caso.rota, nil, "")
			assertStatus(t, res, http.StatusUnauthorized)
		})
	}
}
