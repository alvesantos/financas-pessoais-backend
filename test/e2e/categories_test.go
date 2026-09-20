//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"
)

func criarCategoria(t *testing.T, token string, corpo map[string]any) apiResponse {
	t.Helper()
	return postAuth(t, "/api/categories", corpo, token)
}

func TestCriarECategoriaAparecerNaLista(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	res := criarCategoria(t, token, map[string]any{
		"name": "Mercado", "kind": "despesa", "color": "#aabbcc",
	})
	assertStatus(t, res, http.StatusCreated)

	if res.String("name") != "Mercado" {
		t.Errorf("nome = %q, esperava Mercado", res.String("name"))
	}
	if res.String("kind_label") != "Despesa" {
		t.Errorf("rótulo = %q, esperava Despesa", res.String("kind_label"))
	}

	status, lista := getList(t, "/api/categories", token)
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if len(lista) != 1 {
		t.Fatalf("esperava 1 categoria, veio %d", len(lista))
	}
}

func TestCategoriaAceitaOsQuatroTipos(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	for _, tipo := range []string{"receita", "despesa", "cartao_credito", "investimento"} {
		res := criarCategoria(t, token, map[string]any{"name": "Categoria " + tipo, "kind": tipo})

		assertStatus(t, res, http.StatusCreated)
	}
}

func TestCategoriaSemCorRecebeAPadrao(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	res := criarCategoria(t, token, map[string]any{"name": "Mercado", "kind": "despesa"})

	assertStatus(t, res, http.StatusCreated)
	if res.String("color") != "#6366f1" {
		t.Errorf("cor = %q, esperava a padrão #6366f1", res.String("color"))
	}
}

func TestCategoriaRecusaNomeRepetidoNoMesmoTipo(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criarCategoria(t, token, map[string]any{"name": "Mercado", "kind": "despesa"})
	repetida := criarCategoria(t, token, map[string]any{"name": "Mercado", "kind": "despesa"})

	assertStatus(t, repetida, http.StatusConflict)

	// O mesmo nome em outro tipo continua valendo.
	outroTipo := criarCategoria(t, token, map[string]any{"name": "Mercado", "kind": "receita"})
	assertStatus(t, outroTipo, http.StatusCreated)
}

func TestValidacaoDaCategoria(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	casos := []struct {
		nome      string
		corpo     map[string]any
		campoErro string
	}{
		{"sem nome", map[string]any{"name": "  ", "kind": "despesa"}, "name"},
		{"tipo desconhecido", map[string]any{"name": "Mercado", "kind": "pix"}, "kind"},
		{"cor fora do formato", map[string]any{"name": "Mercado", "kind": "despesa", "color": "vermelho"}, "color"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			res := criarCategoria(t, token, caso.corpo)

			assertStatus(t, res, http.StatusUnprocessableEntity)
			if res.Field(caso.campoErro) == "" {
				t.Errorf("esperava erro no campo %q, veio %v", caso.campoErro, res.Body)
			}
		})
	}
}

func TestApagarCategoria(t *testing.T) {
	resetDatabase(t)
	token := contaComToken(t)

	criada := criarCategoria(t, token, map[string]any{"name": "Mercado", "kind": "despesa"})
	id, _ := criada.Body["id"].(float64)

	assertStatus(t, del(t, fmt.Sprintf("/api/categories/%d", int64(id)), token), http.StatusNoContent)

	_, lista := getList(t, "/api/categories", token)
	if len(lista) != 0 {
		t.Errorf("esperava a lista vazia, veio %d", len(lista))
	}
}

func TestUmUsuarioNaoVeACategoriaDeOutro(t *testing.T) {
	resetDatabase(t)

	tokenA := contaComToken(t)
	tokenB := contaComToken(t)

	criarCategoria(t, tokenA, map[string]any{"name": "Mercado", "kind": "despesa"})

	_, listaDeB := getList(t, "/api/categories", tokenB)
	if len(listaDeB) != 0 {
		t.Errorf("o outro usuário não deveria ver nada, veio %d", len(listaDeB))
	}
}

func TestCategoriasExigemAutenticacao(t *testing.T) {
	casos := []struct{ metodo, rota string }{
		{http.MethodGet, "/api/categories"},
		{http.MethodPost, "/api/categories"},
	}

	for _, caso := range casos {
		t.Run(caso.metodo+" "+caso.rota, func(t *testing.T) {
			assertStatus(t, do(t, caso.metodo, caso.rota, nil, ""), http.StatusUnauthorized)
		})
	}
}
