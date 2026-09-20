package domain

import (
	"strconv"
	"time"
)

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

	// Paid diz se o dinheiro já saiu ou entrou de fato. É o que decide o
	// saldo atual: uma conta lançada para o dia 25 e ainda não paga pesa no
	// previsto, nunca no atual.
	Paid bool

	// Cartão e fatura, quando o lançamento é de cartão de crédito.
	CreditCardID   *int64
	CreditCardName *string
	InvoiceMonth   *time.Time

	// Categoria, quando houver. Nome e cor vêm junto para a listagem não
	// precisar de uma consulta por linha.
	CategoryID    *int64
	CategoryName  *string
	CategoryColor *string

	// Projected diz que o lançamento foi calculado na leitura em vez de lido
	// do banco. Uma ocorrência materializada carrega a origem abaixo e
	// mesmo assim não é projeção: ela virou linha.
	Projected bool

	// Origem, quando o lançamento vem de um fixo.
	RecurringID *int64
	Frequency   *Frequency

	// Origem, quando o lançamento é parcela de uma dívida.
	DebtID            *int64
	InstallmentNumber *int
	InstallmentsTotal *int
}

// IsProjected diz se o lançamento foi calculado em vez de lido do banco.
func (t Transaction) IsProjected() bool {
	return t.Projected
}

// OccurrenceKey identifica uma ocorrência pela origem e pela data, que é
// como a projeção descobre que aquela data já virou linha.
func (t Transaction) OccurrenceKey() (string, bool) {
	if t.RecurringID != nil {
		return "fixo:" + strconv.FormatInt(*t.RecurringID, 10) + ":" + t.OccurredAt.Format("2006-01-02"), true
	}
	if t.DebtID != nil && t.InstallmentNumber != nil {
		return "divida:" + strconv.FormatInt(*t.DebtID, 10) + ":" + strconv.Itoa(*t.InstallmentNumber), true
	}

	return "", false
}

// SignedAmount é o efeito do lançamento sobre o saldo.
func (t Transaction) SignedAmount() int64 {
	return t.Kind.Signed(t.AmountCents)
}

// NewTransaction são os dados para criar um lançamento.
type NewTransaction struct {
	UserID       int64
	Description  string
	AmountCents  int64
	Kind         Kind
	OccurredAt   time.Time
	CategoryID   *int64
	Paid         bool
	CreditCardID *int64
	Invoice      InvoiceChoice

	// Origem, quando o lançamento materializa uma ocorrência.
	RecurringID       *int64
	DebtID            *int64
	InstallmentNumber *int
	// InvoiceMonth é calculado pelo serviço a partir do cartão e da escolha
	// de fatura, para o repositório não precisar conhecer essa regra.
	InvoiceMonth *time.Time
}

// OccurrenceOrigin diz de onde a ocorrência veio.
type OccurrenceOrigin string

const (
	OriginRecurring OccurrenceOrigin = "fixo"
	OriginDebt      OccurrenceOrigin = "divida"
)

func (o OccurrenceOrigin) Valid() bool {
	return o == OriginRecurring || o == OriginDebt
}

// PayOccurrence são os dados para transformar uma ocorrência projetada em
// lançamento pago.
type PayOccurrence struct {
	UserID     int64
	Origin     OccurrenceOrigin
	OriginID   int64
	OccurredAt time.Time
}

var (
	ErrOccurrenceNotFound    = NewError(CodeNotFound, "ocorrência não encontrada nessa data")
	ErrOccurrenceAlreadyPaid = NewError(CodeConflict, "esta ocorrência já foi marcada como paga")
)

// UpdateTransaction são os dados para reescrever um lançamento.
type UpdateTransaction struct {
	ID int64
	NewTransaction
}

// MonthSummary são os totais de um mês.
//
// SaldoAtual conta só o que foi pago ou recebido; SaldoPrevisto conta o mês
// inteiro, incluindo o que ainda vai cair e as projeções.
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
