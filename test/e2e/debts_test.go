//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"
)

func criarDivida(t *testing.T, token string, corpo map[string]any) apiResponse {
	t.Helper()
	return postAuth(t, "/api/debts", corpo, token)
}

// emprestimo é o caso do enunciado: 21 parcelas de R$ 877,66 desde 07/10/2026.
func emprestimo() map[string]any {
	return map[string]any{
		"description":              "Empréstimo",
		"installment_amount_cents": 87766,
		"installments":             21,
		"kind":                     "despesa",
		"frequency":                "mensal",
		"first_due_date":           "2026-10-07",
	}
}

func progressoDe(t *testing.T, res apiResponse) map[string]any {
	t.Helper()

	progresso, ok := res.Body["progress"].(map[string]any)
	if !ok {
		t.Fatalf("esperava o progresso na resposta, veio %v", res.Body)
	}

	return progresso
}

func TestRegistrarDividaTrazOProgresso(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	res := criarDivida(t, token, emprestimo())
	assertStatus(t, res, http.StatusCreated)

	progresso := progressoDe(t, res)

	// 877,66 x 21 = 18.430,86
	if got := number(t, progresso, "total_cents"); got != 1843086 {
		t.Errorf("total = %d, esperava 1843086", got)
	}
	if got := text(t, progresso, "final_due_date"); got != "2028-06-07" {
		t.Errorf("última parcela = %q, esperava 2028-06-07", got)
	}
}

func TestParcelasAparecemNosLancamentosDoMes(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarDivida(t, token, emprestimo())

	_, lista := getList(t, "/api/transactions?year=2026&month=12", token)
	if len(lista) != 1 {
		t.Fatalf("esperava 1 parcela em dezembro, veio %d", len(lista))
	}

	parcela := lista[0]
	if got := text(t, parcela, "occurred_at"); got != "2026-12-07" {
		t.Errorf("data = %q, esperava 2026-12-07", got)
	}
	if got := number(t, parcela, "installment_number"); got != 3 {
		t.Errorf("número da parcela = %d, esperava 3", got)
	}
	if got := number(t, parcela, "installments_total"); got != 21 {
		t.Errorf("total de parcelas = %d, esperava 21", got)
	}
	if projetada, _ := parcela["projected"].(bool); !projetada {
		t.Error("a parcela precisa vir marcada como projeção")
	}
}

func TestNadaAntesDaPrimeiraNemDepoisDaUltimaParcela(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarDivida(t, token, emprestimo())

	_, antes := getList(t, "/api/transactions?year=2026&month=9", token)
	if len(antes) != 0 {
		t.Errorf("esperava nada antes da primeira parcela, veio %d", len(antes))
	}

	// A última vence em junho de 2028.
	_, depois := getList(t, "/api/transactions?year=2028&month=7", token)
	if len(depois) != 0 {
		t.Errorf("a dívida acabou, esperava nada, veio %d", len(depois))
	}
}

func TestParcelaPesaNoSaldoDoMes(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarDivida(t, token, emprestimo())

	res := get(t, "/api/transactions/summary?year=2026&month=10", token)
	assertStatus(t, res, http.StatusOK)

	if got := number(t, res.Body, "saldo_previsto_cents"); got != -87766 {
		t.Errorf("saldo previsto = %d, esperava -87766", got)
	}
}

func TestPainelAgregaAsDividasEmAberto(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarDivida(t, token, emprestimo())

	res := get(t, "/api/dashboard?year=2026&month=10", token)
	assertStatus(t, res, http.StatusOK)

	dividas, ok := res.Body["dividas"].(map[string]any)
	if !ok {
		t.Fatalf("esperava o bloco das dívidas, veio %v", res.Body)
	}

	if got := number(t, dividas, "total_cents"); got != 1843086 {
		t.Errorf("total = %d, esperava 1843086", got)
	}
	if got := number(t, dividas, "open_count"); got != 1 {
		t.Errorf("em aberto = %d, esperava 1", got)
	}
}

func TestDividaComCategoria(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)
	categoria := categoriaCriada(t, token, "Empréstimos", "despesa")

	corpo := emprestimo()
	corpo["category_id"] = categoria

	res := criarDivida(t, token, corpo)
	assertStatus(t, res, http.StatusCreated)

	if res.String("category_name") != "Empréstimos" {
		t.Errorf("categoria = %q, esperava Empréstimos", res.String("category_name"))
	}

	// A parcela projetada carrega a mesma categoria.
	_, lista := getList(t, "/api/transactions?year=2026&month=12", token)
	if got := text(t, lista[0], "category_name"); got != "Empréstimos" {
		t.Errorf("categoria da parcela = %q, esperava Empréstimos", got)
	}
}

func TestValidacaoDaDivida(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	casos := []struct {
		nome      string
		ajuste    func(map[string]any)
		campoErro string
	}{
		{"receita não é dívida", func(c map[string]any) { c["kind"] = "receita" }, "kind"},
		{"sem parcelas", func(c map[string]any) { c["installments"] = 0 }, "installments"},
		{"parcela sem valor", func(c map[string]any) { c["installment_amount_cents"] = 0 }, "installment_amount_cents"},
		{"data inválida", func(c map[string]any) { c["first_due_date"] = "07/10/2026" }, "first_due_date"},
		{"frequência desconhecida", func(c map[string]any) { c["frequency"] = "bimestral" }, "frequency"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			corpo := emprestimo()
			caso.ajuste(corpo)

			res := criarDivida(t, token, corpo)

			assertStatus(t, res, http.StatusUnprocessableEntity)
			if res.Field(caso.campoErro) == "" {
				t.Errorf("esperava erro no campo %q, veio %v", caso.campoErro, res.Body)
			}
		})
	}
}

func TestApagarDividaTiraAsParcelas(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criada := criarDivida(t, token, emprestimo())
	id, _ := criada.Body["id"].(float64)

	assertStatus(t, del(t, fmt.Sprintf("/api/debts/%d", int64(id)), token), http.StatusNoContent)

	_, lista := getList(t, "/api/transactions?year=2026&month=12", token)
	if len(lista) != 0 {
		t.Errorf("apagar a dívida deveria sumir com as parcelas, veio %d", len(lista))
	}
}

func TestUmUsuarioNaoVeADividaDeOutro(t *testing.T) {
	resetDatabase(t)

	tokenA := contaComToken(t)
	tokenB := contaComToken(t)

	criarDivida(t, tokenA, emprestimo())

	_, listaDeB := getList(t, "/api/debts", tokenB)
	if len(listaDeB) != 0 {
		t.Errorf("o outro usuário não deveria ver nada, veio %d", len(listaDeB))
	}
}

func TestDividasExigemAutenticacao(t *testing.T) {
	for _, metodo := range []string{http.MethodGet, http.MethodPost} {
		t.Run(metodo, func(t *testing.T) {
			assertStatus(t, do(t, metodo, "/api/debts", nil, ""), http.StatusUnauthorized)
		})
	}
}
