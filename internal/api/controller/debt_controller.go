package controller

import (
	"net/http"
	"time"

	"github.com/alvesantos/financas-backend/internal/api/dto"
	"github.com/alvesantos/financas-backend/internal/api/request"
	"github.com/alvesantos/financas-backend/internal/api/response"
	"github.com/alvesantos/financas-backend/internal/domain"
)

// DebtController expõe as dívidas parceladas.
type DebtController struct {
	debts domain.DebtService
	clock domain.Clock
}

func NewDebtController(debts domain.DebtService, clock domain.Clock) *DebtController {
	return &DebtController{debts: debts, clock: clock}
}

// List devolve as dívidas com o progresso de cada uma. GET /api/debts
func (c *DebtController) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	debts, err := c.debts.List(r.Context(), userID)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, c.withProgress(debts))
}

// Create registra uma dívida. POST /api/debts
func (c *DebtController) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	body, err := request.DecodeJSON[dto.CreateDebtRequest](r, w)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	created, err := c.debts.Create(r.Context(), body.ToDomain(userID))
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewDebtResponse(*created, created.Progress(c.today())))
}

// Update reescreve uma dívida. PUT /api/debts/{id}
func (c *DebtController) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	id, err := request.PathID(r, "id")
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	body, err := request.DecodeJSON[dto.CreateDebtRequest](r, w)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	updated, err := c.debts.Update(r.Context(), domain.UpdateDebt{ID: id, NewDebt: body.ToDomain(userID)})
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewDebtResponse(*updated, updated.Progress(c.today())))
}

// Amortize abate o saldo devedor. POST /api/debts/{id}/amortize
func (c *DebtController) Amortize(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	id, err := request.PathID(r, "id")
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	body, err := request.DecodeJSON[dto.AmortizeRequest](r, w)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	updated, err := c.debts.Amortize(r.Context(), userID, id, body.ToDomain())
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewDebtResponse(*updated, updated.Progress(c.today())))
}

// Settle quita a dívida. POST /api/debts/{id}/settle
func (c *DebtController) Settle(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	id, err := request.PathID(r, "id")
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	body, err := request.DecodeJSON[dto.SettleRequest](r, w)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	updated, err := c.debts.Settle(r.Context(), userID, id, body.SubtractFromBalance)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewDebtResponse(*updated, updated.Progress(c.today())))
}

// Delete apaga uma dívida. DELETE /api/debts/{id}
func (c *DebtController) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	id, err := request.PathID(r, "id")
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	if err := c.debts.Delete(r.Context(), userID, id); err != nil {
		response.Fail(w, r, err)
		return
	}

	response.NoContent(w)
}

func (c *DebtController) withProgress(debts []domain.Debt) []dto.DebtResponse {
	today := c.today()
	list := make([]dto.DebtResponse, 0, len(debts))

	for _, debt := range debts {
		list = append(list, dto.NewDebtResponse(debt, debt.Progress(today)))
	}

	return list
}

func (c *DebtController) today() time.Time {
	return c.clock.Today()
}
