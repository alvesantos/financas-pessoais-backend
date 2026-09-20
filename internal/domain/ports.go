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
	Update(ctx context.Context, input UpdateTransaction) (*Transaction, error)
	ListByPeriod(ctx context.Context, userID int64, period Period) ([]Transaction, error)
	// SumPaid soma o efeito no saldo de tudo que já foi pago ou recebido,
	// sem recorte de mês ou ano.
	SumPaid(ctx context.Context, userID int64) (int64, error)
	Delete(ctx context.Context, userID, id int64) error
}

// RecurringRepository é a porta de persistência de lançamentos fixos.
type RecurringRepository interface {
	Create(ctx context.Context, input NewRecurringEntry) (*RecurringEntry, error)
	Update(ctx context.Context, input UpdateRecurringEntry) (*RecurringEntry, error)
	FindByID(ctx context.Context, userID, id int64) (*RecurringEntry, error)
	ListActive(ctx context.Context, userID int64) ([]RecurringEntry, error)
	List(ctx context.Context, userID int64) ([]RecurringEntry, error)
	Delete(ctx context.Context, userID, id int64) error
}

// TransactionService é a porta de entrada dos casos de uso de lançamentos.
type TransactionService interface {
	Create(ctx context.Context, input NewTransaction) (*Transaction, error)
	Update(ctx context.Context, input UpdateTransaction) (*Transaction, error)
	// PayOccurrence transforma uma ocorrência projetada em lançamento pago.
	PayOccurrence(ctx context.Context, input PayOccurrence) (*Transaction, error)
	ListMonth(ctx context.Context, userID int64, year int, month time.Month) ([]Transaction, error)
	Summary(ctx context.Context, userID int64, year int, month time.Month) (*MonthSummary, error)
	Delete(ctx context.Context, userID, id int64) error
}

// RecurringService é a porta de entrada dos casos de uso de fixos.
type RecurringService interface {
	Create(ctx context.Context, input NewRecurringEntry) (*RecurringEntry, error)
	Update(ctx context.Context, input UpdateRecurringEntry) (*RecurringEntry, error)
	List(ctx context.Context, userID int64) ([]RecurringEntry, error)
	Delete(ctx context.Context, userID, id int64) error
}

// DashboardService monta as métricas do painel.
type DashboardService interface {
	Overview(ctx context.Context, userID int64, year int, month time.Month) (*Dashboard, error)
}

// DebtRepository é a porta de persistência de dívidas.
type DebtRepository interface {
	Create(ctx context.Context, input NewDebt) (*Debt, error)
	Save(ctx context.Context, debt Debt) (*Debt, error)
	FindByID(ctx context.Context, userID, id int64) (*Debt, error)
	List(ctx context.Context, userID int64) ([]Debt, error)
	Delete(ctx context.Context, userID, id int64) error
}

// DebtService é a porta de entrada dos casos de uso de dívidas.
type DebtService interface {
	Create(ctx context.Context, input NewDebt) (*Debt, error)
	Update(ctx context.Context, input UpdateDebt) (*Debt, error)
	Amortize(ctx context.Context, userID, id int64, input Amortization) (*Debt, error)
	// Settle quita a dívida. Com subtractFromBalance, o que faltava vira um
	// lançamento de verdade, para o saldo em carteira refletir a saída.
	Settle(ctx context.Context, userID, id int64, subtractFromBalance bool) (*Debt, error)
	List(ctx context.Context, userID int64) ([]Debt, error)
	Delete(ctx context.Context, userID, id int64) error
}

// CreditCardRepository é a porta de persistência de cartões.
type CreditCardRepository interface {
	Create(ctx context.Context, input NewCreditCard) (*CreditCard, error)
	Update(ctx context.Context, card CreditCard) (*CreditCard, error)
	List(ctx context.Context, userID int64) ([]CreditCard, error)
	FindByID(ctx context.Context, userID, id int64) (*CreditCard, error)
	Delete(ctx context.Context, userID, id int64) error
}

// CreditCardService é a porta de entrada dos casos de uso de cartões.
type CreditCardService interface {
	Create(ctx context.Context, input NewCreditCard) (*CreditCard, error)
	Update(ctx context.Context, card CreditCard) (*CreditCard, error)
	List(ctx context.Context, userID int64) ([]CreditCard, error)
	Delete(ctx context.Context, userID, id int64) error
}

// CategoryRepository é a porta de persistência de categorias.
type CategoryRepository interface {
	Create(ctx context.Context, input NewCategory) (*Category, error)
	Update(ctx context.Context, category Category) (*Category, error)
	FindByID(ctx context.Context, userID, id int64) (*Category, error)
	List(ctx context.Context, userID int64) ([]Category, error)
	Delete(ctx context.Context, userID, id int64) error
}

// CategoryService é a porta de entrada dos casos de uso de categorias.
type CategoryService interface {
	Create(ctx context.Context, input NewCategory) (*Category, error)
	Update(ctx context.Context, category Category) (*Category, error)
	List(ctx context.Context, userID int64) ([]Category, error)
	Delete(ctx context.Context, userID, id int64) error
}
