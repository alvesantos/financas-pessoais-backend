package domain

import "time"

// CreditCard é um cartão de crédito da pessoa.
type CreditCard struct {
	ID     int64
	UserID int64
	Name   string
	// LimitCents é o limite total do cartão, em centavos.
	LimitCents int64
	// BestPurchaseDay é o dia a partir do qual a compra já cai na fatura
	// seguinte, que é o que faz dele o melhor dia para comprar.
	BestPurchaseDay int
	DueDay          int
	CreatedAt       time.Time
}

// NewCreditCard são os dados para cadastrar um cartão.
type NewCreditCard struct {
	UserID          int64
	Name            string
	LimitCents      int64
	BestPurchaseDay int
	DueDay          int
}

// InvoiceChoice diz em qual fatura a compra entra.
type InvoiceChoice string

const (
	InvoiceCurrent InvoiceChoice = "atual"
	InvoiceNext    InvoiceChoice = "proxima"
)

func (c InvoiceChoice) Valid() bool {
	return c == InvoiceCurrent || c == InvoiceNext
}

// InvoiceMonthFor devolve o primeiro dia do mês da fatura em que a compra
// entra. Comprar no melhor dia ou depois dele joga a compra para a fatura
// seguinte, que é justamente a vantagem do melhor dia.
func (c CreditCard) InvoiceMonthFor(purchaseDate time.Time, choice InvoiceChoice) time.Time {
	date := Day(purchaseDate)
	month := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)

	if date.Day() >= c.BestPurchaseDay {
		month = month.AddDate(0, 1, 0)
	}

	if choice == InvoiceNext {
		month = month.AddDate(0, 1, 0)
	}

	return month
}

// DueDateFor é o vencimento da fatura do mês informado, com o dia preso ao
// último dia dos meses mais curtos.
func (c CreditCard) DueDateFor(invoiceMonth time.Time) time.Time {
	day := c.DueDay
	if last := daysInMonth(invoiceMonth.Year(), invoiceMonth.Month()); day > last {
		day = last
	}

	return time.Date(invoiceMonth.Year(), invoiceMonth.Month(), day, 0, 0, 0, 0, time.UTC)
}

var (
	ErrCreditCardNotFound = NewError(CodeNotFound, "cartão não encontrado")
	ErrCreditCardTaken    = NewError(CodeConflict, "já existe um cartão com esse nome")
)
