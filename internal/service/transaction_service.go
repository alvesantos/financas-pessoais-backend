package service

import (
	"context"
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
	categories   domain.CategoryRepository
	clock        domain.Clock
}

var _ domain.TransactionService = (*TransactionService)(nil)

func NewTransactionService(
	transactions domain.TransactionRepository,
	recurring domain.RecurringRepository,
	categories domain.CategoryRepository,
	clock domain.Clock,
) *TransactionService {
	return &TransactionService{
		transactions: transactions,
		recurring:    recurring,
		categories:   categories,
		clock:        clock,
	}
}

// Create grava um lançamento avulso. Sem descrição, o lançamento recebe o
// nome do próprio tipo.
func (s *TransactionService) Create(ctx context.Context, input domain.NewTransaction) (*domain.Transaction, error) {
	if err := validateAmountAndKind(input.AmountCents, input.Kind); err != nil {
		return nil, err
	}

	if err := ensureCategory(ctx, s.categories, input.UserID, input.CategoryID, input.Kind); err != nil {
		return nil, err
	}

	input.Description = defaultDescription(input.Description, input.Kind)
	input.OccurredAt = domain.Day(input.OccurredAt)

	return s.transactions.Create(ctx, input)
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

	return sortByDate(append(stored, projected...)), nil
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

	var projected []domain.Transaction
	for _, entry := range entries {
		projected = append(projected, entry.ProjectInto(period)...)
	}

	return projected, nil
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
		if !entry.OccurredAt.After(today) {
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
