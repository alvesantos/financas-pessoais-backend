package controller

import (
	"net/http"

	"github.com/alvesantos/financas-backend/internal/api/middleware"
	"github.com/alvesantos/financas-backend/internal/api/response"
	"github.com/alvesantos/financas-backend/internal/domain"
)

// currentUser lê o usuário autenticado do contexto. O segundo retorno é
// falso quando a resposta de erro já foi escrita — só acontece se a rota
// for registrada fora do grupo autenticado.
func currentUser(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := middleware.UserIDFrom(r.Context())
	if !ok {
		response.Fail(w, r, domain.ErrUnauthenticated)
		return 0, false
	}

	return userID, true
}
