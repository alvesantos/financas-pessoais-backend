// Command api sobe a API de finanças pessoais.
//
// A montagem das dependências acontece toda aqui: cada camada recebe as
// portas de que precisa e nenhuma delas conhece quem a construiu.
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/alvesantos/financas-backend/internal/api/controller"
	"github.com/alvesantos/financas-backend/internal/api/router"
	"github.com/alvesantos/financas-backend/internal/auth"
	"github.com/alvesantos/financas-backend/internal/config"
	"github.com/alvesantos/financas-backend/internal/database"
	"github.com/alvesantos/financas-backend/internal/repository/postgres"
	"github.com/alvesantos/financas-backend/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("erro fatal: %v", err)
	}
}

func run() error {
	if err := config.LoadDotEnv(".env"); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.SetupLogger()

	// O contexto morre no primeiro SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	slog.Info("conectado ao Postgres")

	// Adaptadores de saída.
	userRepository := postgres.NewUserRepository(pool)
	transactionRepository := postgres.NewTransactionRepository(pool)
	recurringRepository := postgres.NewRecurringRepository(pool)
	categoryRepository := postgres.NewCategoryRepository(pool)
	hasher := auth.NewBcryptHasher(cfg.BcryptCost)
	tokens := auth.NewJWTIssuer(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTExpiration)

	// Casos de uso.
	clock := service.SystemClock{}
	authService := service.NewAuthService(userRepository, hasher, tokens)
	transactionService := service.NewTransactionService(transactionRepository, recurringRepository, clock)
	recurringService := service.NewRecurringService(recurringRepository)
	categoryService := service.NewCategoryService(categoryRepository)
	dashboardService := service.NewDashboardService(transactionRepository, recurringRepository, clock)

	// Adaptador de entrada.
	handler := router.New(router.Deps{
		Auth:           authService,
		Transactions:   transactionService,
		Recurring:      recurringService,
		Categories:     categoryService,
		Dashboard:      dashboardService,
		Tokens:         tokens,
		DB:             controller.Pinger(pool),
		AllowedOrigins: cfg.AllowedOrigins,
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("API no ar", "url", "http://localhost:"+cfg.Port, "ambiente", cfg.Env)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		stop()
	}

	// Desligamento gracioso: espera as requisições em voo terminarem.
	slog.Info("encerrando servidor...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	slog.Info("servidor encerrado")
	return nil
}
