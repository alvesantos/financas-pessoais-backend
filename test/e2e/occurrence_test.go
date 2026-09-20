//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

func pagarOcorrencia(t *testing.T, token string, corpo map[string]any) apiResponse {
	t.Helper()
	return postAuth(t, "/api/transactions/occurrence", corpo, token)
}

func TestMarcarParcelaDeFixoComoPaga(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criado := criarFixo(t, token, map[string]any{
		"description": "Academia", "amount_cents": 15990, "kind": "despesa",
		"frequency": "mensal", "start_date": "2026-01-20",
	})
	id, _ := criado.Body["id"].(float64)

	res := pagarOcorrencia(t, token, map[string]any{
		"origin": "fixo", "origin_id": int64(id), "occurred_at": "2026-09-20",
	})
	assertStatus(t, res, http.StatusCreated)

	if pago, _ := res.Body["paid"].(bool); !pago {
		t.Error("a ocorrência marcada precisa vir paga")
	}

	// A projeção daquela data some, em vez de duplicar.
	_, lista := getList(t, "/api/transactions?year=2026&month=9", token)
	if len(lista) != 1 {
		t.Fatalf("esperava 1 lançamento, veio %d", len(lista))
	}
	if projetado, _ := lista[0]["projected"].(bool); projetado {
		t.Error("depois de marcada, a ocorrência é linha de verdade")
	}
}

func TestMarcarParcelaDeDividaComoPagaNaAPI(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criada := criarDivida(t, token, emprestimo())
	id, _ := criada.Body["id"].(float64)

	res := pagarOcorrencia(t, token, map[string]any{
		"origin": "divida", "origin_id": int64(id), "occurred_at": "2026-12-07",
	})
	assertStatus(t, res, http.StatusCreated)

	if got := number(t, res.Body, "installment_number"); got != 3 {
		t.Errorf("parcela = %d, esperava 3", got)
	}

	_, lista := getList(t, "/api/transactions?year=2026&month=12", token)
	if len(lista) != 1 {
		t.Fatalf("esperava 1 lançamento, veio %d", len(lista))
	}
}

func TestMarcarAMesmaOcorrenciaDuasVezes(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criado := criarFixo(t, token, map[string]any{
		"description": "Academia", "amount_cents": 15990, "kind": "despesa",
		"frequency": "mensal", "start_date": "2026-01-20",
	})
	id, _ := criado.Body["id"].(float64)

	corpo := map[string]any{"origin": "fixo", "origin_id": int64(id), "occurred_at": "2026-09-20"}

	assertStatus(t, pagarOcorrencia(t, token, corpo), http.StatusCreated)
	assertStatus(t, pagarOcorrencia(t, token, corpo), http.StatusConflict)
}

func TestMarcarDataQueNaoEhOcorrencia(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criado := criarFixo(t, token, map[string]any{
		"description": "Academia", "amount_cents": 15990, "kind": "despesa",
		"frequency": "mensal", "start_date": "2026-01-20",
	})
	id, _ := criado.Body["id"].(float64)

	// O fixo cai no dia 20, não no 21.
	res := pagarOcorrencia(t, token, map[string]any{
		"origin": "fixo", "origin_id": int64(id), "occurred_at": "2026-09-21",
	})

	assertStatus(t, res, http.StatusNotFound)
}

func TestMarcarOcorrenciaDeOutroUsuario(t *testing.T) {
	resetDatabase(t)

	tokenA := contaComToken(t)
	tokenB := contaComToken(t)

	criado := criarFixo(t, tokenA, map[string]any{
		"description": "Academia", "amount_cents": 15990, "kind": "despesa",
		"frequency": "mensal", "start_date": "2026-01-20",
	})
	id, _ := criado.Body["id"].(float64)

	res := pagarOcorrencia(t, tokenB, map[string]any{
		"origin": "fixo", "origin_id": int64(id), "occurred_at": "2026-09-20",
	})

	assertStatus(t, res, http.StatusNotFound)
}

func TestDespesasFixasSomamTodoOCustoDeVida(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	// Começa mês que vem: o compromisso já existe hoje.
	criarFixo(t, token, map[string]any{
		"description": "Internet", "amount_cents": 15000, "kind": "despesa",
		"frequency": "mensal", "start_date": "2099-01-15",
	})
	criarFixo(t, token, map[string]any{
		"description": "Psicólogo", "amount_cents": 7000, "kind": "despesa",
		"frequency": "semanal", "start_date": "2099-01-09",
	})

	res := get(t, "/api/dashboard?year=2026&month=9", token)
	assertStatus(t, res, http.StatusOK)

	// 15000 mensal mais 7000 semanais espalhados pelo ano.
	esperado := int64(15000 + 7000*52/12)
	if got := number(t, res.Body, "despesas_fixas_cents"); got != esperado {
		t.Errorf("despesas fixas = %d, esperava %d", got, esperado)
	}
}
