package domain

import "errors"

// ErrorCode classifica a falha em termos de negócio. A camada HTTP traduz
// cada código em um status; o domínio não conhece HTTP.
type ErrorCode string

const (
	CodeValidation     ErrorCode = "validation"
	CodeNotFound       ErrorCode = "not_found"
	CodeConflict       ErrorCode = "conflict"
	CodeUnauthorized   ErrorCode = "unauthorized"
	CodeInternal       ErrorCode = "internal"
	CodeUnavailable    ErrorCode = "unavailable"
	CodeInvalidPayload ErrorCode = "invalid_payload"
)

// Error é o erro de aplicação. Message é sempre segura para o cliente;
// a causa interna fica em cause e só vai para o log.
type Error struct {
	Code    ErrorCode
	Message string
	Fields  map[string]string
	cause   error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return e.Message + ": " + e.cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.cause }

// Is compara pelo código, para que errors.Is funcione com as sentinelas
// mesmo quando o erro foi copiado por WithFields ou Wrap.
func (e *Error) Is(target error) bool {
	var t *Error
	return errors.As(target, &t) && t.Code == e.Code && t.Message == e.Message
}

// NewError cria um erro de aplicação.
func NewError(code ErrorCode, message string) *Error {
	return &Error{Code: code, Message: message}
}

// clone evita que as sentinelas compartilhadas sejam mutadas.
func (e *Error) clone() *Error {
	copied := *e

	if e.Fields != nil {
		copied.Fields = make(map[string]string, len(e.Fields))
		for k, v := range e.Fields {
			copied.Fields[k] = v
		}
	}

	return &copied
}

// WithFields devolve uma cópia com os erros por campo.
func (e *Error) WithFields(fields map[string]string) *Error {
	copied := e.clone()
	copied.Fields = fields
	return copied
}

// Wrap devolve uma cópia carregando a causa interna.
func (e *Error) Wrap(cause error) *Error {
	copied := e.clone()
	copied.cause = cause
	return copied
}

// AsError extrai um *Error de qualquer erro da cadeia.
func AsError(err error) (*Error, bool) {
	var appErr *Error
	ok := errors.As(err, &appErr)
	return appErr, ok
}

// Sentinelas usadas por serviços e repositórios.
var (
	ErrUserNotFound        = NewError(CodeNotFound, "usuário não encontrado")
	ErrTransactionNotFound = NewError(CodeNotFound, "lançamento não encontrado")
	ErrRecurringNotFound   = NewError(CodeNotFound, "lançamento fixo não encontrado")
	ErrEmailTaken          = NewError(CodeConflict, "e-mail já cadastrado")
	ErrInvalidCredentials  = NewError(CodeUnauthorized, "e-mail ou senha incorretos")
	ErrUnauthenticated     = NewError(CodeUnauthorized, "token inválido ou expirado")
	ErrMissingToken        = NewError(CodeUnauthorized, "token de acesso ausente")
	ErrInvalidPayload      = NewError(CodeInvalidPayload, "corpo da requisição inválido")
	ErrValidation          = NewError(CodeValidation, "dados inválidos")
	ErrInternal            = NewError(CodeInternal, "erro interno do servidor")
)
