package router

import (
	"net/http"

	"github.com/alvesantos/financas-backend/internal/api/controller"
	"github.com/alvesantos/financas-backend/internal/api/middleware"
	"github.com/alvesantos/financas-backend/internal/api/response"
	"github.com/alvesantos/financas-backend/internal/domain"
)

// Deps são as dependências já construídas que o roteador distribui.
type Deps struct {
	Auth           domain.AuthService
	Tokens         domain.TokenIssuer
	DB             controller.Pinger
	AllowedOrigins []string
}

// New monta o roteador completo: rotas livres, rotas autenticadas e os
// middlewares globais.
func New(deps Deps) http.Handler {
	mux := http.NewServeMux()

	authController := controller.NewAuthController(deps.Auth)
	healthController := controller.NewHealthController(deps.DB)

	registerPublicRoutes(mux, authController, healthController)
	registerProtectedRoutes(mux, authController, middleware.Authenticate(deps.Tokens))

	mux.HandleFunc("/", notFound)

	// O primeiro middleware é o mais externo: Recover envolve todos os demais.
	return middleware.Chain(mux,
		middleware.Recover,
		middleware.Logger,
		middleware.CORS(deps.AllowedOrigins),
	)
}

// registerPublicRoutes: acessíveis sem token.
func registerPublicRoutes(
	mux *http.ServeMux,
	auth *controller.AuthController,
	health *controller.HealthController,
) {
	mux.HandleFunc("GET /api/health", health.Live)
	mux.HandleFunc("GET /api/health/ready", health.Ready)
	mux.HandleFunc("POST /api/auth/register", auth.Register)
	mux.HandleFunc("POST /api/auth/login", auth.Login)
}

// registerProtectedRoutes: exigem Bearer token válido. Cada rota é envolvida
// individualmente, então esquecer o middleware não é possível por descuido.
func registerProtectedRoutes(
	mux *http.ServeMux,
	auth *controller.AuthController,
	authenticated middleware.Middleware,
) {
	protect := func(pattern string, handler http.HandlerFunc) {
		mux.Handle(pattern, authenticated(handler))
	}

	protect("GET /api/auth/me", auth.Me)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	response.Fail(w, r, domain.NewError(domain.CodeNotFound, "rota não encontrada"))
}
