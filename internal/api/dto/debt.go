package dto

import (
	"strings"
	"unicode/utf8"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// CreateDebtRequest é o corpo de POST /api/debts.
type CreateDebtRequest struct {
	Description      string `json:"description"`
	InstallmentCents int64  `json:"installment_amount_cents"`
	Installments     int    `json:"installments"`
	Kind             string `json:"kind"`
	Frequency        string `json:"frequency"`
	FirstDueDate     string `json:"first_due_date"`
	CategoryID       *int64 `json:"category_id"`
}

func (r CreateDebtRequest) Validate() error {
	fields := map[string]string{}

	if r.InstallmentCents <= 0 {
		fields["installment_amount_cents"] = "informe o valor da parcela"
	}
	if r.Installments <= 0 {
		fields["installments"] = "informe quantas parcelas são"
	}
	if !domain.Kind(r.Kind).Valid() {
		fields["kind"] = "escolha um tipo"
	}
	if !domain.Frequency(r.Frequency).Valid() {
		fields["frequency"] = "escolha uma frequência"
	}
	if utf8.RuneCountInString(strings.TrimSpace(r.Description)) > maxDescriptionLength {
		fields["description"] = "a descrição passou de 120 caracteres"
	}
	if _, err := parseDate(r.FirstDueDate); err != nil {
		fields["first_due_date"] = "informe uma data válida"
	}

	if len(fields) > 0 {
		return domain.ErrValidation.WithFields(fields)
	}
	return nil
}

func (r CreateDebtRequest) ToDomain(userID int64) domain.NewDebt {
	firstDueDate, _ := parseDate(r.FirstDueDate)

	return domain.NewDebt{
		UserID:           userID,
		Description:      strings.TrimSpace(r.Description),
		InstallmentCents: r.InstallmentCents,
		Installments:     r.Installments,
		Kind:             domain.Kind(r.Kind),
		Frequency:        domain.Frequency(r.Frequency),
		FirstDueDate:     firstDueDate,
		CategoryID:       r.CategoryID,
	}
}

// DebtProgressResponse é a caminhada até a quitação.
type DebtProgressResponse struct {
	TotalCents     int64   `json:"total_cents"`
	PaidCents      int64   `json:"paid_cents"`
	RemainingCents int64   `json:"remaining_cents"`
	PaidCount      int     `json:"paid_count"`
	RemainingCount int     `json:"remaining_count"`
	Percent        int     `json:"percent"`
	NextDueDate    *string `json:"next_due_date"`
	FinalDueDate   string  `json:"final_due_date"`
	Settled        bool    `json:"settled"`
}

// DebtResponse é uma dívida como o cliente a vê, já com o progresso: é o
// número que a tela mostra, e calculá-lo no cliente duplicaria a regra.
type DebtResponse struct {
	ID               int64  `json:"id"`
	Description      string `json:"description"`
	InstallmentCents int64  `json:"installment_amount_cents"`
	Installments     int    `json:"installments"`
	Kind             string `json:"kind"`
	KindLabel        string `json:"kind_label"`
	Frequency        string `json:"frequency"`
	FrequencyLabel   string `json:"frequency_label"`
	FirstDueDate     string `json:"first_due_date"`

	CategoryID    *int64  `json:"category_id"`
	CategoryName  *string `json:"category_name"`
	CategoryColor *string `json:"category_color"`

	Progress DebtProgressResponse `json:"progress"`
}

func NewDebtResponse(debt domain.Debt, progress domain.DebtProgress) DebtResponse {
	response := DebtResponse{
		ID:               debt.ID,
		Description:      debt.Description,
		InstallmentCents: debt.InstallmentCents,
		Installments:     debt.Installments,
		Kind:             string(debt.Kind),
		KindLabel:        debt.Kind.Label(),
		Frequency:        string(debt.Frequency),
		FrequencyLabel:   debt.Frequency.Label(),
		FirstDueDate:     debt.FirstDueDate.Format(dateLayout),
		CategoryID:       debt.CategoryID,
		CategoryName:     debt.CategoryName,
		CategoryColor:    debt.CategoryColor,
		Progress: DebtProgressResponse{
			TotalCents:     progress.TotalCents,
			PaidCents:      progress.PaidCents,
			RemainingCents: progress.RemainingCents,
			PaidCount:      progress.PaidCount,
			RemainingCount: progress.RemainingCount,
			Percent:        progress.Percent,
			FinalDueDate:   progress.FinalDueDate.Format(dateLayout),
			Settled:        progress.Settled,
		},
	}

	if progress.NextDueDate != nil {
		next := progress.NextDueDate.Format(dateLayout)
		response.Progress.NextDueDate = &next
	}

	return response
}

// DebtsSummaryResponse agrega todas as dívidas, para o painel.
type DebtsSummaryResponse struct {
	TotalCents     int64 `json:"total_cents"`
	PaidCents      int64 `json:"paid_cents"`
	RemainingCents int64 `json:"remaining_cents"`
	OpenCount      int   `json:"open_count"`
	SettledCount   int   `json:"settled_count"`
	Percent        int   `json:"percent"`
}

func NewDebtsSummaryResponse(summary domain.DebtsSummary) DebtsSummaryResponse {
	return DebtsSummaryResponse{
		TotalCents:     summary.TotalCents,
		PaidCents:      summary.PaidCents,
		RemainingCents: summary.RemainingCents,
		OpenCount:      summary.OpenCount,
		SettledCount:   summary.SettledCount,
		Percent:        summary.Percent,
	}
}
