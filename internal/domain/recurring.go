package domain

import "time"

// RecurringEntry é um lançamento fixo: a regra que projeta ocorrências nos
// meses, como "Academia, todo dia 20, R$ 159,90".
type RecurringEntry struct {
	ID          int64
	UserID      int64
	Description string
	AmountCents int64
	Kind        Kind
	Frequency   Frequency
	StartDate   time.Time
	EndDate     *time.Time
	Active      bool
	CreatedAt   time.Time

	CategoryID    *int64
	CategoryName  *string
	CategoryColor *string
}

// NewRecurringEntry são os dados para criar um fixo.
type NewRecurringEntry struct {
	UserID      int64
	Description string
	AmountCents int64
	Kind        Kind
	Frequency   Frequency
	StartDate   time.Time
	EndDate     *time.Time
	CategoryID  *int64
}

// UpdateRecurringEntry são os dados para reescrever um fixo.
type UpdateRecurringEntry struct {
	ID int64
	NewRecurringEntry
	Active bool
}

// MonthlyCostCents é quanto o fixo pesa em um mês médio.
//
// A conta passa pelo ano porque nem toda frequência cabe redonda em um mês:
// um semanal cai quatro ou cinco vezes conforme o mês, e um semestral não
// cai em dez deles. Espalhar pelo ano dá um número estável, que é o que se
// espera de um custo de vida.
func (r RecurringEntry) MonthlyCostCents() int64 {
	return r.AmountCents * int64(r.Frequency.OccurrencesPerYear()) / 12
}

// CountsAsCostOn diz se o fixo ainda representa um custo na data informada.
// Um fixo inativo ou já encerrado saiu do orçamento; um que só começa mês
// que vem continua sendo um compromisso assumido.
func (r RecurringEntry) CountsAsCostOn(today time.Time) bool {
	if !r.Active {
		return false
	}

	if r.EndDate != nil && Day(*r.EndDate).Before(Day(today)) {
		return false
	}

	return true
}

// OccurrencesIn devolve, em ordem, as datas em que o fixo cai dentro do
// período. A regra nunca produz datas antes do início nem depois do fim.
func (r RecurringEntry) OccurrencesIn(period Period) []time.Time {
	if !r.Active {
		return nil
	}

	start := Day(r.StartDate)
	if start.After(period.To) {
		return nil
	}

	limit := period.To
	if r.EndDate != nil {
		end := Day(*r.EndDate)
		if end.Before(period.From) {
			return nil
		}
		if end.Before(limit) {
			limit = end
		}
	}

	if step, ok := r.Frequency.stepInDays(); ok {
		return occurrencesByDays(start, period.From, limit, step)
	}

	step, ok := r.Frequency.stepInMonths()
	if !ok {
		return nil
	}

	return occurrencesByMonths(start, period.From, limit, step)
}

// OccursOn diz se o fixo cai exatamente na data informada.
func (r RecurringEntry) OccursOn(date time.Time) bool {
	date = Day(date)
	return len(r.OccurrencesIn(Period{From: date, To: date})) > 0
}

// ProjectInto converte as ocorrências do período em lançamentos projetados.
func (r RecurringEntry) ProjectInto(period Period) []Transaction {
	dates := r.OccurrencesIn(period)
	projected := make([]Transaction, 0, len(dates))

	for _, date := range dates {
		recurringID := r.ID
		frequency := r.Frequency

		projected = append(projected, Transaction{
			Projected:     true,
			UserID:        r.UserID,
			Description:   r.Description,
			AmountCents:   r.AmountCents,
			Kind:          r.Kind,
			OccurredAt:    date,
			RecurringID:   &recurringID,
			Frequency:     &frequency,
			CategoryID:    r.CategoryID,
			CategoryName:  r.CategoryName,
			CategoryColor: r.CategoryColor,
		})
	}

	return projected
}

// occurrenceAt devolve a data da ocorrência de índice n, contando 0 para a
// primeira. É o que permite numerar as parcelas de uma dívida.
func occurrenceAt(start time.Time, frequency Frequency, index int) time.Time {
	if step, ok := frequency.stepInDays(); ok {
		return start.AddDate(0, 0, index*step)
	}

	if step, ok := frequency.stepInMonths(); ok {
		return monthlyOccurrence(start, index*step)
	}

	return start
}

// occurrencesByDays salta de N em N dias a partir do início, sem percorrer
// todo o intervalo desde a data de início.
func occurrencesByDays(start, from, to time.Time, step int) []time.Time {
	current := start

	if start.Before(from) {
		days := int(from.Sub(start).Hours() / 24)
		saltos := days / step
		if days%step != 0 {
			saltos++
		}
		current = start.AddDate(0, 0, saltos*step)
	}

	var dates []time.Time
	for !current.After(to) {
		dates = append(dates, current)
		current = current.AddDate(0, 0, step)
	}

	return dates
}

// occurrencesByMonths repete o dia do mês de início a cada N meses. Um dia
// 31 cai no último dia dos meses mais curtos, em vez de vazar para o
// mês seguinte.
func occurrencesByMonths(start, from, to time.Time, step int) []time.Time {
	var dates []time.Time

	// Começa no mês de `from`, recuando até alinhar com o ciclo do início.
	months := (from.Year()-start.Year())*12 + int(from.Month()-start.Month())
	if months < 0 {
		months = 0
	}
	months -= months % step

	for {
		candidate := monthlyOccurrence(start, months)
		if candidate.After(to) {
			return dates
		}

		if !candidate.Before(from) && !candidate.Before(start) {
			dates = append(dates, candidate)
		}

		months += step
	}
}

// monthlyOccurrence devolve a data N meses depois do início, com o dia
// preso ao último dia do mês quando ele não existe.
func monthlyOccurrence(start time.Time, monthsAhead int) time.Time {
	target := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC).
		AddDate(0, monthsAhead, 0)

	day := start.Day()
	if last := daysInMonth(target.Year(), target.Month()); day > last {
		day = last
	}

	return time.Date(target.Year(), target.Month(), day, 0, 0, 0, 0, time.UTC)
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
