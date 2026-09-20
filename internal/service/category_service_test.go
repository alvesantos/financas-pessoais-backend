package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/alvesantos/financas-backend/internal/domain"
	"github.com/alvesantos/financas-backend/internal/service"
)

type fakeCategoryRepo struct {
	items  []domain.Category
	nextID int64
}

func (f *fakeCategoryRepo) Create(_ context.Context, input domain.NewCategory) (*domain.Category, error) {
	for _, item := range f.items {
		if item.Name == input.Name && item.Kind == input.Kind {
			return nil, domain.ErrCategoryTaken
		}
	}

	f.nextID++
	category := domain.Category{
		ID: f.nextID, UserID: input.UserID, Name: input.Name, Kind: input.Kind, Color: input.Color,
	}
	f.items = append(f.items, category)

	return &category, nil
}

func (f *fakeCategoryRepo) Update(_ context.Context, category domain.Category) (*domain.Category, error) {
	for i := range f.items {
		if f.items[i].ID == category.ID {
			f.items[i].Name = category.Name
			f.items[i].Kind = category.Kind
			f.items[i].Color = category.Color
			return &f.items[i], nil
		}
	}
	return nil, domain.ErrCategoryNotFound
}

func (f *fakeCategoryRepo) FindByID(_ context.Context, _, id int64) (*domain.Category, error) {
	for i := range f.items {
		if f.items[i].ID == id {
			return &f.items[i], nil
		}
	}
	return nil, domain.ErrCategoryNotFound
}

func (f *fakeCategoryRepo) List(_ context.Context, _ int64) ([]domain.Category, error) {
	return f.items, nil
}

func (f *fakeCategoryRepo) Delete(_ context.Context, _, id int64) error {
	for i, item := range f.items {
		if item.ID == id {
			f.items = append(f.items[:i], f.items[i+1:]...)
			return nil
		}
	}
	return domain.ErrCategoryNotFound
}

func TestCategoriaUsaCorPadraoQuandoNaoInformada(t *testing.T) {
	svc := service.NewCategoryService(&fakeCategoryRepo{})

	criada, err := svc.Create(context.Background(), domain.NewCategory{
		UserID: usuario, Name: "Mercado", Kind: domain.KindDespesa,
	})
	if err != nil {
		t.Fatalf("criar categoria: %v", err)
	}

	if criada.Color != domain.DefaultCategoryColor {
		t.Errorf("cor = %q, esperava %q", criada.Color, domain.DefaultCategoryColor)
	}
}

func TestCategoriaExigeNome(t *testing.T) {
	svc := service.NewCategoryService(&fakeCategoryRepo{})

	_, err := svc.Create(context.Background(), domain.NewCategory{
		UserID: usuario, Name: "   ", Kind: domain.KindDespesa,
	})

	appErr, ok := domain.AsError(err)
	if !ok || appErr.Fields["name"] == "" {
		t.Errorf("esperava erro de validação no campo name, veio %v", err)
	}
}

func TestCategoriaRecusaNomeLongoDemais(t *testing.T) {
	svc := service.NewCategoryService(&fakeCategoryRepo{})

	_, err := svc.Create(context.Background(), domain.NewCategory{
		UserID: usuario, Name: strings.Repeat("a", 41), Kind: domain.KindDespesa,
	})

	appErr, ok := domain.AsError(err)
	if !ok || appErr.Fields["name"] == "" {
		t.Errorf("esperava erro de validação no campo name, veio %v", err)
	}
}

func TestCategoriaRecusaCorForaDoFormato(t *testing.T) {
	svc := service.NewCategoryService(&fakeCategoryRepo{})

	_, err := svc.Create(context.Background(), domain.NewCategory{
		UserID: usuario, Name: "Mercado", Kind: domain.KindDespesa, Color: "vermelho",
	})

	appErr, ok := domain.AsError(err)
	if !ok || appErr.Fields["color"] == "" {
		t.Errorf("esperava erro de validação no campo color, veio %v", err)
	}
}

func TestCategoriaNormalizaNomeECor(t *testing.T) {
	svc := service.NewCategoryService(&fakeCategoryRepo{})

	criada, err := svc.Create(context.Background(), domain.NewCategory{
		UserID: usuario, Name: "  Mercado  ", Kind: domain.KindDespesa, Color: "#AABBCC",
	})
	if err != nil {
		t.Fatalf("criar categoria: %v", err)
	}

	if criada.Name != "Mercado" {
		t.Errorf("nome = %q, esperava %q", criada.Name, "Mercado")
	}
	if criada.Color != "#aabbcc" {
		t.Errorf("cor = %q, esperava %q", criada.Color, "#aabbcc")
	}
}

func TestCategoriaRecusaTipoInvalido(t *testing.T) {
	svc := service.NewCategoryService(&fakeCategoryRepo{})

	_, err := svc.Create(context.Background(), domain.NewCategory{
		UserID: usuario, Name: "Mercado", Kind: "pix",
	})

	appErr, ok := domain.AsError(err)
	if !ok || appErr.Fields["kind"] == "" {
		t.Errorf("esperava erro de validação no campo kind, veio %v", err)
	}
}
