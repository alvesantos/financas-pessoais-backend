package controller

import (
	"net/http"

	"github.com/alvesantos/financas-backend/internal/api/dto"
	"github.com/alvesantos/financas-backend/internal/api/request"
	"github.com/alvesantos/financas-backend/internal/api/response"
	"github.com/alvesantos/financas-backend/internal/domain"
)

// AuthController traduz HTTP em casos de uso. Nenhuma regra de negócio mora
// aqui: ele decodifica, delega e serializa.
type AuthController struct {
	auth domain.AuthService
}

func NewAuthController(auth domain.AuthService) *AuthController {
	return &AuthController{auth: auth}
}

// Register cria uma conta. POST /api/auth/register
func (c *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	body, err := request.DecodeJSON[dto.RegisterRequest](r, w)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	session, err := c.auth.Register(r.Context(), body.ToDomain())
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewSessionResponse(session))
}

// Login autentica o usuário. POST /api/auth/login
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	body, err := request.DecodeJSON[dto.LoginRequest](r, w)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	session, err := c.auth.Login(r.Context(), body.ToDomain())
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewSessionResponse(session))
}

// Me devolve o usuário autenticado. GET /api/auth/me
func (c *AuthController) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	user, err := c.auth.CurrentUser(r.Context(), userID)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewUserResponse(user))
}
