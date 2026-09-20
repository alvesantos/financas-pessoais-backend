package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/alvesantos/financas-backend/internal/domain"
	"github.com/alvesantos/financas-backend/internal/service"
)

// --- dublês -----------------------------------------------------------------

type fakeTransactionRepo struct {
	items  []domain.Transaction
	nextID int64
}

func (f *fakeTransactionRepo) Create(_ context.Context, input domain.NewTransaction) (*domain.Transaction, error) {
	f.nextID++

	t := domain.Transaction{
		ID:          f.nextID,
		UserID:      input.UserID,
		Description: input.Description,
		AmountCents: input.AmountCents,
		Kind:        input.Kind,
		OccurredAt:  input.OccurredAt,
	}
	f.items = append(f.items, t)

	return &t, nil
}

func (f *fakeTransactionRepo) ListByPeriod(_ context.Context, _ int64, period domain.Period) ([]domain.Transaction, error) {
	var out []domain.Transaction
	for _, item := range f.items {
		if period.Contains(item.OccurredAt) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeTransactionRepo) Delete(_ context.Context, _, id int64) error {
	for i, item := range f.items {
		if item.ID == id {
			f.items = append(f.items[:i], f.items[i+1:]...)
			return nil
		}
	}
	return domain.ErrTransactionNotFound
}

type fakeRecurringRepo struct {
	items []domain.RecurringEntry
}

func (f *fakeRecurringRepo) Create(_ context.Context, input domain.NewRecurringEntry) (*domain.RecurringEntry, error) {
	entry := domain.RecurringEntry{
		ID:          int64(len(f.items) + 1),
		UserID:      input.UserID,
		Description: input.Description,
		AmountCents: input.AmountCents,
		Kind:        input.Kind,
		Frequency:   input.Frequency,
		StartDate:   input.StartDate,
		EndDate:     input.EndDate,
		Active:      true,
	}
	f.items = append(f.items, entry)

	return &entry, nil
}

func (f *fakeRecurringRepo) List(_ context.Context, _ int64) ([]domain.RecurringEntry, error) {
	return f.items, nil
}

func (f *fakeRecurringRepo) ListActive(_ context.Context, _ int64) ([]domain.RecurringEntry, error) {
	var out []domain.RecurringEntry
	for _, item := range f.items {
		if item.Active {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeRecurringRepo) Delete(_ context.Context, _, id int64) error {
	for i, item := range f.items {
		if item.ID == id {
			f.items = append(f.items[:i], f.items[i+1:]...)
			return nil
		}
	}
	return domain.ErrRecurringNotFound
}

// relogioFixo congela "hoje" para que o saldo atual seja determinístico.
type relogioFixo struct{ hoje time.Time }

func (r relogioFixo) Today() time.Time { return r.hoje }

// --- apoio ------------------------------------------------------------------

const usuario = int64(1)

func dia(ano int, mes time.Month, d int) time.Time {
	return time.Date(ano, mes, d, 0, 0, 0, 0, time.UTC)
}

func novoServico(hoje time.Time) (*service.TransactionService, *fakeTransactionRepo, *fakeRecurringRepo) {
	transacoes := &fakeTransactionRepo{}
	fixos := &fakeRecurringRepo{}

	return service.NewTransactionService(transacoes, fixos, relogioFixo{hoje: hoje}), transacoes, fixos
}

func criar(t *testing.T, svc *service.TransactionService, descricao string, centavos int64, tipo domain.Kind, data time.Time) *domain.Transaction {
	t.Helper()

	created, err := svc.Create(context.Background(), domain.NewTransaction{
		UserID: usuario, Description: descricao, AmountCents: centavos, Kind: tipo, OccurredAt: data,
	})
	if err != nil {
		t.Fatalf("criar lançamento: %v", err)
	}

	return created
}

// --- testes -----------------------------------------------------------------

func TestDescricaoVaziaViraONomeDoTipo(t *testing.T) {
	svc, _, _ := novoServico(dia(2026, time.September, 20))

	casos := map[domain.Kind]string{
		domain.KindReceita:       "Receita",
		domain.KindDespesa:       "Despesa",
		domain.KindCartaoCredito: "Gasto no cartão de crédito",
		domain.KindInvestimento:  "Investimento",
	}

	for tipo, esperado := range casos {
		criado := criar(t, svc, "   ", 10000, tipo, dia(2026, time.September, 10))

		if criado.Description != esperado {
			t.Errorf("%s: descrição = %q, esperava %q", tipo, criado.Description, esperado)
		}
	}
}

func TestDescricaoInformadaEhPreservada(t *testing.T) {
	svc, _, _ := novoServico(dia(2026, time.September, 20))

	criado := criar(t, svc, "  Mercado  ", 10000, domain.KindDespesa, dia(2026, time.September, 10))

	if criado.Description != "Mercado" {
		t.Errorf("descrição = %q, esperava %q", criado.Description, "Mercado")
	}
}

func TestValorZeroOuNegativoEhRecusado(t *testing.T) {
	svc, _, _ := novoServico(dia(2026, time.September, 20))

	for _, valor := range []int64{0, -100} {
		_, err := svc.Create(context.Background(), domain.NewTransaction{
			UserID: usuario, AmountCents: valor, Kind: domain.KindDespesa, OccurredAt: dia(2026, time.September, 10),
		})

		appErr, ok := domain.AsError(err)
		if !ok || appErr.Code != domain.CodeValidation {
			t.Errorf("valor %d: esperava erro de validação, veio %v", valor, err)
		}
		if appErr != nil && appErr.Fields["amount_cents"] == "" {
			t.Errorf("valor %d: esperava o erro no campo amount_cents", valor)
		}
	}
}

func TestTipoInvalidoEhRecusado(t *testing.T) {
	svc, _, _ := novoServico(dia(2026, time.September, 20))

	_, err := svc.Create(context.Background(), domain.NewTransaction{
		UserID: usuario, AmountCents: 10000, Kind: "pix", OccurredAt: dia(2026, time.September, 10),
	})

	appErr, ok := domain.AsError(err)
	if !ok || appErr.Fields["kind"] == "" {
		t.Errorf("esperava erro de validação no campo kind, veio %v", err)
	}
}

func TestSaldoSoSomaComReceita(t *testing.T) {
	svc, _, _ := novoServico(dia(2026, time.September, 30))

	criar(t, svc, "Salário", 500000, domain.KindReceita, dia(2026, time.September, 5))
	criar(t, svc, "Mercado", 50000, domain.KindDespesa, dia(2026, time.September, 6))
	criar(t, svc, "Cartão", 30000, domain.KindCartaoCredito, dia(2026, time.September, 7))
	criar(t, svc, "Tesouro", 20000, domain.KindInvestimento, dia(2026, time.September, 8))

	resumo, err := svc.Summary(context.Background(), usuario, 2026, time.September)
	if err != nil {
		t.Fatalf("resumo: %v", err)
	}

	// 5000,00 - 500,00 - 300,00 - 200,00 = 4000,00
	if resumo.SaldoPrevisto != 400000 {
		t.Errorf("saldo previsto = %d, esperava 400000", resumo.SaldoPrevisto)
	}
	if resumo.Receitas != 500000 {
		t.Errorf("receitas = %d, esperava 500000", resumo.Receitas)
	}
	if resumo.Despesas != 100000 {
		t.Errorf("despesas = %d, esperava 100000", resumo.Despesas)
	}
}

func TestSaldoAtualIgnoraOQueAindaNaoAconteceu(t *testing.T) {
	hoje := dia(2026, time.September, 15)
	svc, _, _ := novoServico(hoje)

	criar(t, svc, "Salário", 500000, domain.KindReceita, dia(2026, time.September, 5))
	criar(t, svc, "Aluguel", 200000, domain.KindDespesa, dia(2026, time.September, 25))

	resumo, err := svc.Summary(context.Background(), usuario, 2026, time.September)
	if err != nil {
		t.Fatalf("resumo: %v", err)
	}

	if resumo.SaldoAtual != 500000 {
		t.Errorf("saldo atual = %d, esperava 500000 (o aluguel ainda não caiu)", resumo.SaldoAtual)
	}
	if resumo.SaldoPrevisto != 300000 {
		t.Errorf("saldo previsto = %d, esperava 300000", resumo.SaldoPrevisto)
	}
}

func TestLancamentoDeHojeContaNoSaldoAtual(t *testing.T) {
	hoje := dia(2026, time.September, 15)
	svc, _, _ := novoServico(hoje)

	criar(t, svc, "Almoço", 5000, domain.KindDespesa, hoje)

	resumo, _ := svc.Summary(context.Background(), usuario, 2026, time.September)

	if resumo.SaldoAtual != -5000 {
		t.Errorf("saldo atual = %d, esperava -5000", resumo.SaldoAtual)
	}
}

func TestFixoApareceNaListaDoMes(t *testing.T) {
	svc, _, fixos := novoServico(dia(2026, time.September, 30))

	if _, err := fixos.Create(context.Background(), domain.NewRecurringEntry{
		UserID:      usuario,
		Description: "Academia",
		AmountCents: 15990,
		Kind:        domain.KindDespesa,
		Frequency:   domain.FrequencyMensal,
		StartDate:   dia(2026, time.January, 20),
	}); err != nil {
		t.Fatalf("criar fixo: %v", err)
	}

	entradas, err := svc.ListMonth(context.Background(), usuario, 2026, time.September)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}

	if len(entradas) != 1 {
		t.Fatalf("esperava 1 lançamento projetado, veio %d", len(entradas))
	}

	projetado := entradas[0]
	if !projetado.IsProjected() {
		t.Error("o lançamento do fixo precisa vir marcado como projeção")
	}
	if projetado.OccurredAt.Day() != 20 {
		t.Errorf("dia = %d, esperava 20", projetado.OccurredAt.Day())
	}
	if projetado.Description != "Academia" {
		t.Errorf("descrição = %q, esperava %q", projetado.Description, "Academia")
	}
}

func TestFixoEntraNoSaldoPrevisto(t *testing.T) {
	svc, _, fixos := novoServico(dia(2026, time.September, 10))

	_, _ = fixos.Create(context.Background(), domain.NewRecurringEntry{
		UserID: usuario, Description: "Academia", AmountCents: 15990,
		Kind: domain.KindDespesa, Frequency: domain.FrequencyMensal,
		StartDate: dia(2026, time.January, 20),
	})

	resumo, _ := svc.Summary(context.Background(), usuario, 2026, time.September)

	// Hoje é dia 10: o fixo do dia 20 pesa no previsto, não no atual.
	if resumo.SaldoAtual != 0 {
		t.Errorf("saldo atual = %d, esperava 0", resumo.SaldoAtual)
	}
	if resumo.SaldoPrevisto != -15990 {
		t.Errorf("saldo previsto = %d, esperava -15990", resumo.SaldoPrevisto)
	}
}

func TestListaMisturaGravadosEProjetadosEmOrdemDeData(t *testing.T) {
	svc, _, fixos := novoServico(dia(2026, time.September, 30))

	criar(t, svc, "Mercado", 10000, domain.KindDespesa, dia(2026, time.September, 25))
	criar(t, svc, "Salário", 500000, domain.KindReceita, dia(2026, time.September, 5))

	_, _ = fixos.Create(context.Background(), domain.NewRecurringEntry{
		UserID: usuario, Description: "Academia", AmountCents: 15990,
		Kind: domain.KindDespesa, Frequency: domain.FrequencyMensal,
		StartDate: dia(2026, time.January, 20),
	})

	entradas, _ := svc.ListMonth(context.Background(), usuario, 2026, time.September)

	esperado := []int{5, 20, 25}
	if len(entradas) != len(esperado) {
		t.Fatalf("esperava %d lançamentos, veio %d", len(esperado), len(entradas))
	}

	for i, dia := range esperado {
		if entradas[i].OccurredAt.Day() != dia {
			t.Errorf("posição %d: dia = %d, esperava %d", i, entradas[i].OccurredAt.Day(), dia)
		}
	}
}

func TestOutroMesNaoVazaParaALista(t *testing.T) {
	svc, _, _ := novoServico(dia(2026, time.September, 30))

	criar(t, svc, "Agosto", 10000, domain.KindDespesa, dia(2026, time.August, 31))
	criar(t, svc, "Outubro", 10000, domain.KindDespesa, dia(2026, time.October, 1))
	criar(t, svc, "Setembro", 10000, domain.KindDespesa, dia(2026, time.September, 15))

	entradas, _ := svc.ListMonth(context.Background(), usuario, 2026, time.September)

	if len(entradas) != 1 || entradas[0].Description != "Setembro" {
		t.Errorf("esperava só o lançamento de setembro, veio %d", len(entradas))
	}
}

func TestApagarLancamentoInexistente(t *testing.T) {
	svc, _, _ := novoServico(dia(2026, time.September, 30))

	if err := svc.Delete(context.Background(), usuario, 999); err == nil {
		t.Error("esperava erro ao apagar lançamento inexistente")
	}
}
