//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

func TestLoginComCredenciaisCorretas(t *testing.T) {
	resetDatabase(t)
	registerUser(t, "Gabe", "gabe@teste.com", "senha12345")

	res := post(t, "/api/auth/login", map[string]string{
		"email": "gabe@teste.com", "password": "senha12345",
	})

	assertStatus(t, res, http.StatusOK)
	res.Token(t)

	if res.String("expires_at") == "" {
		t.Error("esperava expires_at na resposta")
	}
}

func TestLoginNaoRevelaSeOEmailExiste(t *testing.T) {
	resetDatabase(t)
	registerUser(t, "Gabe", "gabe@teste.com", "senha12345")

	senhaErrada := post(t, "/api/auth/login", map[string]string{
		"email": "gabe@teste.com", "password": "senha-errada",
	})
	emailInexistente := post(t, "/api/auth/login", map[string]string{
		"email": "ninguem@teste.com", "password": "senha12345",
	})

	assertStatus(t, senhaErrada, http.StatusUnauthorized)
	assertStatus(t, emailInexistente, http.StatusUnauthorized)

	// As duas respostas precisam ser indistinguíveis: qualquer diferença
	// vira um oráculo para descobrir quais e-mails estão cadastrados.
	if senhaErrada.String("error") != emailInexistente.String("error") {
		t.Errorf("mensagens diferentes: %q e %q",
			senhaErrada.String("error"), emailInexistente.String("error"))
	}
	if senhaErrada.String("code") != emailInexistente.String("code") {
		t.Errorf("códigos diferentes: %q e %q",
			senhaErrada.String("code"), emailInexistente.String("code"))
	}
}

func TestLoginComCamposVazios(t *testing.T) {
	resetDatabase(t)

	res := post(t, "/api/auth/login", map[string]string{"email": "", "password": ""})

	// Credencial ausente é 401, não 422: validar em detalhe também vazaria informação.
	assertStatus(t, res, http.StatusUnauthorized)
}
