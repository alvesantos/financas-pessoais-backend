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
	fields := map[string]string{}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		fields["name"] = "informe o nome da categoria"
	}
	if utf8.RuneCountInString(name) > maxCategoryNameLength {
		fields["name"] = "o nome passou de 40 caracteres"
	}
	if !input.Kind.Valid() {
		fields["kind"] = "escolha um tipo de lançamento"
	}

	color := strings.TrimSpace(input.Color)
	if color == "" {
		color = domain.DefaultCategoryColor
	}
	if !hexColor.MatchString(color) {
		fields["color"] = "use uma cor no formato #rrggbb"
	}

	if len(fields) > 0 {
		return nil, domain.ErrValidation.WithFields(fields)
	}

	input.Name = name
	input.Color = strings.ToLower(color)

	return s.repository.Create(ctx, input)
}

func (s *CategoryService) List(ctx context.Context, userID int64) ([]domain.Category, error) {
	return s.repository.List(ctx, userID)
}

func (s *CategoryService) Delete(ctx context.Context, userID, id int64) error {
	return s.repository.Delete(ctx, userID, id)
}
