package dto

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// CreateRecurringRequest é o corpo de POST /api/recurring.
type CreateRecurringRequest struct {
	Description string  `json:"description"`
	AmountCents int64   `json:"amount_cents"`
	Kind        string  `json:"kind"`
	Frequency   string  `json:"frequency"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date"`
	CategoryID  *int64  `json:"category_id"`
}

func (r CreateRecurringRequest) Validate() error {
	fields := map[string]string{}

	if r.AmountCents <= 0 {
		fields["amount_cents"] = "informe um valor maior que zero"
	}
	if !domain.Kind(r.Kind).Valid() {
		fields["kind"] = "escolha um tipo de lançamento"
	}
	if !domain.Frequency(r.Frequency).Valid() {
		fields["frequency"] = "escolha uma frequência"
	}
	if utf8.RuneCountInString(strings.TrimSpace(r.Description)) > maxDescriptionLength {
		fields["description"] = "a descrição passou de 120 caracteres"
	}

	start, err := parseDate(r.StartDate)
	if err != nil {
		fields["start_date"] = "informe uma data válida"
	}

	if r.EndDate != nil && strings.TrimSpace(*r.EndDate) != "" {
		end, err := parseDate(*r.EndDate)
		switch {
		case err != nil:
			fields["end_date"] = "informe uma data válida"
		case end.Before(start):
			fields["end_date"] = "o fim não pode ser antes do início"
		}
	}

	if len(fields) > 0 {
		return domain.ErrValidation.WithFields(fields)
	}
	return nil
}

func (r CreateRecurringRequest) ToDomain(userID int64) domain.NewRecurringEntry {
	startDate, _ := parseDate(r.StartDate)

	var endDate *time.Time
	if r.EndDate != nil && strings.TrimSpace(*r.EndDate) != "" {
		if parsed, err := parseDate(*r.EndDate); err == nil {
			endDate = &parsed
		}
	}

	return domain.NewRecurringEntry{
		UserID:      userID,
		Description: strings.TrimSpace(r.Description),
		AmountCents: r.AmountCents,
		Kind:        domain.Kind(r.Kind),
		Frequency:   domain.Frequency(r.Frequency),
		StartDate:   startDate,
		EndDate:     endDate,
		CategoryID:  r.CategoryID,
	}
}

// RecurringResponse é um lançamento fixo como o cliente o vê.
type RecurringResponse struct {
	ID             int64   `json:"id"`
	Description    string  `json:"description"`
	AmountCents    int64   `json:"amount_cents"`
	Kind           string  `json:"kind"`
	KindLabel      string  `json:"kind_label"`
	Frequency      string  `json:"frequency"`
	FrequencyLabel string  `json:"frequency_label"`
	StartDate      string  `json:"start_date"`
	EndDate        *string `json:"end_date"`
	Active         bool    `json:"active"`

	CategoryID    *int64  `json:"category_id"`
	CategoryName  *string `json:"category_name"`
	CategoryColor *string `json:"category_color"`
}

func NewRecurringResponse(entry domain.RecurringEntry) RecurringResponse {
	response := RecurringResponse{
		ID:             entry.ID,
		Description:    entry.Description,
		AmountCents:    entry.AmountCents,
		Kind:           string(entry.Kind),
		KindLabel:      entry.Kind.Label(),
		Frequency:      string(entry.Frequency),
		FrequencyLabel: entry.Frequency.Label(),
		StartDate:      entry.StartDate.Format(dateLayout),
		Active:         entry.Active,
		CategoryID:     entry.CategoryID,
		CategoryName:   entry.CategoryName,
		CategoryColor:  entry.CategoryColor,
	}

	if entry.EndDate != nil {
		end := entry.EndDate.Format(dateLayout)
		response.EndDate = &end
	}

	return response
}

func NewRecurringListResponse(entries []domain.RecurringEntry) []RecurringResponse {
	list := make([]RecurringResponse, 0, len(entries))
	for _, entry := range entries {
		list = append(list, NewRecurringResponse(entry))
	}
	return list
}
