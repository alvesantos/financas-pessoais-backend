package dto

import "github.com/alvesantos/financas-backend/internal/domain"

// MonthTotalsResponse é um ponto da série mensal do gráfico anual.
type MonthTotalsResponse struct {
	Month    int   `json:"month"`
	Receitas int64 `json:"receitas_cents"`
	Despesas int64 `json:"despesas_cents"`
	Saldo    int64 `json:"saldo_cents"`
}

// KindTotalResponse é uma barra do gráfico de composição dos gastos.
type KindTotalResponse struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	Total int64  `json:"total_cents"`
}

// YearTotalsResponse são os números do ano inteiro.
type YearTotalsResponse struct {
	Year     int   `json:"year"`
	Receitas int64 `json:"receitas_cents"`
	Despesas int64 `json:"despesas_cents"`
	Saldo    int64 `json:"saldo_cents"`
}

// DashboardResponse é o painel completo.
type DashboardResponse struct {
	SaldoAtual    int64                 `json:"saldo_atual_cents"`
	DespesasFixas int64                 `json:"despesas_fixas_cents"`
	Year          YearTotalsResponse    `json:"year"`
	Month         MonthSummaryResponse  `json:"month"`
	PorMes        []MonthTotalsResponse `json:"por_mes"`
	GastosPorTipo []KindTotalResponse   `json:"gastos_por_tipo"`
	MaiorGasto    *TransactionResponse  `json:"maior_gasto"`
}

func NewDashboardResponse(d domain.Dashboard) DashboardResponse {
	response := DashboardResponse{
		SaldoAtual:    d.SaldoAtual,
		DespesasFixas: d.DespesasFixas,
		Year: YearTotalsResponse{
			Year:     d.Year.Year,
			Receitas: d.Year.Receitas,
			Despesas: d.Year.Despesas,
			Saldo:    d.Year.Saldo,
		},
		Month:         NewMonthSummaryResponse(d.Month),
		PorMes:        make([]MonthTotalsResponse, 0, len(d.PorMes)),
		GastosPorTipo: make([]KindTotalResponse, 0, len(d.GastosPorTipo)),
	}

	for _, month := range d.PorMes {
		response.PorMes = append(response.PorMes, MonthTotalsResponse{
			Month:    int(month.Month),
			Receitas: month.Receitas,
			Despesas: month.Despesas,
			Saldo:    month.Saldo,
		})
	}

	for _, kind := range d.GastosPorTipo {
		response.GastosPorTipo = append(response.GastosPorTipo, KindTotalResponse{
			Kind:  string(kind.Kind),
			Label: kind.Label,
			Total: kind.Total,
		})
	}

	if d.MaiorGasto != nil {
		biggest := NewTransactionResponse(*d.MaiorGasto)
		response.MaiorGasto = &biggest
	}

	return response
}
