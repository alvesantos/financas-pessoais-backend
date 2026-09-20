package domain

import "time"

// AmortizationMode diz o que fazer com o parcelamento depois de amortizar.
type AmortizationMode string

const (
	// AmortizeKeepInstallment mantém o valor da parcela e encurta o
	// parcelamento. É o padrão: quem amortiza costuma querer terminar antes.
	AmortizeKeepInstallment AmortizationMode = "manter_parcela"
	// AmortizeRecalcInstallment mantém o número de parcelas que faltam e
	// baixa o valor de cada uma.
	AmortizeRecalcInstallment AmortizationMode = "recalcular_parcela"
	// AmortizeRecalcCount usa o número de parcelas informado e distribui o
	// que resta entre elas.
	AmortizeRecalcCount AmortizationMode = "recalcular_parcelas"
)

func (m AmortizationMode) Valid() bool {
	switch m {
	case AmortizeKeepInstallment, AmortizeRecalcInstallment, AmortizeRecalcCount:
		return true
	default:
		return false
	}
}

// Amortization é um abatimento no saldo devedor.
type Amortization struct {
	AmountCents int64
	// NewRemainingCents, quando informado, define o saldo devedor final em
	// vez de deduzi-lo do valor amortizado.
	NewRemainingCents *int64
	Mode              AmortizationMode
	// Installments vale só para AmortizeRecalcCount.
	Installments *int
}

var (
	ErrAmortizationTooBig = NewError(CodeValidation, "o valor passa do que falta pagar")
	ErrDebtAlreadySettled = NewError(CodeConflict, "esta dívida já está quitada")
)

// Amortize devolve a dívida com o abatimento aplicado.
//
// O parcelamento é refeito a partir da próxima parcela: o que já venceu vira
// valor amortizado, então o total e o quanto já foi pago continuam batendo
// mesmo quando o valor da parcela muda.
func (d Debt) Amortize(input Amortization, today time.Time) (Debt, error) {
	if d.IsSettled() {
		return d, ErrDebtAlreadySettled
	}

	progress := d.Progress(today)

	if input.AmountCents <= 0 {
		return d, ErrValidation.WithFields(map[string]string{
			"amount_cents": "informe um valor maior que zero",
		})
	}
	if input.AmountCents > progress.RemainingCents {
		return d, ErrAmortizationTooBig.WithFields(map[string]string{
			"amount_cents": "o valor passa do que falta pagar",
		})
	}

	remainingAfter := progress.RemainingCents - input.AmountCents
	if input.NewRemainingCents != nil {
		remainingAfter = *input.NewRemainingCents

		if remainingAfter < 0 || remainingAfter > progress.RemainingCents {
			return d, ErrValidation.WithFields(map[string]string{
				"new_remaining_cents": "o novo saldo precisa ficar entre zero e o que falta hoje",
			})
		}
	}

	// O que foi abatido saiu do bolso, então conta como pago: o total da
	// dívida não muda, só a divisão entre pago e a pagar.
	abatido := progress.RemainingCents - remainingAfter

	updated := d
	// Tudo que já venceu também sai do cronograma e vira valor amortizado,
	// para o total não se perder quando a parcela mudar de valor.
	updated.AmortizedCents = progress.PaidCents + abatido

	if remainingAfter == 0 {
		// O parcelamento fica de pé para o total não se perder; a data da
		// quitação é o que encerra a dívida.
		settledAt := Day(today)
		updated.SettledAt = &settledAt

		return updated, nil
	}

	count, err := installmentCountFor(input, progress, d.InstallmentCents, remainingAfter)
	if err != nil {
		return d, err
	}

	// A sobra da divisão vira valor amortizado, para as parcelas ficarem
	// todas iguais e o total continuar exato.
	value := remainingAfter / int64(count)
	leftover := remainingAfter - value*int64(count)

	if value == 0 {
		return d, ErrValidation.WithFields(map[string]string{
			"installments": "são parcelas demais para o saldo que resta",
		})
	}

	updated.InstallmentCents = value
	updated.Installments = count
	updated.AmortizedCents += leftover
	updated.FirstDueDate = nextDueDateAfter(d, progress, today)

	return updated, nil
}

// installmentCountFor decide em quantas parcelas o saldo restante cabe.
func installmentCountFor(
	input Amortization, progress DebtProgress, installmentCents, remainingAfter int64,
) (int, error) {
	switch input.Mode {
	case AmortizeRecalcInstallment:
		// Mesmas parcelas que faltavam, cada uma mais barata.
		return progress.RemainingCount, nil

	case AmortizeRecalcCount:
		if input.Installments == nil || *input.Installments <= 0 {
			return 0, ErrValidation.WithFields(map[string]string{
				"installments": "informe quantas parcelas ficam",
			})
		}
		if *input.Installments > maxInstallments {
			return 0, ErrValidation.WithFields(map[string]string{
				"installments": "o limite é de 600 parcelas",
			})
		}
		return *input.Installments, nil

	default:
		// Mantém o valor da parcela: o parcelamento encurta.
		count := int(remainingAfter / installmentCents)
		if remainingAfter%installmentCents != 0 {
			count++
		}
		return count, nil
	}
}

// nextDueDateAfter é a data em que o novo parcelamento começa.
func nextDueDateAfter(debt Debt, progress DebtProgress, today time.Time) time.Time {
	if progress.NextDueDate != nil {
		return *progress.NextDueDate
	}

	// Sem próxima parcela no cronograma antigo, o novo começa um período
	// depois da última que venceu.
	return occurrenceAt(Day(today), debt.Frequency, 1)
}

// Settle marca a dívida como quitada.
func (d Debt) Settle(today time.Time) (Debt, error) {
	if d.IsSettled() {
		return d, ErrDebtAlreadySettled
	}

	settledAt := Day(today)

	// Só a data muda: o parcelamento fica de pé para o total continuar
	// visível, e o que faltava não entra em AmortizedCents porque pode ter
	// virado um lançamento de verdade.
	updated := d
	updated.SettledAt = &settledAt

	return updated, nil
}
