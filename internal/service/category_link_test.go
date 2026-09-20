package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/alvesantos/financas-backend/internal/domain"
	"github.com/alvesantos/financas-backend/internal/service"
)

// categoriaDe cria uma categoria já persistida no repositório falso.
func categoriaDe(t *testing.T, repo *fakeCategoryRepo, nome string, tipo domain.Kind) *domain.Category {
	t.Helper()

	criada, err := repo.Create(context.Background(), domain.NewCategory{
		UserID: usuario, Name: nome, Kind: tipo, Color: "#aabbcc",
	})
	if err != nil {
		t.Fatalf("criar categoria: %v", err)
	}

	return criada
}

func servicoComCategorias(hoje time.Time) (*service.TransactionService, *fakeCategoryRepo) {
	categorias := &fakeCategoryRepo{}
	svc := service.NewTransactionService(
		&fakeTransactionRepo{}, &fakeRecurringRepo{}, &fakeDebtRepo{}, categorias, relogioFixo{hoje: hoje},
	)

	return svc, categorias
}

func TestLancamentoAceitaCategoriaDoMesmoTipo(t *testing.T) {
	svc, categorias := servicoComCategorias(dia(2026, time.September, 20))
	categoria := categoriaDe(t, categorias, "Mercado", domain.KindDespesa)

	criado, err := svc.Create(context.Background(), domain.NewTransaction{
		UserID: usuario, AmountCents: 8550, Kind: domain.KindDespesa,
		OccurredAt: dia(2026, time.September, 10), CategoryID: &categoria.ID,
	})
	if err != nil {
		t.Fatalf("criar lançamento: %v", err)
	}

	if criado.CategoryID == nil || *criado.CategoryID != categoria.ID {
		t.Errorf("categoria = %v, esperava %d", criado.CategoryID, categoria.ID)
	}
}

func TestLancamentoSemCategoriaContinuaValendo(t *testing.T) {
	svc, _ := servicoComCategorias(dia(2026, time.September, 20))

	criado, err := svc.Create(context.Background(), domain.NewTransaction{
		UserID: usuario, AmountCents: 8550, Kind: domain.KindDespesa,
		OccurredAt: dia(2026, time.September, 10),
	})
	if err != nil {
		t.Fatalf("criar lançamento: %v", err)
	}

	if criado.CategoryID != nil {
		t.Errorf("esperava nenhuma categoria, veio %v", criado.CategoryID)
	}
}

func TestLancamentoRecusaCategoriaInexistente(t *testing.T) {
	svc, _ := servicoComCategorias(dia(2026, time.September, 20))
	inexistente := int64(999)

	_, err := svc.Create(context.Background(), domain.NewTransaction{
		UserID: usuario, AmountCents: 8550, Kind: domain.KindDespesa,
		OccurredAt: dia(2026, time.September, 10), CategoryID: &inexistente,
	})

	appErr, ok := domain.AsError(err)
	if !ok || appErr.Fields["category_id"] == "" {
		t.Errorf("esperava erro de validação no campo category_id, veio %v", err)
	}
}

func TestLancamentoRecusaCategoriaDeOutroTipo(t *testing.T) {
	svc, categorias := servicoComCategorias(dia(2026, time.September, 20))
	categoria := categoriaDe(t, categorias, "Salário", domain.KindReceita)

	// Categoria de receita em uma despesa produziria um agrupamento sem
	// sentido no painel.
	_, err := svc.Create(context.Background(), domain.NewTransaction{
		UserID: usuario, AmountCents: 8550, Kind: domain.KindDespesa,
		OccurredAt: dia(2026, time.September, 10), CategoryID: &categoria.ID,
	})

	appErr, ok := domain.AsError(err)
	if !ok || appErr.Fields["category_id"] == "" {
		t.Errorf("esperava erro de validação no campo category_id, veio %v", err)
	}
}

func TestFixoAceitaCategoria(t *testing.T) {
	categorias := &fakeCategoryRepo{}
	svc := service.NewRecurringService(&fakeRecurringRepo{}, categorias)
	categoria := categoriaDe(t, categorias, "Academia", domain.KindDespesa)

	criado, err := svc.Create(context.Background(), domain.NewRecurringEntry{
		UserID: usuario, Description: "Academia", AmountCents: 15990,
		Kind: domain.KindDespesa, Frequency: domain.FrequencyMensal,
		StartDate: dia(2026, time.January, 20), CategoryID: &categoria.ID,
	})
	if err != nil {
		t.Fatalf("criar fixo: %v", err)
	}

	if criado.CategoryID == nil || *criado.CategoryID != categoria.ID {
		t.Errorf("categoria = %v, esperava %d", criado.CategoryID, categoria.ID)
	}
}

func TestFixoRecusaCategoriaDeOutroTipo(t *testing.T) {
	categorias := &fakeCategoryRepo{}
	svc := service.NewRecurringService(&fakeRecurringRepo{}, categorias)
	categoria := categoriaDe(t, categorias, "Salário", domain.KindReceita)

	_, err := svc.Create(context.Background(), domain.NewRecurringEntry{
		UserID: usuario, AmountCents: 15990, Kind: domain.KindDespesa,
		Frequency: domain.FrequencyMensal, StartDate: dia(2026, time.January, 20),
		CategoryID: &categoria.ID,
	})

	appErr, ok := domain.AsError(err)
	if !ok || appErr.Fields["category_id"] == "" {
		t.Errorf("esperava erro de validação no campo category_id, veio %v", err)
	}
}

func TestProjecaoDoFixoCarregaACategoria(t *testing.T) {
	fixos := &fakeRecurringRepo{}
	categorias := &fakeCategoryRepo{}
	svc := service.NewTransactionService(
		&fakeTransactionRepo{}, fixos, &fakeDebtRepo{}, categorias, relogioFixo{hoje: dia(2026, time.September, 30)},
	)

	nome, cor := "Academia", "#aabbcc"
	id := int64(7)
	fixos.items = append(fixos.items, domain.RecurringEntry{
		ID: 1, UserID: usuario, Description: "Academia", AmountCents: 15990,
		Kind: domain.KindDespesa, Frequency: domain.FrequencyMensal,
		StartDate: dia(2026, time.January, 20), Active: true,
		CategoryID: &id, CategoryName: &nome, CategoryColor: &cor,
	})

	entradas, err := svc.ListMonth(context.Background(), usuario, 2026, time.September)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}

	if len(entradas) != 1 {
		t.Fatalf("esperava 1 projeção, veio %d", len(entradas))
	}
	if entradas[0].CategoryName == nil || *entradas[0].CategoryName != nome {
		t.Errorf("a projeção precisa carregar a categoria do fixo, veio %v", entradas[0].CategoryName)
	}
}
