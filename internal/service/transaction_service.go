package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// TransactionService reúne os casos de uso de lançamentos, incluindo a
// projeção dos fixos no mês pedido.
type TransactionService struct {
	transactions domain.TransactionRepository
	recurring    domain.RecurringRepository
	debts        domain.DebtRepository
	categories   domain.CategoryRepository
	cards        domain.CreditCardRepository
	clock        domain.Clock
}

var _ domain.TransactionService = (*TransactionService)(nil)

func NewTransactionService(
	transactions domain.TransactionRepository,
	recurring domain.RecurringRepository,
	debts domain.DebtRepository,
	categories domain.CategoryRepository,
	cards domain.CreditCardRepository,
	clock domain.Clock,
) *TransactionService {
	return &TransactionService{
		transactions: transactions,
		recurring:    recurring,
		debts:        debts,
		categories:   categories,
		cards:        cards,
		clock:        clock,
	}
}

// Create grava um lançamento avulso. Sem descrição, o lançamento recebe o
// nome do próprio tipo.
func (s *TransactionService) Create(ctx context.Context, input domain.NewTransaction) (*domain.Transaction, error) {
	if err := validateAmountAndKind(input.AmountCents, input.Kind); err != nil {
		return nil, err
	}

	prepared, err := s.prepare(ctx, input)
	if err != nil {
		return nil, err
	}

	return s.transactions.Create(ctx, prepared)
}

// Update reescreve um lançamento já gravado, com as mesmas regras da criação.
func (s *TransactionService) Update(
	ctx context.Context, input domain.UpdateTransaction,
) (*domain.Transaction, error) {
	if err := validateAmountAndKind(input.AmountCents, input.Kind); err != nil {
		return nil, err
	}

	prepared, err := s.prepare(ctx, input.NewTransaction)
	if err != nil {
		return nil, err
	}

	return s.transactions.Update(ctx, domain.UpdateTransaction{ID: input.ID, NewTransaction: prepared})
}

// prepare aplica as regras comuns à criação e à edição: categoria válida,
// descrição padrão e o mês da fatura quando o lançamento é de cartão.
func (s *TransactionService) prepare(
	ctx context.Context, input domain.NewTransaction,
) (domain.NewTransaction, error) {
	if err := ensureCategory(ctx, s.categories, input.UserID, input.CategoryID, input.Kind); err != nil {
		return input, err
	}

	input.Description = defaultDescription(input.Description, input.Kind)
	input.OccurredAt = domain.Day(input.OccurredAt)
	input.InvoiceMonth = nil

	if input.CreditCardID == nil {
		return input, nil
	}

	// Cartão só faz sentido em gasto de cartão de crédito.
	if input.Kind != domain.KindCartaoCredito {
		return input, domain.ErrValidation.WithFields(map[string]string{
			"credit_card_id": "o cartão vale só para gasto no cartão de crédito",
		})
	}

	card, err := s.cards.FindByID(ctx, input.UserID, *input.CreditCardID)
	if err != nil {
		if errors.Is(err, domain.ErrCreditCardNotFound) {
			return input, domain.ErrValidation.WithFields(map[string]string{
				"credit_card_id": "cartão não encontrado",
			})
		}
		return input, err
	}

	choice := input.Invoice
	if choice == "" {
		choice = domain.InvoiceCurrent
	}
	if !choice.Valid() {
		return input, domain.ErrValidation.WithFields(map[string]string{
			"invoice": "escolha a fatura atual ou a próxima",
		})
	}

	invoiceMonth := card.InvoiceMonthFor(input.OccurredAt, choice)
	input.InvoiceMonth = &invoiceMonth

	return input, nil
}

// ListMonth devolve os lançamentos do mês: os gravados no banco e os
// projetados a partir dos fixos, em ordem de data.
func (s *TransactionService) ListMonth(
	ctx context.Context, userID int64, year int, month time.Month,
) ([]domain.Transaction, error) {
	period := domain.MonthPeriod(year, month)

	stored, err := s.transactions.ListByPeriod(ctx, userID, period)
	if err != nil {
		return nil, err
	}

	projected, err := s.projectRecurring(ctx, userID, period)
	if err != nil {
		return nil, err
	}

	installments, err := s.projectDebts(ctx, userID, period)
	if err != nil {
		return nil, err
	}

	entries := append(stored, projected...)
	entries = append(entries, installments...)

	return sortByDate(entries), nil
}

// Summary calcula os saldos do mês.
func (s *TransactionService) Summary(
	ctx context.Context, userID int64, year int, month time.Month,
) (*domain.MonthSummary, error) {
	entries, err := s.ListMonth(ctx, userID, year, month)
	if err != nil {
		return nil, err
	}

	summary := summarize(entries, year, month, s.clock.Today())
	return &summary, nil
}

func (s *TransactionService) Delete(ctx context.Context, userID, id int64) error {
	return s.transactions.Delete(ctx, userID, id)
}

// projectRecurring transforma os fixos ativos em lançamentos do período.
func (s *TransactionService) projectRecurring(
	ctx context.Context, userID int64, period domain.Period,
) ([]domain.Transaction, error) {
	entries, err := s.recurring.ListActive(ctx, userID)
	if err != nil {
		return nil, err
	}

	return markPaidByDate(projectRecurringEntries(entries, period), s.clock.Today()), nil
}

// projectRecurringEntries projeta todos os fixos de uma vez.
func projectRecurringEntries(entries []domain.RecurringEntry, period domain.Period) []domain.Transaction {
	var projected []domain.Transaction
	for _, entry := range entries {
		projected = append(projected, entry.ProjectInto(period)...)
	}

	return projected
}

// markPaidByDate vale para as projeções, que não têm marcação própria: o que
// já venceu é tratado como pago, como o extrato de quem paga em dia.
func markPaidByDate(entries []domain.Transaction, today time.Time) []domain.Transaction {
	for i := range entries {
		entries[i].Paid = !entries[i].OccurredAt.After(today)
	}

	return entries
}

// projectDebts transforma as parcelas que vencem no período em lançamentos.
func (s *TransactionService) projectDebts(
	ctx context.Context, userID int64, period domain.Period,
) ([]domain.Transaction, error) {
	debts, err := s.debts.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	var projected []domain.Transaction
	for _, debt := range debts {
		projected = append(projected, debt.ProjectInto(period)...)
	}

	return markPaidByDate(projected, s.clock.Today()), nil
}

// summarize separa o que já aconteceu do que ainda vai acontecer.
func summarize(entries []domain.Transaction, year int, month time.Month, today time.Time) domain.MonthSummary {
	summary := domain.MonthSummary{
		Year:            year,
		Month:           month,
		TotalPorTipo:    map[domain.Kind]int64{},
		QuantidadeItens: len(entries),
	}

	for _, entry := range entries {
		signed := entry.SignedAmount()

		summary.SaldoPrevisto += signed
		if entry.Paid {
			summary.SaldoAtual += signed
		}

		summary.TotalPorTipo[entry.Kind] += entry.AmountCents

		if entry.Kind.IsIncome() {
			summary.Receitas += entry.AmountCents
		} else {
			summary.Despesas += entry.AmountCents
		}
	}

	return summary
}

// sortByDate ordena por data e, no mesmo dia, mantém os gravados antes das
// projeções, para a lista não dançar entre requisições.
func sortByDate(entries []domain.Transaction) []domain.Transaction {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]

		if !a.OccurredAt.Equal(b.OccurredAt) {
			return a.OccurredAt.Before(b.OccurredAt)
		}
		if a.IsProjected() != b.IsProjected() {
			return !a.IsProjected()
		}

		return a.ID < b.ID
	})

	return entries
}

// defaultDescription aplica a regra: sem descrição, usa o nome do tipo.
func defaultDescription(description string, kind domain.Kind) string {
	if trimmed := strings.TrimSpace(description); trimmed != "" {
		return trimmed
	}
	return kind.Label()
}

func validateAmountAndKind(amountCents int64, kind domain.Kind) error {
	fields := map[string]string{}

	if amountCents <= 0 {
		fields["amount_cents"] = "informe um valor maior que zero"
	}
	if !kind.Valid() {
		fields["kind"] = "tipo de lançamento inválido"
	}

	if len(fields) > 0 {
		return domain.ErrValidation.WithFields(fields)
	}
	return nil
}
