package domain

import "time"

// Transaction é um lançamento. O valor é sempre positivo e em centavos; o
// efeito sobre o saldo vem do tipo.
type Transaction struct {
	ID          int64
	UserID      int64
	Description string
	AmountCents int64
	Kind        Kind
	OccurredAt  time.Time
	CreatedAt   time.Time

	// Categoria, quando houver. Nome e cor vêm junto para a listagem não
	// precisar de uma consulta por linha.
	CategoryID    *int64
	CategoryName  *string
	CategoryColor *string

	// Preenchidos quando o lançamento foi projetado de um fixo. Projeções
	// não existem como linha no banco e não podem ser apagadas isoladamente.
	RecurringID *int64
	Frequency   *Frequency
}

// IsProjected diz se o lançamento veio de um fixo em vez do banco.
func (t Transaction) IsProjected() bool {
	return t.RecurringID != nil
}

// SignedAmount é o efeito do lançamento sobre o saldo.
func (t Transaction) SignedAmount() int64 {
	return t.Kind.Signed(t.AmountCents)
}

// NewTransaction são os dados para criar um lançamento.
type NewTransaction struct {
	UserID      int64
	Description string
	AmountCents int64
	Kind        Kind
	OccurredAt  time.Time
	CategoryID  *int64
}

// MonthSummary são os totais de um mês.
//
// SaldoAtual conta só o que já aconteceu (até hoje); SaldoPrevisto conta o
// mês inteiro, incluindo o que ainda vai cair e as projeções dos fixos.
type MonthSummary struct {
	Year            int
	Month           time.Month
	SaldoAtual      int64
	SaldoPrevisto   int64
	Receitas        int64
	Despesas        int64
	TotalPorTipo    map[Kind]int64
	QuantidadeItens int
}

// Period é um intervalo fechado de datas.
type Period struct {
	From time.Time
	To   time.Time
}

// MonthPeriod devolve o primeiro e o último dia do mês, em UTC.
func MonthPeriod(year int, month time.Month) Period {
	from := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	return Period{From: from, To: from.AddDate(0, 1, -1)}
}

// YearPeriod devolve o primeiro e o último dia do ano, em UTC.
func YearPeriod(year int) Period {
	from := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	return Period{From: from, To: time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC)}
}

// Contains diz se a data cai dentro do período (limites incluídos).
func (p Period) Contains(date time.Time) bool {
	return !date.Before(p.From) && !date.After(p.To)
}

// Day normaliza um instante para a meia-noite UTC daquele dia, que é como
// as datas são comparadas e gravadas.
func Day(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
