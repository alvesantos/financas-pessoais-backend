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
	Description string `json:"description"`
	AmountCents int64  `json:"amount_cents"`
	Kind        string `json:"kind"`
	OccurredAt  string `json:"occurred_at"`
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

	return domain.NewTransaction{
		UserID:      userID,
		Description: strings.TrimSpace(r.Description),
		AmountCents: r.AmountCents,
		Kind:        domain.Kind(r.Kind),
		OccurredAt:  occurredAt,
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
}

func NewTransactionResponse(t domain.Transaction) TransactionResponse {
	response := TransactionResponse{
		ID:          t.ID,
		Description: t.Description,
		AmountCents: t.AmountCents,
		SignedCents: t.SignedAmount(),
		Kind:        string(t.Kind),
		KindLabel:   t.Kind.Label(),
		OccurredAt:  t.OccurredAt.Format(dateLayout),
		Projected:   t.IsProjected(),
		RecurringID: t.RecurringID,
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
