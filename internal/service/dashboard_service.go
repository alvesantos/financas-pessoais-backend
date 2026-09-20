package service

import (
	"context"
	"sort"
	"time"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// DashboardService monta as métricas do painel a partir dos mesmos dados da
// tela de lançamentos, incluindo as projeções dos fixos, para que os
// números das duas telas nunca discordem.
type DashboardService struct {
	transactions domain.TransactionRepository
	recurring    domain.RecurringRepository
	debts        domain.DebtRepository
	clock        domain.Clock
}

var _ domain.DashboardService = (*DashboardService)(nil)

func NewDashboardService(
	transactions domain.TransactionRepository,
	recurring domain.RecurringRepository,
	debts domain.DebtRepository,
	clock domain.Clock,
) *DashboardService {
	return &DashboardService{
		transactions: transactions,
		recurring:    recurring,
		debts:        debts,
		clock:        clock,
	}
}

func (s *DashboardService) Overview(
	ctx context.Context, userID int64, year int, month time.Month,
) (*domain.Dashboard, error) {
	// Um único SELECT cobre o ano inteiro; os fixos são projetados mês a mês.
	stored, err := s.transactions.ListByPeriod(ctx, userID, domain.YearPeriod(year))
	if err != nil {
		return nil, err
	}

	recurringEntries, err := s.recurring.ListActive(ctx, userID)
	if err != nil {
		return nil, err
	}

	debts, err := s.debts.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	today := s.clock.Today()
	byMonth := groupByMonth(stored)

	dashboard := &domain.Dashboard{
		Year:   domain.YearTotals{Year: year},
		PorMes: make([]domain.MonthTotals, 0, 12),
	}

	var entriesDoMes []domain.Transaction

	for m := time.January; m <= time.December; m++ {
		period := domain.MonthPeriod(year, m)

		entries := byMonth[m]
		for _, entry := range recurringEntries {
			entries = append(entries, entry.ProjectInto(period)...)
		}
		for _, debt := range debts {
			entries = append(entries, debt.ProjectInto(period)...)
		}

		summary := summarize(entries, year, m, today)

		dashboard.PorMes = append(dashboard.PorMes, domain.MonthTotals{
			Month:    m,
			Receitas: summary.Receitas,
			Despesas: summary.Despesas,
			Saldo:    summary.SaldoPrevisto,
		})

		dashboard.Year.Receitas += summary.Receitas
		dashboard.Year.Despesas += summary.Despesas

		if m == month {
			dashboard.Month = summary
			entriesDoMes = entries
		}
	}

	dashboard.Year.Saldo = dashboard.Year.Receitas - dashboard.Year.Despesas
	dashboard.GastosPorTipo = expensesByKind(dashboard.Month.TotalPorTipo)
	dashboard.GastosPorCategoria = expensesByCategory(entriesDoMes)
	dashboard.MaiorGasto = biggestExpense(entriesDoMes)
	dashboard.DespesasFixas = fixedExpenses(recurringEntries, domain.MonthPeriod(year, month))

	saldoAtual, err := s.accumulatedBalance(ctx, userID, recurringEntries, debts, today)
	if err != nil {
		return nil, err
	}
	dashboard.SaldoAtual = saldoAtual
	dashboard.Dividas = summarizeDebts(debts, today)

	return dashboard, nil
}

// summarizeDebts agrega o que ainda falta pagar em todas as dívidas.
func summarizeDebts(debts []domain.Debt, today time.Time) domain.DebtsSummary {
	summary := domain.DebtsSummary{}

	for _, debt := range debts {
		progress := debt.Progress(today)

		summary.TotalCents += progress.TotalCents
		summary.PaidCents += progress.PaidCents
		summary.RemainingCents += progress.RemainingCents

		if progress.Settled {
			summary.SettledCount++
			continue
		}

		summary.OpenCount++
	}

	if summary.TotalCents > 0 {
		summary.Percent = int(summary.PaidCents * 100 / summary.TotalCents)
	}

	return summary
}

// accumulatedBalance é o saldo de verdade: tudo que já entrou e saiu desde a
// primeira movimentação, sem recorte de período. Os lançamentos gravados são
// somados no banco; os fixos são projetados desde o início de cada regra.
func (s *DashboardService) accumulatedBalance(
	ctx context.Context,
	userID int64,
	recurringEntries []domain.RecurringEntry,
	debts []domain.Debt,
	today time.Time,
) (int64, error) {
	total, err := s.transactions.SumUntil(ctx, userID, today)
	if err != nil {
		return 0, err
	}

	// Parcela vencida já saiu do bolso, como qualquer outro lançamento.
	for _, debt := range debts {
		total -= debt.Progress(today).PaidCents
	}

	for _, entry := range recurringEntries {
		// Cada regra é projetada do próprio início até hoje: o que ainda vai
		// cair não conta no saldo atual.
		period := domain.Period{From: domain.Day(entry.StartDate), To: today}

		for _, occurrence := range entry.ProjectInto(period) {
			total += occurrence.SignedAmount()
		}
	}

	return total, nil
}

// fixedExpenses soma o que os fixos de saída pesam no período. Como a conta
// usa as ocorrências projetadas, um fixo semanal pesa quatro ou cinco vezes
// no mês, sem nenhuma conversão de frequência à mão.
func fixedExpenses(entries []domain.RecurringEntry, period domain.Period) int64 {
	var total int64

	for _, entry := range entries {
		if entry.Kind.IsIncome() {
			continue
		}

		for _, occurrence := range entry.ProjectInto(period) {
			total += occurrence.AmountCents
		}
	}

	return total
}

func groupByMonth(entries []domain.Transaction) map[time.Month][]domain.Transaction {
	byMonth := map[time.Month][]domain.Transaction{}

	for _, entry := range entries {
		month := entry.OccurredAt.Month()
		byMonth[month] = append(byMonth[month], entry)
	}

	return byMonth
}

// expensesByKind devolve só os tipos que subtraem saldo, do maior para o
// menor: o gráfico mostra onde o dinheiro foi.
func expensesByKind(totals map[domain.Kind]int64) []domain.KindTotal {
	result := make([]domain.KindTotal, 0, len(totals))

	for _, kind := range domain.AllKinds {
		if kind.IsIncome() {
			continue
		}

		if total := totals[kind]; total > 0 {
			result = append(result, domain.KindTotal{Kind: kind, Label: kind.Label(), Total: total})
		}
	}

	sort.SliceStable(result, func(i, j int) bool { return result[i].Total > result[j].Total })

	return result
}

// semCategoria é o balde de quem ainda não escolheu uma.
const semCategoria = "Sem categoria"

// corSemCategoria é neutra de propósito: o balde não é uma categoria de
// verdade e não deve competir com as que a pessoa criou.
const corSemCategoria = "#94a3b8"

// expensesByCategory agrupa os gastos do mês por categoria, do maior para o
// menor. É o que o gráfico de composição mostra.
func expensesByCategory(entries []domain.Transaction) []domain.CategoryTotal {
	type bucket struct {
		total domain.CategoryTotal
	}

	// A chave separa cada categoria do balde dos sem categoria.
	buckets := map[int64]*bucket{}
	var ordem []int64

	for _, entry := range entries {
		if entry.Kind.IsIncome() {
			continue
		}

		var chave int64
		rotulo, cor := semCategoria, corSemCategoria

		if entry.CategoryID != nil {
			chave = *entry.CategoryID
			if entry.CategoryName != nil {
				rotulo = *entry.CategoryName
			}
			if entry.CategoryColor != nil {
				cor = *entry.CategoryColor
			}
		}

		if _, existe := buckets[chave]; !existe {
			id := entry.CategoryID
			buckets[chave] = &bucket{total: domain.CategoryTotal{CategoryID: id, Label: rotulo, Color: cor}}
			ordem = append(ordem, chave)
		}

		buckets[chave].total.Total += entry.AmountCents
	}

	result := make([]domain.CategoryTotal, 0, len(ordem))
	for _, chave := range ordem {
		result = append(result, buckets[chave].total)
	}

	sort.SliceStable(result, func(i, j int) bool { return result[i].Total > result[j].Total })

	return result
}

// biggestExpense é o maior gasto isolado do mês, útil como destaque.
func biggestExpense(entries []domain.Transaction) *domain.Transaction {
	var biggest *domain.Transaction

	for i := range entries {
		entry := entries[i]
		if entry.Kind.IsIncome() {
			continue
		}

		if biggest == nil || entry.AmountCents > biggest.AmountCents {
			biggest = &entry
		}
	}

	return biggest
}
