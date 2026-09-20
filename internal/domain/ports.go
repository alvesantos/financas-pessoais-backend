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
