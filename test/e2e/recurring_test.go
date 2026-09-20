//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"
)

func criarFixo(t *testing.T, token string, corpo map[string]any) apiResponse {
	t.Helper()
	return postAuth(t, "/api/recurring", corpo, token)
}

func TestFixoApareceNosLancamentosDoMes(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	// "Academia, todo dia 20, R$ 159,90", o caso do enunciado.
	criado := criarFixo(t, token, map[string]any{
		"description":  "Academia",
		"amount_cents": 15990,
		"kind":         "despesa",
		"frequency":    "mensal",
		"start_date":   "2026-01-20",
	})
	assertStatus(t, criado, http.StatusCreated)

	_, lista := getList(t, "/api/transactions?year=2026&month=9", token)

	if len(lista) != 1 {
		t.Fatalf("esperava o fixo projetado em setembro, veio %d", len(lista))
	}

	item := lista[0]
	if got := text(t, item, "occurred_at"); got != "2026-09-20" {
		t.Errorf("data = %q, esperava 2026-09-20", got)
	}
	if got := number(t, item, "amount_cents"); got != 15990 {
		t.Errorf("valor = %d, esperava 15990", got)
	}
	if got := text(t, item, "description"); got != "Academia" {
		t.Errorf("descrição = %q, esperava Academia", got)
	}
	if projetado, _ := item["projected"].(bool); !projetado {
		t.Error("o lançamento do fixo precisa vir marcado como projeção")
	}
	if got := text(t, item, "frequency_label"); got != "Mensal" {
		t.Errorf("frequência = %q, esperava Mensal", got)
	}
}

func TestFixoSeRepeteMesAMes(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarFixo(t, token, map[string]any{
		"description": "Academia", "amount_cents": 15990, "kind": "despesa",
		"frequency": "mensal", "start_date": "2026-01-20",
	})

	for _, mes := range []int{1, 5, 9, 12} {
		_, lista := getList(t, fmt.Sprintf("/api/transactions?year=2026&month=%d", mes), token)

		if len(lista) != 1 {
			t.Errorf("mês %d: esperava 1 projeção, veio %d", mes, len(lista))
		}
	}

	// Antes do início não aparece.
	_, anterior := getList(t, "/api/transactions?year=2025&month=12", token)
	if len(anterior) != 0 {
		t.Errorf("esperava nada antes do início, veio %d", len(anterior))
	}
}

func TestTodasAsFrequenciasDeFixo(t *testing.T) {
	// Começando em 01/01/2026, o ciclo não reinicia a cada mês: ele continua
	// contando desde o início. Em setembro isso dá 3, 10, 17 e 24 no semanal,
	// e 10 e 24 no quinzenal.
	casos := map[string]int{
		"diario":    30, // setembro tem 30 dias
		"semanal":   4,
		"quinzenal": 2,
		"mensal":    1,
		"semestral": 0, // cai em janeiro e julho
		"anual":     0, // cai só em janeiro
	}

	for frequencia, esperado := range casos {
		t.Run(frequencia, func(t *testing.T) {
			resetDatabase(t)
			token := contaComToken(t)

			criarFixo(t, token, map[string]any{
				"description": "Teste", "amount_cents": 1000, "kind": "despesa",
				"frequency": frequencia, "start_date": "2026-01-01",
			})

			_, lista := getList(t, "/api/transactions?year=2026&month=9", token)

			if len(lista) != esperado {
				t.Errorf("%s: esperava %d ocorrências em setembro, veio %d", frequencia, esperado, len(lista))
			}
		})
	}
}

func TestFixoSemDescricaoUsaONomeDoTipoNaAPI(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	res := criarFixo(t, token, map[string]any{
		"description": "", "amount_cents": 15990, "kind": "cartao_credito",
		"frequency": "mensal", "start_date": "2026-01-20",
	})

	assertStatus(t, res, http.StatusCreated)
	if res.String("description") != "Gasto no cartão de crédito" {
		t.Errorf("descrição = %q, esperava %q", res.String("description"), "Gasto no cartão de crédito")
	}
}

func TestFixoDeReceitaSomaNoSaldo(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarFixo(t, token, map[string]any{
		"description": "Salário", "amount_cents": 500000, "kind": "receita",
		"frequency": "mensal", "start_date": "2026-01-05",
	})

	res := get(t, "/api/transactions/summary?year=2026&month=9", token)
	assertStatus(t, res, http.StatusOK)

	previsto, _ := res.Body["saldo_previsto_cents"].(float64)
	if int64(previsto) != 500000 {
		t.Errorf("saldo previsto = %v, esperava 500000", previsto)
	}
}

func TestValidacaoDoFixo(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	casos := []struct {
		nome      string
		corpo     map[string]any
		campoErro string
	}{
		{
			"frequência desconhecida",
			map[string]any{"amount_cents": 1000, "kind": "despesa", "frequency": "bimestral", "start_date": "2026-01-20"},
			"frequency",
		},
		{
			"valor zero",
			map[string]any{"amount_cents": 0, "kind": "despesa", "frequency": "mensal", "start_date": "2026-01-20"},
			"amount_cents",
		},
		{
			"fim antes do início",
			map[string]any{"amount_cents": 1000, "kind": "despesa", "frequency": "mensal", "start_date": "2026-01-20", "end_date": "2025-12-01"},
			"end_date",
		},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			res := criarFixo(t, token, caso.corpo)

			assertStatus(t, res, http.StatusUnprocessableEntity)
			if res.Field(caso.campoErro) == "" {
				t.Errorf("esperava erro no campo %q, veio %v", caso.campoErro, res.Body)
			}
		})
	}
}

func TestApagarFixoRemoveAsProjecoes(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criado := criarFixo(t, token, map[string]any{
		"description": "Academia", "amount_cents": 15990, "kind": "despesa",
		"frequency": "mensal", "start_date": "2026-01-20",
	})
	id, _ := criado.Body["id"].(float64)

	assertStatus(t, del(t, fmt.Sprintf("/api/recurring/%d", int64(id)), token), http.StatusNoContent)

	_, lista := getList(t, "/api/transactions?year=2026&month=9", token)
	if len(lista) != 0 {
		t.Errorf("apagar o fixo deveria sumir com as projeções, veio %d", len(lista))
	}
}

func TestListaDeFixos(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarFixo(t, token, map[string]any{
		"description": "Academia", "amount_cents": 15990, "kind": "despesa",
		"frequency": "mensal", "start_date": "2026-01-20",
	})
	criarFixo(t, token, map[string]any{
		"description": "Salário", "amount_cents": 500000, "kind": "receita",
		"frequency": "mensal", "start_date": "2026-01-05",
	})

	status, lista := getList(t, "/api/recurring", token)
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}

	if len(lista) != 2 {
		t.Fatalf("esperava 2 fixos, veio %d", len(lista))
	}
}
