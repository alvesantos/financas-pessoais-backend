package dto

import (
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alvesantos/financas-backend/internal/domain"
)

const minPasswordLength = 8

// RegisterRequest é o corpo de POST /api/auth/register.
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate checa o formato dos campos antes de chegar ao serviço.
func (r RegisterRequest) Validate() error {
	fields := map[string]string{}

	if utf8.RuneCountInString(strings.TrimSpace(r.Name)) < 2 {
		fields["name"] = "informe seu nome"
	}
	if !validEmail(r.Email) {
		fields["email"] = "e-mail inválido"
	}
	if utf8.RuneCountInString(r.Password) < minPasswordLength {
		fields["password"] = "a senha precisa de ao menos 8 caracteres"
	}

	if len(fields) > 0 {
		return domain.ErrValidation.WithFields(fields)
	}
	return nil
}

// ToDomain converte o corpo validado no input do caso de uso.
func (r RegisterRequest) ToDomain() domain.Registration {
	return domain.Registration{
		Name:     strings.TrimSpace(r.Name),
		Email:    strings.TrimSpace(r.Email),
		Password: r.Password,
	}
}

// LoginRequest é o corpo de POST /api/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate exige apenas presença: credenciais erradas não viram erro de
// validação, para não vazar quais e-mails existem.
func (r LoginRequest) Validate() error {
	if strings.TrimSpace(r.Email) == "" || r.Password == "" {
		return domain.ErrInvalidCredentials
	}
	return nil
}

func (r LoginRequest) ToDomain() domain.Credentials {
	return domain.Credentials{
		Email:    strings.TrimSpace(r.Email),
		Password: r.Password,
	}
}

// UserResponse é a projeção pública de um usuário. O hash nunca aparece aqui.
type UserResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func NewUserResponse(user *domain.User) UserResponse {
	return UserResponse{ID: user.ID, Name: user.Name, Email: user.Email}
}

// SessionResponse é a resposta de cadastro e login.
type SessionResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      UserResponse `json:"user"`
}

func NewSessionResponse(session *domain.Session) SessionResponse {
	return SessionResponse{
		Token:     session.Token,
		ExpiresAt: session.ExpiresAt,
		User:      NewUserResponse(session.User),
	}
}

func validEmail(email string) bool {
	trimmed := strings.TrimSpace(email)
	addr, err := mail.ParseAddress(trimmed)
	return err == nil && addr.Address == trimmed
}
