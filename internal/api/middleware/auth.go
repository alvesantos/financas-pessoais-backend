package middleware

import (
	"net/http"
	"strings"

	"github.com/alvesantos/financas-backend/internal/api/response"
	"github.com/alvesantos/financas-backend/internal/domain"
)

// Authenticate exige um Bearer token válido e injeta o ID do usuário no
// contexto. Usado apenas no grupo de rotas autenticadas.
func Authenticate(tokens domain.TokenIssuer) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			token := strings.TrimSpace(raw)

			if !found || token == "" {
				response.Fail(w, r, domain.ErrMissingToken)
				return
			}

			userID, err := tokens.Verify(token)
			if err != nil {
				response.Fail(w, r, domain.ErrUnauthenticated)
				return
			}

			next.ServeHTTP(w, r.WithContext(withUserID(r.Context(), userID)))
		})
	}
}
