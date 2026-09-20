package domain

import "time"

// MonthTotals são os totais de um mês do ano, para o gráfico anual.
type MonthTotals struct {
	Month    time.Month
	Receitas int64
	Despesas int64
	Saldo    int64
}

// KindTotal é quanto foi gasto em um tipo, para o gráfico de composição.
type KindTotal struct {
	Kind  Kind
	Label string
	Total int64
}

// YearTotals são os números do ano inteiro.
type YearTotals struct {
	Year     int
	Receitas int64
	Despesas int64
	Saldo    int64
}

// Dashboard reúne as métricas do painel: o ano, o mês escolhido, a série
// mensal do gráfico e a composição dos gastos do mês.
type Dashboard struct {
	Year          YearTotals
	Month         MonthSummary
	PorMes        []MonthTotals
	GastosPorTipo []KindTotal
	MaiorGasto    *Transaction
}
