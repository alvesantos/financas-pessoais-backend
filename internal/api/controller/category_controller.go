package controller

import (
	"net/http"

	"github.com/alvesantos/financas-backend/internal/api/dto"
	"github.com/alvesantos/financas-backend/internal/api/request"
	"github.com/alvesantos/financas-backend/internal/api/response"
	"github.com/alvesantos/financas-backend/internal/domain"
)

// CategoryController expõe as categorias.
type CategoryController struct {
	categories domain.CategoryService
}

func NewCategoryController(categories domain.CategoryService) *CategoryController {
	return &CategoryController{categories: categories}
}

// List devolve as categorias do usuário. GET /api/categories
func (c *CategoryController) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	categories, err := c.categories.List(r.Context(), userID)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewCategoryListResponse(categories))
}

// Create cria uma categoria. POST /api/categories
func (c *CategoryController) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	body, err := request.DecodeJSON[dto.CreateCategoryRequest](r, w)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	created, err := c.categories.Create(r.Context(), body.ToDomain(userID))
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewCategoryResponse(*created))
}

// Update reescreve uma categoria. PUT /api/categories/{id}
func (c *CategoryController) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	id, err := request.PathID(r, "id")
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	body, err := request.DecodeJSON[dto.CreateCategoryRequest](r, w)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	category := domain.Category{
		ID:     id,
		UserID: userID,
		Name:   body.Name,
		Kind:   domain.Kind(body.Kind),
		Color:  body.Color,
	}

	updated, err := c.categories.Update(r.Context(), category)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewCategoryResponse(*updated))
}

// Delete apaga uma categoria. DELETE /api/categories/{id}
func (c *CategoryController) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	id, err := request.PathID(r, "id")
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	if err := c.categories.Delete(r.Context(), userID, id); err != nil {
		response.Fail(w, r, err)
		return
	}

	response.NoContent(w)
}
