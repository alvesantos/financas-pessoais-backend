package service

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/alvesantos/financas-backend/internal/domain"
)

const maxCategoryNameLength = 40

// hexColor aceita só as cores no formato #rrggbb, que é o que a interface usa.
var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// CategoryService reúne os casos de uso de categorias.
type CategoryService struct {
	repository domain.CategoryRepository
}

var _ domain.CategoryService = (*CategoryService)(nil)

func NewCategoryService(repository domain.CategoryRepository) *CategoryService {
	return &CategoryService{repository: repository}
}

func (s *CategoryService) Create(ctx context.Context, input domain.NewCategory) (*domain.Category, error) {
	name, color, err := validateCategory(input.Name, input.Kind, input.Color)
	if err != nil {
		return nil, err
	}

	input.Name = name
	input.Color = color

	return s.repository.Create(ctx, input)
}

// Update reescreve a categoria, com as mesmas regras da criação.
func (s *CategoryService) Update(ctx context.Context, category domain.Category) (*domain.Category, error) {
	name, color, err := validateCategory(category.Name, category.Kind, category.Color)
	if err != nil {
		return nil, err
	}

	category.Name = name
	category.Color = color

	return s.repository.Update(ctx, category)
}

func validateCategory(rawName string, kind domain.Kind, rawColor string) (string, string, error) {
	fields := map[string]string{}

	name := strings.TrimSpace(rawName)
	if name == "" {
		fields["name"] = "informe o nome da categoria"
	}
	if utf8.RuneCountInString(name) > maxCategoryNameLength {
		fields["name"] = "o nome passou de 40 caracteres"
	}
	if !kind.Valid() {
		fields["kind"] = "escolha um tipo de lançamento"
	}

	color := strings.TrimSpace(rawColor)
	if color == "" {
		color = domain.DefaultCategoryColor
	}
	if !hexColor.MatchString(color) {
		fields["color"] = "use uma cor no formato #rrggbb"
	}

	if len(fields) > 0 {
		return "", "", domain.ErrValidation.WithFields(fields)
	}

	return name, strings.ToLower(color), nil
}

func (s *CategoryService) List(ctx context.Context, userID int64) ([]domain.Category, error) {
	return s.repository.List(ctx, userID)
}

func (s *CategoryService) Delete(ctx context.Context, userID, id int64) error {
	return s.repository.Delete(ctx, userID, id)
}
