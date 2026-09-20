//go:build e2e

// Package e2e exercita a API de ponta a ponta: roteador real, servidor HTTP
// real e Postgres real. Não há mock em nenhuma camada.
//
// Rode com: make test-e2e
package e2e

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alvesantos/financas-backend/internal/api/controller"
	"github.com/alvesantos/financas-backend/internal/api/router"
	"github.com/alvesantos/financas-backend/internal/auth"
	"github.com/alvesantos/financas-backend/internal/database"
	"github.com/alvesantos/financas-backend/internal/repository/postgres"
	"github.com/alvesantos/financas-backend/internal/service"
)

// defaultTestDSN aponta para um banco separado do de desenvolvimento: a
// suíte apaga tabelas entre os testes e não pode tocar em dados reais.
const defaultTestDSN = "postgres://financas:financas@localhost:5434/financas_test?sslmode=disable"

const testJWTSecret = "segredo-de-teste-com-mais-de-32-caracteres"

var (
	server *httptest.Server
	pool   *pgxpool.Pool
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	if err := ensureDatabase(ctx, dsn); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: %v\n\nSuba o banco com: make db-up\n", err)
		os.Exit(1)
	}

	var err error
	pool, err = database.Connect(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: conectar ao banco de teste: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	server = httptest.NewServer(buildHandler(pool))
	defer server.Close()

	os.Exit(m.Run())
}

// buildHandler monta a mesma cadeia de dependências do cmd/api.
func buildHandler(pool *pgxpool.Pool) http.Handler {
	userRepository := postgres.NewUserRepository(pool)
	// Custo mínimo do bcrypt: a suíte testa o fluxo, não a criptografia.
	hasher := auth.NewBcryptHasher(4)
	tokens := auth.NewJWTIssuer(testJWTSecret, "financas-api", time.Hour)

	return router.New(router.Deps{
		Auth:           service.NewAuthService(userRepository, hasher, tokens),
		Tokens:         tokens,
		DB:             controller.Pinger(pool),
		AllowedOrigins: []string{"http://localhost:5173"},
	})
}

// ensureDatabase cria o banco de teste se ele ainda não existir.
func ensureDatabase(ctx context.Context, dsn string) error {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("TEST_DATABASE_URL inválida: %w", err)
	}

	dbName := parsed.Path[1:]
	if dbName == "" {
		return fmt.Errorf("TEST_DATABASE_URL não informa o nome do banco")
	}

	// Conecta ao banco administrativo para poder criar o de teste.
	admin := *parsed
	admin.Path = "/postgres"

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, admin.String())
	if err != nil {
		return fmt.Errorf("conectar ao Postgres em %s: %w", admin.Host, err)
	}
	defer conn.Close(ctx)

	var exists bool
	err = conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, dbName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("verificar banco de teste: %w", err)
	}

	if exists {
		return nil
	}

	// CREATE DATABASE não aceita parâmetro: o nome vem da configuração,
	// não de entrada do usuário.
	if _, err := conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %q", dbName)); err != nil {
		return fmt.Errorf("criar banco de teste %q: %w", dbName, err)
	}

	return nil
}

// resetDatabase deixa o banco vazio antes de cada teste.
func resetDatabase(t *testing.T) {
	t.Helper()

	_, err := pool.Exec(context.Background(),
		`TRUNCATE transactions, categories, accounts, users RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("limpar banco: %v", err)
	}
}
