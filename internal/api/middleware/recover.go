package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/alvesantos/financas-backend/internal/api/response"
	"github.com/alvesantos/financas-backend/internal/domain"
)

// Recover impede que um panic em um handler derrube o servidor inteiro.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.ErrorContext(r.Context(), "panic recuperado",
					"rota", r.URL.Path,
					"panic", recovered,
					"stack", string(debug.Stack()),
				)

				// A conexão é fechada: o cliente não deve reutilizá-la.
				w.Header().Set("Connection", "close")
				response.Fail(w, r, domain.ErrInternal)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
