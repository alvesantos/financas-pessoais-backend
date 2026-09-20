//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

func TestCORSLiberaApenasOrigensConfiguradas(t *testing.T) {
	casos := []struct {
		nome     string
		origem   string
		liberada bool
	}{
		{nome: "origem do frontend", origem: "http://localhost:5173", liberada: true},
		{nome: "origem desconhecida", origem: "http://site-malicioso.com", liberada: false},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodOptions, server.URL+"/api/auth/login", nil)
			if err != nil {
				t.Fatalf("montar requisição: %v", err)
			}
			req.Header.Set("Origin", caso.origem)
			req.Header.Set("Access-Control-Request-Method", http.MethodPost)

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("executar requisição: %v", err)
			}
			defer res.Body.Close()

			liberada := res.Header.Get("Access-Control-Allow-Origin") != ""
			if liberada != caso.liberada {
				t.Errorf("Access-Control-Allow-Origin = %q, esperava liberada=%v",
					res.Header.Get("Access-Control-Allow-Origin"), caso.liberada)
			}
		})
	}
}
