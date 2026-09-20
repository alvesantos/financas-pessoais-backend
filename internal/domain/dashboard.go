package domain

import "time"

// MonthTotals são os totais de um mês do ano, para o gráfico anual.
type MonthTotals struct {
	Month    time.Month
	Receitas int64
	Despesas int64
	Saldo    int64
}

// KindTotal é quanto foi gasto em um tipo.
type KindTotal struct {
	Kind  Kind
	Label string
	Total int64
}

// CategoryTotal é quanto foi gasto em uma categoria, para o gráfico de
// composição. CategoryID nulo é o balde de quem ainda não tem categoria.
type CategoryTotal struct {
	CategoryID *int64
	Label      string
	Color      string
	Total      int64
}

// YearTotals são os números do ano inteiro.
type YearTotals struct {
	Year     int
	Receitas int64
	Despesas int64
	Saldo    int64
}

// DebtsSummary agrega todas as dívidas: quanto já venceu e quanto falta.
type DebtsSummary struct {
	TotalCents     int64
	PaidCents      int64
	RemainingCents int64
	OpenCount      int
	SettledCount   int
	Percent        int
}

// Dashboard reúne as métricas do painel: o ano, o mês escolhido, a série
// mensal do gráfico e a composição dos gastos do mês.
type Dashboard struct {
	// SaldoAtual acumula tudo que já foi pago ou recebido, desde a primeira
	// movimentação. Não é recortado por mês nem por ano.
	SaldoAtual int64
	// DespesasFixas é o custo de vida do mês: o que os fixos de saída somam
	// no período, qualquer que seja a frequência de cada um.
	DespesasFixas int64

	Year               YearTotals
	Month              MonthSummary
	PorMes             []MonthTotals
	GastosPorTipo      []KindTotal
	GastosPorCategoria []CategoryTotal
	MaiorGasto         *Transaction
	Dividas            DebtsSummary
}
