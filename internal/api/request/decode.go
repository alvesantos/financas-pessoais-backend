package request

import (
	"encoding/json"
	"net/http"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// maxBodyBytes limita o corpo da requisição a 1 MiB.
const maxBodyBytes = 1 << 20

// Validatable é implementada pelos DTOs que se validam sozinhos.
type Validatable interface {
	Validate() error
}

// DecodeJSON lê, valida e devolve o corpo da requisição. Campos desconhecidos
// são rejeitados, para que um typo no cliente não passe despercebido.
func DecodeJSON[T Validatable](r *http.Request, w http.ResponseWriter) (T, error) {
	var payload T

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&payload); err != nil {
		return payload, domain.ErrInvalidPayload.Wrap(err)
	}

	// Um segundo objeto no corpo indica um cliente mal comportado.
	if decoder.More() {
		return payload, domain.ErrInvalidPayload
	}

	if err := payload.Validate(); err != nil {
		return payload, err
	}

	return payload, nil
}
