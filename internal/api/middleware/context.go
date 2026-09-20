package middleware

import "context"

type contextKey string

const userIDKey contextKey = "auth.userID"

// withUserID guarda o usuário autenticado no contexto da requisição.
func withUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFrom recupera o usuário autenticado. O segundo retorno é falso em
// rotas que não passaram pelo middleware Authenticate.
func UserIDFrom(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}
