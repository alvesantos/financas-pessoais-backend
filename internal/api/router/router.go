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
	Transactions   domain.TransactionService
	Recurring      domain.RecurringService
	Categories     domain.CategoryService
	Debts          domain.DebtService
	Cards          domain.CreditCardService
	Clock          domain.Clock
	Dashboard      domain.DashboardService
	Tokens         domain.TokenIssuer
	DB             controller.Pinger
	AllowedOrigins []string
}

// New monta o roteador completo: rotas livres, rotas autenticadas e os
// middlewares globais.
func New(deps Deps) http.Handler {
	mux := http.NewServeMux()

	controllers := controllers{
		auth:         controller.NewAuthController(deps.Auth),
		health:       controller.NewHealthController(deps.DB),
		transactions: controller.NewTransactionController(deps.Transactions),
		recurring:    controller.NewRecurringController(deps.Recurring),
		categories:   controller.NewCategoryController(deps.Categories),
		debts:        controller.NewDebtController(deps.Debts, deps.Clock),
		cards:        controller.NewCreditCardController(deps.Cards),
		dashboard:    controller.NewDashboardController(deps.Dashboard),
	}

	registerPublicRoutes(mux, controllers)
	registerProtectedRoutes(mux, controllers, middleware.Authenticate(deps.Tokens))

	mux.HandleFunc("/", notFound)

	// O primeiro middleware é o mais externo: Recover envolve todos os demais.
	return middleware.Chain(mux,
		middleware.Recover,
		middleware.Logger,
		middleware.CORS(deps.AllowedOrigins),
	)
}

// controllers agrupa os controllers para as funções de registro não
// crescerem um parâmetro por rota nova.
type controllers struct {
	auth         *controller.AuthController
	health       *controller.HealthController
	transactions *controller.TransactionController
	recurring    *controller.RecurringController
	categories   *controller.CategoryController
	debts        *controller.DebtController
	cards        *controller.CreditCardController
	dashboard    *controller.DashboardController
}

// registerPublicRoutes: acessíveis sem token.
func registerPublicRoutes(mux *http.ServeMux, c controllers) {
	mux.HandleFunc("GET /api/health", c.health.Live)
	mux.HandleFunc("GET /api/health/ready", c.health.Ready)
	mux.HandleFunc("POST /api/auth/register", c.auth.Register)
	mux.HandleFunc("POST /api/auth/login", c.auth.Login)
}

// registerProtectedRoutes: exigem Bearer token válido. Cada rota é envolvida
// individualmente, então esquecer o middleware não é possível por descuido.
func registerProtectedRoutes(mux *http.ServeMux, c controllers, authenticated middleware.Middleware) {
	protect := func(pattern string, handler http.HandlerFunc) {
		mux.Handle(pattern, authenticated(handler))
	}

	protect("GET /api/auth/me", c.auth.Me)

	protect("GET /api/transactions", c.transactions.List)
	protect("POST /api/transactions", c.transactions.Create)
	protect("PUT /api/transactions/{id}", c.transactions.Update)
	protect("POST /api/transactions/occurrence", c.transactions.PayOccurrence)
	protect("GET /api/transactions/summary", c.transactions.Summary)
	protect("DELETE /api/transactions/{id}", c.transactions.Delete)

	protect("GET /api/recurring", c.recurring.List)
	protect("POST /api/recurring", c.recurring.Create)
	protect("PUT /api/recurring/{id}", c.recurring.Update)
	protect("DELETE /api/recurring/{id}", c.recurring.Delete)

	protect("GET /api/categories", c.categories.List)
	protect("POST /api/categories", c.categories.Create)
	protect("PUT /api/categories/{id}", c.categories.Update)
	protect("DELETE /api/categories/{id}", c.categories.Delete)

	protect("GET /api/debts", c.debts.List)
	protect("POST /api/debts", c.debts.Create)
	protect("PUT /api/debts/{id}", c.debts.Update)
	protect("POST /api/debts/{id}/amortize", c.debts.Amortize)
	protect("POST /api/debts/{id}/settle", c.debts.Settle)
	protect("DELETE /api/debts/{id}", c.debts.Delete)

	protect("GET /api/cards", c.cards.List)
	protect("POST /api/cards", c.cards.Create)
	protect("PUT /api/cards/{id}", c.cards.Update)
	protect("DELETE /api/cards/{id}", c.cards.Delete)

	protect("GET /api/dashboard", c.dashboard.Overview)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	response.Fail(w, r, domain.NewError(domain.CodeNotFound, "rota não encontrada"))
}
