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

	// AmortizedCents é o que já saiu fora do cronograma: amortizações e o
	// que ficou para trás quando o parcelamento foi refeito.
	AmortizedCents int64
	// SettledAt marca a quitação declarada pela pessoa.
	SettledAt *time.Time

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

// UpdateDebt são os dados para reescrever uma dívida.
type UpdateDebt struct {
	ID int64
	NewDebt
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

// TotalCents é quanto a dívida custa por inteiro: o que ainda está no
// cronograma mais o que já saiu fora dele.
func (d Debt) TotalCents() int64 {
	return d.InstallmentCents*int64(d.Installments) + d.AmortizedCents
}

// IsSettled diz se a dívida foi declarada quitada.
func (d Debt) IsSettled() bool {
	return d.SettledAt != nil
}

// OutOfPocketCents é o dinheiro que saiu da carteira sem virar lançamento:
// as parcelas que venceram e o que foi amortizado. A quitação fica de fora
// de propósito, porque ela vira um lançamento de verdade quando a pessoa
// pede para descontar do saldo, e não deve ser contada duas vezes.
func (d Debt) OutOfPocketCents(today time.Time) int64 {
	_, cents := d.paidByTime(today)
	return cents + d.AmortizedCents
}

// paidByTime é quanto do cronograma já venceu.
func (d Debt) paidByTime(today time.Time) (count int, cents int64) {
	today = Day(today)

	for number := 1; number <= d.Installments; number++ {
		if d.DueDateOf(number).After(today) {
			break
		}
		count++
	}

	return count, int64(count) * d.InstallmentCents
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
// Uma dívida quitada não tem mais parcelas a cumprir.
func (d Debt) Schedule(today time.Time) []Installment {
	if d.IsSettled() {
		return nil
	}

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
		PaidCents:    d.AmortizedCents,
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

	progress.Settled = d.IsSettled() || progress.RemainingCount == 0
	if progress.Settled {
		// Quitada, tudo que havia a pagar está pago, venha de onde vier.
		progress.PaidCents = progress.TotalCents
		progress.Percent = 100
		progress.RemainingCents = 0
		progress.RemainingCount = 0
		progress.NextDueDate = nil
	}

	return progress
}

// ProjectInto devolve as parcelas que caem no período, como lançamentos.
// Dívida quitada não projeta mais nada.
func (d Debt) ProjectInto(period Period) []Transaction {
	if d.IsSettled() {
		return nil
	}

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

// FinalDueDate de uma dívida sem parcelas restantes é a data da quitação.
var ErrDebtNotFound = NewError(CodeNotFound, "dívida não encontrada")
