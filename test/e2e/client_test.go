//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// apiResponse é a resposta já lida, pronta para as asserções.
type apiResponse struct {
	StatusCode int
	Body       map[string]any
}

// Field devolve um erro de validação por campo, ou "" se não houver.
func (r apiResponse) Field(name string) string {
	fields, ok := r.Body["fields"].(map[string]any)
	if !ok {
		return ""
	}

	value, _ := fields[name].(string)
	return value
}

// String devolve um campo de topo do corpo como texto.
func (r apiResponse) String(key string) string {
	value, _ := r.Body[key].(string)
	return value
}

// Token extrai o token de uma resposta de sessão.
func (r apiResponse) Token(t *testing.T) string {
	t.Helper()

	token := r.String("token")
	if token == "" {
		t.Fatalf("esperava um token na resposta, veio %v", r.Body)
	}

	return token
}

// do executa uma requisição contra o servidor de teste.
func do(t *testing.T, method, path string, body any, token string) apiResponse {
	t.Helper()

	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("serializar corpo: %v", err)
		}
		payload = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, server.URL+path, payload)
	if err != nil {
		t.Fatalf("montar requisição: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("executar requisição: %v", err)
	}
	defer res.Body.Close()

	parsed := apiResponse{StatusCode: res.StatusCode, Body: map[string]any{}}

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("ler resposta: %v", err)
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &parsed.Body); err != nil {
			t.Fatalf("resposta não é JSON: %s", raw)
		}
	}

	return parsed
}

func post(t *testing.T, path string, body any) apiResponse {
	t.Helper()
	return do(t, http.MethodPost, path, body, "")
}

func get(t *testing.T, path, token string) apiResponse {
	t.Helper()
	return do(t, http.MethodGet, path, nil, token)
}

// registerUser cria uma conta e devolve o token da sessão.
func registerUser(t *testing.T, name, email, password string) string {
	t.Helper()

	res := post(t, "/api/auth/register", map[string]string{
		"name": name, "email": email, "password": password,
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("cadastro falhou: %d %v", res.StatusCode, res.Body)
	}

	return res.Token(t)
}

// assertStatus falha com o corpo da resposta, que costuma explicar o motivo.
func assertStatus(t *testing.T, res apiResponse, want int) {
	t.Helper()

	if res.StatusCode != want {
		t.Fatalf("status = %d, esperava %d (corpo: %v)", res.StatusCode, want, res.Body)
	}
}
