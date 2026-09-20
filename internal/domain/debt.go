package domain

import "time"

// maxInstallments limita o parcelamento ao que o banco também aceita.
const maxInstallments = 600

// Debt é uma dívida parcelada: um empréstimo, um acordo, uma compra em
// muitas vezes. Diferente de um fixo, ela tem um número de parcelas e, por
// isso, um fim e um progresso.
type Debt struct {
	ID               int64
	UserID           int64
	Description      string
	InstallmentCents int64
	Installments     int
	Kind             Kind
	Frequency        Frequency
	FirstDueDate     time.Time
	CreatedAt        time.Time

	CategoryID    *int64
	CategoryName  *string
	CategoryColor *string
}

// NewDebt são os dados para registrar uma dívida.
type NewDebt struct {
	UserID           int64
	Description      string
	InstallmentCents int64
	Installments     int
	Kind             Kind
	Frequency        Frequency
	FirstDueDate     time.Time
	CategoryID       *int64
}

// Installment é uma parcela da dívida.
type Installment struct {
	Number      int
	DueDate     time.Time
	AmountCents int64
	// Paid diz que a parcela já venceu. O sistema não registra pagamento em
	// separado: o que venceu é tratado como pago, igual ao saldo atual.
	Paid bool
}

// DebtProgress é o quanto da dívida já foi vencido e o quanto falta.
type DebtProgress struct {
	TotalCents     int64
	PaidCents      int64
	RemainingCents int64
	PaidCount      int
	RemainingCount int
	// Percent vai de 0 a 100, arredondado para baixo.
	Percent      int
	NextDueDate  *time.Time
	FinalDueDate time.Time
	Settled      bool
}

// TotalCents é quanto a dívida custa por inteiro.
func (d Debt) TotalCents() int64 {
	return d.InstallmentCents * int64(d.Installments)
}

// DueDateOf devolve o vencimento da parcela informada, contando de 1.
func (d Debt) DueDateOf(number int) time.Time {
	return occurrenceAt(Day(d.FirstDueDate), d.Frequency, number-1)
}

// FinalDueDate é o vencimento da última parcela.
func (d Debt) FinalDueDate() time.Time {
	return d.DueDateOf(d.Installments)
}

// Schedule devolve todas as parcelas, em ordem, marcando as já vencidas.
func (d Debt) Schedule(today time.Time) []Installment {
	today = Day(today)
	schedule := make([]Installment, 0, d.Installments)

	for number := 1; number <= d.Installments; number++ {
		dueDate := d.DueDateOf(number)

		schedule = append(schedule, Installment{
			Number:      number,
			DueDate:     dueDate,
			AmountCents: d.InstallmentCents,
			Paid:        !dueDate.After(today),
		})
	}

	return schedule
}

// Progress resume a caminhada até a quitação.
func (d Debt) Progress(today time.Time) DebtProgress {
	today = Day(today)

	progress := DebtProgress{
		TotalCents:   d.TotalCents(),
		FinalDueDate: d.FinalDueDate(),
	}

	for _, installment := range d.Schedule(today) {
		if installment.Paid {
			progress.PaidCount++
			progress.PaidCents += installment.AmountCents
			continue
		}

		progress.RemainingCount++
		progress.RemainingCents += installment.AmountCents

		if progress.NextDueDate == nil {
			dueDate := installment.DueDate
			progress.NextDueDate = &dueDate
		}
	}

	if progress.TotalCents > 0 {
		progress.Percent = int(progress.PaidCents * 100 / progress.TotalCents)
	}
	progress.Settled = progress.RemainingCount == 0

	return progress
}

// ProjectInto devolve as parcelas que caem no período, como lançamentos.
func (d Debt) ProjectInto(period Period) []Transaction {
	var projected []Transaction

	for number := 1; number <= d.Installments; number++ {
		dueDate := d.DueDateOf(number)

		if dueDate.After(period.To) {
			break
		}
		if dueDate.Before(period.From) {
			continue
		}

		debtID := d.ID
		installmentNumber := number
		installmentsTotal := d.Installments

		projected = append(projected, Transaction{
			UserID:            d.UserID,
			Description:       d.Description,
			AmountCents:       d.InstallmentCents,
			Kind:              d.Kind,
			OccurredAt:        dueDate,
			DebtID:            &debtID,
			InstallmentNumber: &installmentNumber,
			InstallmentsTotal: &installmentsTotal,
			CategoryID:        d.CategoryID,
			CategoryName:      d.CategoryName,
			CategoryColor:     d.CategoryColor,
		})
	}

	return projected
}

var ErrDebtNotFound = NewError(CodeNotFound, "dívida não encontrada")
