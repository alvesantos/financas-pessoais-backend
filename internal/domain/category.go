package domain

import "time"

// Category agrupa lançamentos de um mesmo tipo, como "Mercado" dentro de
// despesa. O nome é único por usuário e tipo.
type Category struct {
	ID        int64
	UserID    int64
	Name      string
	Kind      Kind
	Color     string
	CreatedAt time.Time
}

// NewCategory são os dados para criar uma categoria.
type NewCategory struct {
	UserID int64
	Name   string
	Kind   Kind
	Color  string
}

// DefaultCategoryColor é usada quando nenhuma cor é informada.
const DefaultCategoryColor = "#6366f1"

var (
	ErrCategoryNotFound = NewError(CodeNotFound, "categoria não encontrada")
	ErrCategoryTaken    = NewError(CodeConflict, "já existe uma categoria com esse nome para este tipo")
)
