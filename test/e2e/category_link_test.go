//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

// categoriaCriada cria uma categoria e devolve o id dela.
func categoriaCriada(t *testing.T, token, nome, tipo string) int64 {
	t.Helper()

	res := criarCategoria(t, token, map[string]any{"name": nome, "kind": tipo, "color": "#aabbcc"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("criar categoria: %d %v", res.StatusCode, res.Body)
	}

	id, _ := res.Body["id"].(float64)
	return int64(id)
}

func TestLancamentoGuardaACategoriaEscolhida(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)
	categoria := categoriaCriada(t, token, "Mercado", "despesa")

	res := criarLancamento(t, token, map[string]any{
		"description": "Feira", "amount_cents": 8550, "kind": "despesa",
		"occurred_at": "2026-09-10", "category_id": categoria,
	})
	assertStatus(t, res, http.StatusCreated)

	if res.String("category_name") != "Mercado" {
		t.Errorf("category_name = %q, esperava Mercado", res.String("category_name"))
	}
	if res.String("category_color") != "#aabbcc" {
		t.Errorf("category_color = %q, esperava #aabbcc", res.String("category_color"))
	}
}

func TestListaTrazNomeECorDaCategoria(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)
	categoria := categoriaCriada(t, token, "Transporte", "despesa")

	criarLancamento(t, token, map[string]any{
		"description": "Ônibus", "amount_cents": 500, "kind": "despesa",
		"occurred_at": "2026-09-10", "category_id": categoria,
	})

	_, lista := getList(t, "/api/transactions?year=2026&month=9", token)
	if len(lista) != 1 {
		t.Fatalf("esperava 1 lançamento, veio %d", len(lista))
	}

	if got := text(t, lista[0], "category_name"); got != "Transporte" {
		t.Errorf("categoria na lista = %q, esperava Transporte", got)
	}
}

func TestLancamentoSemCategoriaContinuaAceito(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	res := criarLancamento(t, token, map[string]any{
		"description": "Avulso", "amount_cents": 500, "kind": "despesa", "occurred_at": "2026-09-10",
	})

	assertStatus(t, res, http.StatusCreated)
	if res.Body["category_id"] != nil {
		t.Errorf("esperava nenhuma categoria, veio %v", res.Body["category_id"])
	}
}

func TestLancamentoRecusaCategoriaDeOutroTipo(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)
	receita := categoriaCriada(t, token, "Salário", "receita")

	res := criarLancamento(t, token, map[string]any{
		"amount_cents": 500, "kind": "despesa", "occurred_at": "2026-09-10", "category_id": receita,
	})

	assertStatus(t, res, http.StatusUnprocessableEntity)
	if res.Field("category_id") == "" {
		t.Errorf("esperava erro no campo category_id, veio %v", res.Body)
	}
}

func TestLancamentoRecusaCategoriaDeOutroUsuario(t *testing.T) {
	resetDatabase(t)

	tokenA := contaComToken(t)
	tokenB := contaComToken(t)
	categoriaDeA := categoriaCriada(t, tokenA, "Mercado", "despesa")

	// Para B, a categoria de A simplesmente não existe.
	res := criarLancamento(t, tokenB, map[string]any{
		"amount_cents": 500, "kind": "despesa", "occurred_at": "2026-09-10", "category_id": categoriaDeA,
	})

	assertStatus(t, res, http.StatusUnprocessableEntity)
	if res.Field("category_id") == "" {
		t.Errorf("esperava erro no campo category_id, veio %v", res.Body)
	}
}

func TestFixoGuardaACategoriaEProjetaComEla(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)
	categoria := categoriaCriada(t, token, "Academia", "despesa")

	criado := criarFixo(t, token, map[string]any{
		"description": "Academia", "amount_cents": 15990, "kind": "despesa",
		"frequency": "mensal", "start_date": "2026-01-20", "category_id": categoria,
	})
	assertStatus(t, criado, http.StatusCreated)

	if criado.String("category_name") != "Academia" {
		t.Errorf("category_name = %q, esperava Academia", criado.String("category_name"))
	}

	// A projeção no mês precisa carregar a mesma categoria.
	_, lista := getList(t, "/api/transactions?year=2026&month=9", token)
	if len(lista) != 1 {
		t.Fatalf("esperava 1 projeção, veio %d", len(lista))
	}
	if got := text(t, lista[0], "category_name"); got != "Academia" {
		t.Errorf("categoria da projeção = %q, esperava Academia", got)
	}
}

func TestApagarCategoriaDeixaOLancamentoSemEla(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)
	categoria := categoriaCriada(t, token, "Mercado", "despesa")

	criarLancamento(t, token, map[string]any{
		"description": "Feira", "amount_cents": 8550, "kind": "despesa",
		"occurred_at": "2026-09-10", "category_id": categoria,
	})

	assertStatus(t, del(t, "/api/categories/"+itoa(categoria), token), http.StatusNoContent)

	// ON DELETE SET NULL: o lançamento fica, sem categoria.
	_, lista := getList(t, "/api/transactions?year=2026&month=9", token)
	if len(lista) != 1 {
		t.Fatalf("o lançamento não deveria sumir, veio %d", len(lista))
	}
	if lista[0]["category_id"] != nil {
		t.Errorf("esperava o lançamento sem categoria, veio %v", lista[0]["category_id"])
	}
}

func TestPainelAgrupaGastosPorCategoria(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	mercado := categoriaCriada(t, token, "Mercado", "despesa")
	transporte := categoriaCriada(t, token, "Transporte", "despesa")

	criarLancamento(t, token, map[string]any{"amount_cents": 30000, "kind": "despesa", "occurred_at": "2026-09-02", "category_id": mercado})
	criarLancamento(t, token, map[string]any{"amount_cents": 50000, "kind": "despesa", "occurred_at": "2026-09-03", "category_id": mercado})
	criarLancamento(t, token, map[string]any{"amount_cents": 20000, "kind": "despesa", "occurred_at": "2026-09-04", "category_id": transporte})
	criarLancamento(t, token, map[string]any{"amount_cents": 10000, "kind": "despesa", "occurred_at": "2026-09-05"})

	res := get(t, "/api/dashboard?year=2026&month=9", token)
	assertStatus(t, res, http.StatusOK)

	grupos, ok := res.Body["gastos_por_categoria"].([]any)
	if !ok {
		t.Fatalf("esperava os gastos por categoria, veio %v", res.Body)
	}
	if len(grupos) != 3 {
		t.Fatalf("esperava 3 grupos, veio %d", len(grupos))
	}

	primeiro, _ := grupos[0].(map[string]any)
	if text(t, primeiro, "label") != "Mercado" || number(t, primeiro, "total_cents") != 80000 {
		t.Errorf("primeiro grupo = %v, esperava Mercado com 80000", primeiro)
	}

	ultimo, _ := grupos[2].(map[string]any)
	if text(t, ultimo, "label") != "Sem categoria" {
		t.Errorf("último grupo = %q, esperava Sem categoria", text(t, ultimo, "label"))
	}
}

func itoa(value int64) string {
	if value == 0 {
		return "0"
	}

	var digits []byte
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}

	return string(digits)
}
