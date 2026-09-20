package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// JSON escreve o payload serializado com o status informado.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if payload == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// Cabeçalho já enviado: só resta registrar.
		slog.Error("falha ao serializar resposta", "erro", err)
	}
}

// NoContent responde 204.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
