package controller

import (
	"net/http"

	"github.com/alvesantos/financas-backend/internal/api/dto"
	"github.com/alvesantos/financas-backend/internal/api/request"
	"github.com/alvesantos/financas-backend/internal/api/response"
	"github.com/alvesantos/financas-backend/internal/domain"
)

// TransactionController expõe os lançamentos do mês.
type TransactionController struct {
	transactions domain.TransactionService
}

func NewTransactionController(transactions domain.TransactionService) *TransactionController {
	return &TransactionController{transactions: transactions}
}

// List devolve os lançamentos do mês. GET /api/transactions?year=&month=
func (c *TransactionController) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	year, month, err := request.YearMonth(r)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	entries, err := c.transactions.ListMonth(r.Context(), userID, year, month)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewTransactionListResponse(entries))
}

// Summary devolve os saldos do mês. GET /api/transactions/summary
func (c *TransactionController) Summary(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	year, month, err := request.YearMonth(r)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	summary, err := c.transactions.Summary(r.Context(), userID, year, month)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewMonthSummaryResponse(*summary))
}

// Create grava um lançamento. POST /api/transactions
func (c *TransactionController) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	body, err := request.DecodeJSON[dto.CreateTransactionRequest](r, w)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	created, err := c.transactions.Create(r.Context(), body.ToDomain(userID))
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewTransactionResponse(*created))
}

// Delete apaga um lançamento. DELETE /api/transactions/{id}
func (c *TransactionController) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	id, err := request.PathID(r, "id")
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	if err := c.transactions.Delete(r.Context(), userID, id); err != nil {
		response.Fail(w, r, err)
		return
	}

	response.NoContent(w)
}
