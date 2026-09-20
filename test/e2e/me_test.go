//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"testing"
)

func TestMeComTokenValido(t *testing.T) {
	resetDatabase(t)
	token := registerUser(t, "Gabe", "gabe@teste.com", "senha12345")

	res := get(t, "/api/auth/me", token)

	assertStatus(t, res, http.StatusOK)
	if res.String("email") != "gabe@teste.com" {
		t.Errorf("email = %q, esperava %q", res.String("email"), "gabe@teste.com")
	}
	if res.String("name") != "Gabe" {
		t.Errorf("name = %q, esperava %q", res.String("name"), "Gabe")
	}
}

func TestMeSemToken(t *testing.T) {
	resetDatabase(t)

	res := get(t, "/api/auth/me", "")

	assertStatus(t, res, http.StatusUnauthorized)
	if res.String("code") != "unauthorized" {
		t.Errorf("code = %q, esperava %q", res.String("code"), "unauthorized")
	}
}

func TestMeComTokenInvalido(t *testing.T) {
	resetDatabase(t)

	casos := map[string]string{
		"texto qualquer":        "isso-nao-e-um-jwt",
		"assinatura adulterada": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIn0.assinatura-falsa",
		"token vazio":           "   ",
	}

	for nome, token := range casos {
		t.Run(nome, func(t *testing.T) {
			res := get(t, "/api/auth/me", token)
			assertStatus(t, res, http.StatusUnauthorized)
		})
	}
}

func TestMeComUsuarioRemovidoDepoisDoToken(t *testing.T) {
	resetDatabase(t)
	token := registerUser(t, "Gabe", "gabe@teste.com", "senha12345")

	_, err := pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, "gabe@teste.com")
	if err != nil {
		t.Fatalf("remover usuário: %v", err)
	}

	// O token continua assinado e no prazo, mas a sessão não vale mais.
	res := get(t, "/api/auth/me", token)

	assertStatus(t, res, http.StatusUnauthorized)
}

func TestFluxoCompletoDeCadastroELogin(t *testing.T) {
	resetDatabase(t)

	cadastro := post(t, "/api/auth/register", map[string]string{
		"name": "Gabe", "email": "gabe@teste.com", "password": "senha12345",
	})
	assertStatus(t, cadastro, http.StatusCreated)

	login := post(t, "/api/auth/login", map[string]string{
		"email": "gabe@teste.com", "password": "senha12345",
	})
	assertStatus(t, login, http.StatusOK)

	// O token do login abre a rota autenticada: o ciclo fecha.
	perfil := get(t, "/api/auth/me", login.Token(t))
	assertStatus(t, perfil, http.StatusOK)

	if perfil.String("email") != "gabe@teste.com" {
		t.Errorf("email = %q, esperava %q", perfil.String("email"), "gabe@teste.com")
	}
}
