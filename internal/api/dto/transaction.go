package dto

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// dateLayout é o formato de data trocado com o cliente.
const dateLayout = "2006-01-02"

const maxDescriptionLength = 120

// CreateTransactionRequest é o corpo de POST /api/transactions.
// A descrição é opcional: em branco, vira o nome do tipo.
type CreateTransactionRequest struct {
	Description  string `json:"description"`
	AmountCents  int64  `json:"amount_cents"`
	Kind         string `json:"kind"`
	OccurredAt   string `json:"occurred_at"`
	CategoryID   *int64 `json:"category_id"`
	Paid         *bool  `json:"paid"`
	CreditCardID *int64 `json:"credit_card_id"`
	Invoice      string `json:"invoice"`
}

func (r CreateTransactionRequest) Validate() error {
	fields := map[string]string{}

	if r.AmountCents <= 0 {
		fields["amount_cents"] = "informe um valor maior que zero"
	}
	if !domain.Kind(r.Kind).Valid() {
		fields["kind"] = "escolha um tipo de lançamento"
	}
	if utf8.RuneCountInString(strings.TrimSpace(r.Description)) > maxDescriptionLength {
		fields["description"] = "a descrição passou de 120 caracteres"
	}
	if _, err := parseDate(r.OccurredAt); err != nil {
		fields["occurred_at"] = "informe uma data válida"
	}

	if len(fields) > 0 {
		return domain.ErrValidation.WithFields(fields)
	}
	return nil
}

func (r CreateTransactionRequest) ToDomain(userID int64) domain.NewTransaction {
	occurredAt, _ := parseDate(r.OccurredAt)

	// Sem marcação explícita, vale a data: o que já passou conta como pago.
	paid := !occurredAt.After(domain.Day(time.Now().UTC()))
	if r.Paid != nil {
		paid = *r.Paid
	}

	invoice := domain.InvoiceChoice(r.Invoice)
	if invoice == "" {
		invoice = domain.InvoiceCurrent
	}

	return domain.NewTransaction{
		UserID:       userID,
		Description:  strings.TrimSpace(r.Description),
		AmountCents:  r.AmountCents,
		Kind:         domain.Kind(r.Kind),
		OccurredAt:   occurredAt,
		CategoryID:   r.CategoryID,
		Paid:         paid,
		CreditCardID: r.CreditCardID,
		Invoice:      invoice,
	}
}

// PayOccurrenceRequest é o corpo de POST /api/transactions/occurrence.
type PayOccurrenceRequest struct {
	Origin     string `json:"origin"`
	OriginID   int64  `json:"origin_id"`
	OccurredAt string `json:"occurred_at"`
}

func (r PayOccurrenceRequest) Validate() error {
	fields := map[string]string{}

	if !domain.OccurrenceOrigin(r.Origin).Valid() {
		fields["origin"] = "origem desconhecida"
	}
	if r.OriginID <= 0 {
		fields["origin_id"] = "identificador inválido"
	}
	if _, err := parseDate(r.OccurredAt); err != nil {
		fields["occurred_at"] = "informe uma data válida"
	}

	if len(fields) > 0 {
		return domain.ErrValidation.WithFields(fields)
	}
	return nil
}

func (r PayOccurrenceRequest) ToDomain(userID int64) domain.PayOccurrence {
	occurredAt, _ := parseDate(r.OccurredAt)

	return domain.PayOccurrence{
		UserID:     userID,
		Origin:     domain.OccurrenceOrigin(r.Origin),
		OriginID:   r.OriginID,
		OccurredAt: occurredAt,
	}
}

// TransactionResponse é um lançamento como o cliente o vê. Projeções de
// fixos vêm com recurring_id preenchido e sem id próprio.
type TransactionResponse struct {
	ID             int64   `json:"id"`
	Description    string  `json:"description"`
	AmountCents    int64   `json:"amount_cents"`
	SignedCents    int64   `json:"signed_cents"`
	Kind           string  `json:"kind"`
	KindLabel      string  `json:"kind_label"`
	OccurredAt     string  `json:"occurred_at"`
	Projected      bool    `json:"projected"`
	RecurringID    *int64  `json:"recurring_id,omitempty"`
	Frequency      *string `json:"frequency,omitempty"`
	FrequencyLabel *string `json:"frequency_label,omitempty"`

	CategoryID    *int64  `json:"category_id"`
	CategoryName  *string `json:"category_name"`
	CategoryColor *string `json:"category_color"`

	Paid           bool    `json:"paid"`
	CreditCardID   *int64  `json:"credit_card_id"`
	CreditCardName *string `json:"credit_card_name"`
	InvoiceMonth   *string `json:"invoice_month"`

	DebtID            *int64 `json:"debt_id,omitempty"`
	InstallmentNumber *int   `json:"installment_number,omitempty"`
	InstallmentsTotal *int   `json:"installments_total,omitempty"`
	RecurringOrigin   *int64 `json:"recurring_origin_id,omitempty"`
}

func NewTransactionResponse(t domain.Transaction) TransactionResponse {
	response := TransactionResponse{
		ID:                t.ID,
		Description:       t.Description,
		AmountCents:       t.AmountCents,
		SignedCents:       t.SignedAmount(),
		Kind:              string(t.Kind),
		KindLabel:         t.Kind.Label(),
		OccurredAt:        t.OccurredAt.Format(dateLayout),
		Projected:         t.IsProjected(),
		RecurringID:       t.RecurringID,
		CategoryID:        t.CategoryID,
		CategoryName:      t.CategoryName,
		CategoryColor:     t.CategoryColor,
		DebtID:            t.DebtID,
		RecurringOrigin:   t.RecurringID,
		InstallmentNumber: t.InstallmentNumber,
		InstallmentsTotal: t.InstallmentsTotal,
		Paid:              t.Paid,
		CreditCardID:      t.CreditCardID,
		CreditCardName:    t.CreditCardName,
	}

	if t.InvoiceMonth != nil {
		invoice := t.InvoiceMonth.Format(dateLayout)
		response.InvoiceMonth = &invoice
	}

	if t.Frequency != nil {
		frequency := string(*t.Frequency)
		label := t.Frequency.Label()
		response.Frequency = &frequency
		response.FrequencyLabel = &label
	}

	return response
}

func NewTransactionListResponse(entries []domain.Transaction) []TransactionResponse {
	list := make([]TransactionResponse, 0, len(entries))
	for _, entry := range entries {
		list = append(list, NewTransactionResponse(entry))
	}
	return list
}

// MonthSummaryResponse são os saldos do mês.
type MonthSummaryResponse struct {
	Year          int   `json:"year"`
	Month         int   `json:"month"`
	SaldoAtual    int64 `json:"saldo_atual_cents"`
	SaldoPrevisto int64 `json:"saldo_previsto_cents"`
	Receitas      int64 `json:"receitas_cents"`
	Despesas      int64 `json:"despesas_cents"`
	Quantidade    int   `json:"quantidade"`
}

func NewMonthSummaryResponse(s domain.MonthSummary) MonthSummaryResponse {
	return MonthSummaryResponse{
		Year:          s.Year,
		Month:         int(s.Month),
		SaldoAtual:    s.SaldoAtual,
		SaldoPrevisto: s.SaldoPrevisto,
		Receitas:      s.Receitas,
		Despesas:      s.Despesas,
		Quantidade:    s.QuantidadeItens,
	}
}

func parseDate(value string) (time.Time, error) {
	return time.ParseInLocation(dateLayout, strings.TrimSpace(value), time.UTC)
}
