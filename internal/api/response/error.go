package response

import (
	"log/slog"
	"net/http"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// ErrorBody é o formato único de erro da API.
type ErrorBody struct {
	Error  string            `json:"error"`
	Code   domain.ErrorCode  `json:"code"`
	Fields map[string]string `json:"fields,omitempty"`
}

// statusByCode traduz o código de domínio em status HTTP. Este é o único
// ponto do sistema onde as duas linguagens se encontram.
var statusByCode = map[domain.ErrorCode]int{
	domain.CodeValidation:     http.StatusUnprocessableEntity,
	domain.CodeInvalidPayload: http.StatusBadRequest,
	domain.CodeNotFound:       http.StatusNotFound,
	domain.CodeConflict:       http.StatusConflict,
	domain.CodeUnauthorized:   http.StatusUnauthorized,
	domain.CodeUnavailable:    http.StatusServiceUnavailable,
	domain.CodeInternal:       http.StatusInternalServerError,
}

// Fail traduz qualquer erro em uma resposta JSON. Erros internos viram uma
// mensagem genérica; os detalhes vão só para o log.
func Fail(w http.ResponseWriter, r *http.Request, err error) {
	appErr, ok := domain.AsError(err)
	if !ok {
		appErr = domain.ErrInternal.Wrap(err)
	}

	status, known := statusByCode[appErr.Code]
	if !known {
		status = http.StatusInternalServerError
	}

	if status >= http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "erro ao tratar requisição",
			"metodo", r.Method,
			"rota", r.URL.Path,
			"erro", appErr.Error(),
		)
	}

	JSON(w, status, ErrorBody{
		Error:  appErr.Message,
		Code:   appErr.Code,
		Fields: appErr.Fields,
	})
}
