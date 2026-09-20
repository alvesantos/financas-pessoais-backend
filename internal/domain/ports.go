package domain

import (
	"context"
	"time"
)

// UserRepository é a porta de persistência de usuários. Implementada em
// internal/repository/postgres.
type UserRepository interface {
	Create(ctx context.Context, user NewUser) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id int64) (*User, error)
}

// PasswordHasher isola o algoritmo de hash de senha.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}

// TokenIssuer emite e valida os tokens de acesso.
type TokenIssuer interface {
	Issue(user *User) (token string, expiresAt time.Time, err error)
	Verify(token string) (userID int64, err error)
}

// AuthService é a porta de entrada dos casos de uso de autenticação,
// consumida pelos controllers.
type AuthService interface {
	Register(ctx context.Context, input Registration) (*Session, error)
	Login(ctx context.Context, input Credentials) (*Session, error)
	CurrentUser(ctx context.Context, userID int64) (*User, error)
}

// Clock isola "hoje" do relógio real, para que o saldo atual seja testável.
type Clock interface {
	Today() time.Time
}

// TransactionRepository é a porta de persistência de lançamentos.
type TransactionRepository interface {
	Create(ctx context.Context, input NewTransaction) (*Transaction, error)
	ListByPeriod(ctx context.Context, userID int64, period Period) ([]Transaction, error)
	// SumUntil soma o efeito no saldo de tudo que ocorreu até a data, sem
	// recorte de mês ou ano.
	SumUntil(ctx context.Context, userID int64, until time.Time) (int64, error)
	Delete(ctx context.Context, userID, id int64) error
}

// RecurringRepository é a porta de persistência de lançamentos fixos.
type RecurringRepository interface {
	Create(ctx context.Context, input NewRecurringEntry) (*RecurringEntry, error)
	ListActive(ctx context.Context, userID int64) ([]RecurringEntry, error)
	List(ctx context.Context, userID int64) ([]RecurringEntry, error)
	Delete(ctx context.Context, userID, id int64) error
}

// TransactionService é a porta de entrada dos casos de uso de lançamentos.
type TransactionService interface {
	Create(ctx context.Context, input NewTransaction) (*Transaction, error)
	ListMonth(ctx context.Context, userID int64, year int, month time.Month) ([]Transaction, error)
	Summary(ctx context.Context, userID int64, year int, month time.Month) (*MonthSummary, error)
	Delete(ctx context.Context, userID, id int64) error
}

// RecurringService é a porta de entrada dos casos de uso de fixos.
type RecurringService interface {
	Create(ctx context.Context, input NewRecurringEntry) (*RecurringEntry, error)
	List(ctx context.Context, userID int64) ([]RecurringEntry, error)
	Delete(ctx context.Context, userID, id int64) error
}

// DashboardService monta as métricas do painel.
type DashboardService interface {
	Overview(ctx context.Context, userID int64, year int, month time.Month) (*Dashboard, error)
}

// CategoryRepository é a porta de persistência de categorias.
type CategoryRepository interface {
	Create(ctx context.Context, input NewCategory) (*Category, error)
	FindByID(ctx context.Context, userID, id int64) (*Category, error)
	List(ctx context.Context, userID int64) ([]Category, error)
	Delete(ctx context.Context, userID, id int64) error
}

// CategoryService é a porta de entrada dos casos de uso de categorias.
type CategoryService interface {
	Create(ctx context.Context, input NewCategory) (*Category, error)
	List(ctx context.Context, userID int64) ([]Category, error)
	Delete(ctx context.Context, userID, id int64) error
}
