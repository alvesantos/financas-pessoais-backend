package dto

import (
	"strings"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// CreateCategoryRequest é o corpo de POST /api/categories.
type CreateCategoryRequest struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Color string `json:"color"`
}

func (r CreateCategoryRequest) Validate() error {
	fields := map[string]string{}

	if strings.TrimSpace(r.Name) == "" {
		fields["name"] = "informe o nome da categoria"
	}
	if !domain.Kind(r.Kind).Valid() {
		fields["kind"] = "escolha um tipo de lançamento"
	}

	if len(fields) > 0 {
		return domain.ErrValidation.WithFields(fields)
	}
	return nil
}

func (r CreateCategoryRequest) ToDomain(userID int64) domain.NewCategory {
	return domain.NewCategory{
		UserID: userID,
		Name:   strings.TrimSpace(r.Name),
		Kind:   domain.Kind(r.Kind),
		Color:  strings.TrimSpace(r.Color),
	}
}

// CategoryResponse é uma categoria como o cliente a vê.
type CategoryResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	KindLabel string `json:"kind_label"`
	Color     string `json:"color"`
}

func NewCategoryResponse(category domain.Category) CategoryResponse {
	return CategoryResponse{
		ID:        category.ID,
		Name:      category.Name,
		Kind:      string(category.Kind),
		KindLabel: category.Kind.Label(),
		Color:     category.Color,
	}
}

func NewCategoryListResponse(categories []domain.Category) []CategoryResponse {
	list := make([]CategoryResponse, 0, len(categories))
	for _, category := range categories {
		list = append(list, NewCategoryResponse(category))
	}
	return list
}
