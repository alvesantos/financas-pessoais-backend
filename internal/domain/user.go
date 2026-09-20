package domain

import "time"

// User é o dono de uma carteira de finanças pessoais.
type User struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewUser são os dados necessários para criar um usuário.
type NewUser struct {
	Name         string
	Email        string
	PasswordHash string
}

// Credentials são as credenciais de login já normalizadas.
type Credentials struct {
	Email    string
	Password string
}

// Registration são os dados de cadastro já validados.
type Registration struct {
	Name     string
	Email    string
	Password string
}

// Session é o resultado de um login ou cadastro bem-sucedido.
type Session struct {
	Token     string
	ExpiresAt time.Time
	User      *User
}
