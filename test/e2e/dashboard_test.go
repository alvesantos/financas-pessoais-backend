//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

func TestPainelSomaOAnoEOMes(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarLancamento(t, token, map[string]any{"amount_cents": 500000, "kind": "receita", "occurred_at": "2026-01-05"})
	criarLancamento(t, token, map[string]any{"amount_cents": 500000, "kind": "receita", "occurred_at": "2026-09-05"})
	criarLancamento(t, token, map[string]any{"amount_cents": 80000, "kind": "despesa", "occurred_at": "2026-09-06"})

	res := get(t, "/api/dashboard?year=2026&month=9", token)
	assertStatus(t, res, http.StatusOK)

	ano, ok := res.Body["year"].(map[string]any)
	if !ok {
		t.Fatalf("esperava o bloco do ano, veio %v", res.Body)
	}
	if got := number(t, ano, "receitas_cents"); got != 1000000 {
		t.Errorf("receitas do ano = %d, esperava 1000000", got)
	}
	if got := number(t, ano, "saldo_cents"); got != 920000 {
		t.Errorf("saldo do ano = %d, esperava 920000", got)
	}

	mes, _ := res.Body["month"].(map[string]any)
	if got := number(t, mes, "saldo_previsto_cents"); got != 420000 {
		t.Errorf("saldo previsto do mês = %d, esperava 420000", got)
	}
}

func TestPainelTrazDozeMesesParaOGrafico(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	res := get(t, "/api/dashboard?year=2026&month=9", token)
	assertStatus(t, res, http.StatusOK)

	porMes, ok := res.Body["por_mes"].([]any)
	if !ok {
		t.Fatalf("esperava a série mensal, veio %v", res.Body)
	}
	if len(porMes) != 12 {
		t.Errorf("esperava 12 meses, veio %d", len(porMes))
	}
}

func TestPainelOrdenaOsGastosPorTipo(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarLancamento(t, token, map[string]any{"amount_cents": 20000, "kind": "investimento", "occurred_at": "2026-09-02"})
	criarLancamento(t, token, map[string]any{"amount_cents": 90000, "kind": "despesa", "occurred_at": "2026-09-03"})
	criarLancamento(t, token, map[string]any{"amount_cents": 50000, "kind": "cartao_credito", "occurred_at": "2026-09-04"})
	criarLancamento(t, token, map[string]any{"amount_cents": 800000, "kind": "receita", "occurred_at": "2026-09-05"})

	res := get(t, "/api/dashboard?year=2026&month=9", token)

	gastos, ok := res.Body["gastos_por_tipo"].([]any)
	if !ok {
		t.Fatalf("esperava os gastos por tipo, veio %v", res.Body)
	}

	esperado := []string{"despesa", "cartao_credito", "investimento"}
	if len(gastos) != len(esperado) {
		t.Fatalf("esperava %d tipos, veio %d", len(esperado), len(gastos))
	}

	for i, tipo := range esperado {
		item, _ := gastos[i].(map[string]any)
		if got := text(t, item, "kind"); got != tipo {
			t.Errorf("posição %d: tipo = %q, esperava %q", i, got, tipo)
		}
	}
}

func TestPainelContaOsFixosProjetados(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarFixo(t, token, map[string]any{
		"description": "Academia", "amount_cents": 15990, "kind": "despesa",
		"frequency": "mensal", "start_date": "2026-01-20",
	})

	res := get(t, "/api/dashboard?year=2026&month=9", token)
	assertStatus(t, res, http.StatusOK)

	// Doze meses de academia: o painel e a tela de lançamentos contam igual.
	ano, _ := res.Body["year"].(map[string]any)
	if got := number(t, ano, "despesas_cents"); got != 15990*12 {
		t.Errorf("despesas do ano = %d, esperava %d", got, 15990*12)
	}
}

func TestPainelDeUsuarioNovoVemZerado(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	res := get(t, "/api/dashboard?year=2026&month=9", token)
	assertStatus(t, res, http.StatusOK)

	ano, _ := res.Body["year"].(map[string]any)
	if got := number(t, ano, "saldo_cents"); got != 0 {
		t.Errorf("saldo do ano = %d, esperava 0", got)
	}
	if res.Body["maior_gasto"] != nil {
		t.Errorf("esperava nenhum maior gasto, veio %v", res.Body["maior_gasto"])
	}
}

func TestPainelRecusaMesInvalido(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	res := get(t, "/api/dashboard?year=2026&month=13", token)

	assertStatus(t, res, http.StatusUnprocessableEntity)
	if res.Field("month") == "" {
		t.Errorf("esperava erro no campo month, veio %v", res.Body)
	}
}

func TestPainelTrazSaldoAtualAcumulado(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	// Datas passadas, em anos diferentes: o saldo atual não é do ano.
	criarLancamento(t, token, map[string]any{"amount_cents": 300000, "kind": "receita", "occurred_at": "2024-05-05"})
	criarLancamento(t, token, map[string]any{"amount_cents": 100000, "kind": "despesa", "occurred_at": "2025-03-10"})

	res := get(t, "/api/dashboard?year=2026&month=9", token)
	assertStatus(t, res, http.StatusOK)

	if got := number(t, res.Body, "saldo_atual_cents"); got != 200000 {
		t.Errorf("saldo atual = %d, esperava 200000", got)
	}
}

func TestSaldoAtualNaoContaLancamentoFuturo(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarLancamento(t, token, map[string]any{"amount_cents": 500000, "kind": "receita", "occurred_at": "2024-01-05"})
	// Bem no futuro: não pode entrar no saldo atual.
	criarLancamento(t, token, map[string]any{"amount_cents": 400000, "kind": "despesa", "occurred_at": "2099-01-05"})

	res := get(t, "/api/dashboard?year=2026&month=9", token)

	if got := number(t, res.Body, "saldo_atual_cents"); got != 500000 {
		t.Errorf("saldo atual = %d, esperava 500000", got)
	}
}

func TestPainelTrazDespesasFixasDoMes(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarFixo(t, token, map[string]any{
		"description": "Academia", "amount_cents": 15990, "kind": "despesa",
		"frequency": "mensal", "start_date": "2026-01-20",
	})
	criarFixo(t, token, map[string]any{
		"description": "Aluguel", "amount_cents": 200000, "kind": "despesa",
		"frequency": "mensal", "start_date": "2026-01-05",
	})
	// Receita fixa não entra no custo de vida.
	criarFixo(t, token, map[string]any{
		"description": "Salário", "amount_cents": 500000, "kind": "receita",
		"frequency": "mensal", "start_date": "2026-01-05",
	})

	res := get(t, "/api/dashboard?year=2026&month=9", token)

	if got := number(t, res.Body, "despesas_fixas_cents"); got != 215990 {
		t.Errorf("despesas fixas = %d, esperava 215990", got)
	}
}
