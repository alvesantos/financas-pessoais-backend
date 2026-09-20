package controller

import (
	"net/http"

	"github.com/alvesantos/financas-backend/internal/api/dto"
	"github.com/alvesantos/financas-backend/internal/api/request"
	"github.com/alvesantos/financas-backend/internal/api/response"
	"github.com/alvesantos/financas-backend/internal/domain"
)

// RecurringController expõe os lançamentos fixos.
type RecurringController struct {
	recurring domain.RecurringService
}

func NewRecurringController(recurring domain.RecurringService) *RecurringController {
	return &RecurringController{recurring: recurring}
}

// List devolve os fixos do usuário. GET /api/recurring
func (c *RecurringController) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	entries, err := c.recurring.List(r.Context(), userID)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewRecurringListResponse(entries))
}

// Create grava um fixo. POST /api/recurring
func (c *RecurringController) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	body, err := request.DecodeJSON[dto.CreateRecurringRequest](r, w)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	created, err := c.recurring.Create(r.Context(), body.ToDomain(userID))
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewRecurringResponse(*created))
}

// Delete apaga um fixo. DELETE /api/recurring/{id}
func (c *RecurringController) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	id, err := request.PathID(r, "id")
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	if err := c.recurring.Delete(r.Context(), userID, id); err != nil {
		response.Fail(w, r, err)
		return
	}

	response.NoContent(w)
}
