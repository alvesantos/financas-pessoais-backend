package service

import (
	"context"
	"errors"
	"strings"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// AuthService concentra os casos de uso de autenticação. Depende apenas de
// portas do domínio, o que o mantém testável sem banco nem HTTP.
type AuthService struct {
	users  domain.UserRepository
	hasher domain.PasswordHasher
	tokens domain.TokenIssuer
}

var _ domain.AuthService = (*AuthService)(nil)

func NewAuthService(
	users domain.UserRepository,
	hasher domain.PasswordHasher,
	tokens domain.TokenIssuer,
) *AuthService {
	return &AuthService{users: users, hasher: hasher, tokens: tokens}
}

// Register cria a conta e já devolve uma sessão ativa.
func (s *AuthService) Register(ctx context.Context, input domain.Registration) (*domain.Session, error) {
	hash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	user, err := s.users.Create(ctx, domain.NewUser{
		Name:         strings.TrimSpace(input.Name),
		Email:        input.Email,
		PasswordHash: hash,
	})
	if err != nil {
		if errors.Is(err, domain.ErrEmailTaken) {
			return nil, domain.ErrEmailTaken.WithFields(map[string]string{
				"email": "e-mail já cadastrado",
			})
		}
		return nil, err
	}

	return s.newSession(user)
}

// Login valida as credenciais e devolve uma sessão.
func (s *AuthService) Login(ctx context.Context, input domain.Credentials) (*domain.Session, error) {
	user, err := s.users.FindByEmail(ctx, input.Email)
	if err != nil {
		// Mesma resposta para e-mail inexistente e senha errada: não revela
		// quais e-mails estão cadastrados.
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	if !s.hasher.Compare(user.PasswordHash, input.Password) {
		return nil, domain.ErrInvalidCredentials
	}

	return s.newSession(user)
}

// CurrentUser devolve o usuário de uma sessão já autenticada.
func (s *AuthService) CurrentUser(ctx context.Context, userID int64) (*domain.User, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		// O token é válido, mas o usuário sumiu: a sessão não vale mais.
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUnauthenticated
		}
		return nil, err
	}

	return user, nil
}

func (s *AuthService) newSession(user *domain.User) (*domain.Session, error) {
	token, expiresAt, err := s.tokens.Issue(user)
	if err != nil {
		return nil, domain.ErrInternal.Wrap(err)
	}

	return &domain.Session{Token: token, ExpiresAt: expiresAt, User: user}, nil
}
