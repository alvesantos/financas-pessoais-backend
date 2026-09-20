//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"testing"
)

func TestRegisterCriaUsuarioNoBanco(t *testing.T) {
	resetDatabase(t)

	res := post(t, "/api/auth/register", map[string]string{
		"name": "Gabe", "email": "gabe@teste.com", "password": "senha12345",
	})

	assertStatus(t, res, http.StatusCreated)
	res.Token(t)

	user, ok := res.Body["user"].(map[string]any)
	if !ok {
		t.Fatalf("esperava o usuário na resposta, veio %v", res.Body)
	}
	if user["email"] != "gabe@teste.com" {
		t.Errorf("email = %v, esperava %q", user["email"], "gabe@teste.com")
	}

	// O efeito no banco é parte do contrato: a senha precisa estar hasheada.
	var name, hash string
	err := pool.QueryRow(context.Background(),
		`SELECT name, password_hash FROM users WHERE email = $1`, "gabe@teste.com",
	).Scan(&name, &hash)
	if err != nil {
		t.Fatalf("usuário não foi persistido: %v", err)
	}

	if name != "Gabe" {
		t.Errorf("name no banco = %q, esperava %q", name, "Gabe")
	}
	if hash == "senha12345" || hash == "" {
		t.Errorf("senha não foi hasheada: %q", hash)
	}
}

func TestRegisterNuncaDevolveOHashDaSenha(t *testing.T) {
	resetDatabase(t)

	res := post(t, "/api/auth/register", map[string]string{
		"name": "Gabe", "email": "gabe@teste.com", "password": "senha12345",
	})

	assertStatus(t, res, http.StatusCreated)

	user, _ := res.Body["user"].(map[string]any)
	for _, campo := range []string{"password", "password_hash", "PasswordHash"} {
		if _, presente := user[campo]; presente {
			t.Errorf("a resposta expôs o campo %q", campo)
		}
	}
}

func TestRegisterNormalizaOEmail(t *testing.T) {
	resetDatabase(t)

	res := post(t, "/api/auth/register", map[string]string{
		"name": "Gabe", "email": "  GABE@Teste.COM  ", "password": "senha12345",
	})

	assertStatus(t, res, http.StatusCreated)

	// E-mail normalizado permite login com qualquer caixa depois.
	login := post(t, "/api/auth/login", map[string]string{
		"email": "gabe@teste.com", "password": "senha12345",
	})
	assertStatus(t, login, http.StatusOK)
}

func TestRegisterRecusaEmailDuplicado(t *testing.T) {
	resetDatabase(t)
	registerUser(t, "Gabe", "gabe@teste.com", "senha12345")

	res := post(t, "/api/auth/register", map[string]string{
		"name": "Outro", "email": "gabe@teste.com", "password": "outrasenha1",
	})

	assertStatus(t, res, http.StatusConflict)
	if res.Field("email") == "" {
		t.Error("esperava o erro apontando o campo email")
	}
}

func TestRegisterValidaOsCampos(t *testing.T) {
	resetDatabase(t)

	casos := []struct {
		nome      string
		corpo     map[string]string
		campoErro string
	}{
		{
			nome:      "nome curto demais",
			corpo:     map[string]string{"name": "G", "email": "gabe@teste.com", "password": "senha12345"},
			campoErro: "name",
		},
		{
			nome:      "e-mail sem formato válido",
			corpo:     map[string]string{"name": "Gabe", "email": "não-é-email", "password": "senha12345"},
			campoErro: "email",
		},
		{
			nome:      "senha com menos de 8 caracteres",
			corpo:     map[string]string{"name": "Gabe", "email": "gabe@teste.com", "password": "curta"},
			campoErro: "password",
		},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			res := post(t, "/api/auth/register", caso.corpo)

			assertStatus(t, res, http.StatusUnprocessableEntity)
			if res.Field(caso.campoErro) == "" {
				t.Errorf("esperava erro no campo %q, veio %v", caso.campoErro, res.Body)
			}
		})
	}
}

func TestRegisterRecusaCorpoInvalido(t *testing.T) {
	resetDatabase(t)

	// Campo desconhecido costuma ser typo no cliente: rejeitar é melhor que ignorar.
	res := post(t, "/api/auth/register", map[string]string{
		"name": "Gabe", "email": "gabe@teste.com", "password": "senha12345", "admin": "true",
	})

	assertStatus(t, res, http.StatusBadRequest)
	if res.String("code") != "invalid_payload" {
		t.Errorf("code = %q, esperava %q", res.String("code"), "invalid_payload")
	}
}
