package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alvesantos/financas-backend/internal/domain"
	"github.com/alvesantos/financas-backend/internal/service"
)

// As portas do domínio permitem testar os casos de uso sem banco nem HTTP.

type fakeRepository struct {
	byEmail   map[string]*domain.User
	createErr error
}

func (f *fakeRepository) Create(_ context.Context, input domain.NewUser) (*domain.User, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	user := &domain.User{ID: 1, Name: input.Name, Email: input.Email, PasswordHash: input.PasswordHash}
	f.byEmail[input.Email] = user
	return user, nil
}

func (f *fakeRepository) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	if user, ok := f.byEmail[email]; ok {
		return user, nil
	}
	return nil, domain.ErrUserNotFound
}

func (f *fakeRepository) FindByID(_ context.Context, id int64) (*domain.User, error) {
	for _, user := range f.byEmail {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

// plainHasher troca bcrypt por um prefixo: o teste não mede criptografia.
type plainHasher struct{}

func (plainHasher) Hash(password string) (string, error) { return "hash:" + password, nil }
func (plainHasher) Compare(hash, password string) bool   { return hash == "hash:"+password }

type stubIssuer struct{}

func (stubIssuer) Issue(*domain.User) (string, time.Time, error) {
	return "token-de-teste", time.Now().Add(time.Hour), nil
}

func (stubIssuer) Verify(string) (int64, error) { return 1, nil }

func newService(repo *fakeRepository) *service.AuthService {
	return service.NewAuthService(repo, plainHasher{}, stubIssuer{})
}

func newRepository() *fakeRepository {
	return &fakeRepository{byEmail: map[string]*domain.User{}}
}

func TestRegisterCriaSessao(t *testing.T) {
	svc := newService(newRepository())

	session, err := svc.Register(context.Background(), domain.Registration{
		Name: "Gabe", Email: "gabe@teste.com", Password: "senha12345",
	})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if session.Token == "" {
		t.Error("esperava um token na sessão")
	}
	if session.User.PasswordHash != "hash:senha12345" {
		t.Errorf("senha não foi hasheada: %q", session.User.PasswordHash)
	}
}

func TestRegisterEmailDuplicadoVemComCampo(t *testing.T) {
	repo := newRepository()
	repo.createErr = domain.ErrEmailTaken

	_, err := newService(repo).Register(context.Background(), domain.Registration{
		Name: "Gabe", Email: "gabe@teste.com", Password: "senha12345",
	})

	appErr, ok := domain.AsError(err)
	if !ok {
		t.Fatalf("esperava um domain.Error, veio %v", err)
	}
	if appErr.Code != domain.CodeConflict {
		t.Errorf("código = %q, esperava %q", appErr.Code, domain.CodeConflict)
	}
	if appErr.Fields["email"] == "" {
		t.Error("esperava o erro apontando o campo email")
	}
}

func TestLoginComSenhaErrada(t *testing.T) {
	repo := newRepository()
	svc := newService(repo)

	if _, err := svc.Register(context.Background(), domain.Registration{
		Name: "Gabe", Email: "gabe@teste.com", Password: "senha12345",
	}); err != nil {
		t.Fatalf("cadastro falhou: %v", err)
	}

	_, err := svc.Login(context.Background(), domain.Credentials{
		Email: "gabe@teste.com", Password: "errada",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("esperava ErrInvalidCredentials, veio %v", err)
	}
}

func TestLoginComEmailInexistenteNaoVazaExistencia(t *testing.T) {
	_, err := newService(newRepository()).Login(context.Background(), domain.Credentials{
		Email: "ninguem@teste.com", Password: "senha12345",
	})

	// A mesma mensagem da senha errada: a resposta não distingue os casos.
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("esperava ErrInvalidCredentials, veio %v", err)
	}
}

func TestCurrentUserComUsuarioRemovido(t *testing.T) {
	_, err := newService(newRepository()).CurrentUser(context.Background(), 42)

	if !errors.Is(err, domain.ErrUnauthenticated) {
		t.Errorf("esperava ErrUnauthenticated, veio %v", err)
	}
}
