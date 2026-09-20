package controller

import (
	"net/http"

	"github.com/alvesantos/financas-backend/internal/api/dto"
	"github.com/alvesantos/financas-backend/internal/api/request"
	"github.com/alvesantos/financas-backend/internal/api/response"
	"github.com/alvesantos/financas-backend/internal/domain"
)

// CreditCardController expõe os cartões de crédito.
type CreditCardController struct {
	cards domain.CreditCardService
}

func NewCreditCardController(cards domain.CreditCardService) *CreditCardController {
	return &CreditCardController{cards: cards}
}

// List devolve os cartões do usuário. GET /api/cards
func (c *CreditCardController) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	cards, err := c.cards.List(r.Context(), userID)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewCreditCardListResponse(cards))
}

// Create cadastra um cartão. POST /api/cards
func (c *CreditCardController) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	body, err := request.DecodeJSON[dto.CreditCardRequest](r, w)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	created, err := c.cards.Create(r.Context(), body.ToDomain(userID))
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewCreditCardResponse(*created))
}

// Update reescreve um cartão. PUT /api/cards/{id}
func (c *CreditCardController) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	id, err := request.PathID(r, "id")
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	body, err := request.DecodeJSON[dto.CreditCardRequest](r, w)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	card := domain.CreditCard{
		ID:              id,
		UserID:          userID,
		Name:            body.Name,
		LimitCents:      body.LimitCents,
		BestPurchaseDay: body.BestPurchaseDay,
		DueDay:          body.DueDay,
	}

	updated, err := c.cards.Update(r.Context(), card)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewCreditCardResponse(*updated))
}

// Delete apaga um cartão. DELETE /api/cards/{id}
func (c *CreditCardController) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	id, err := request.PathID(r, "id")
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	if err := c.cards.Delete(r.Context(), userID, id); err != nil {
		response.Fail(w, r, err)
		return
	}

	response.NoContent(w)
}
