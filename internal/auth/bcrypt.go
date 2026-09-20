package auth

import (
	"golang.org/x/crypto/bcrypt"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// BcryptHasher implementa domain.PasswordHasher.
type BcryptHasher struct {
	cost int
}

var _ domain.PasswordHasher = (*BcryptHasher)(nil)

// NewBcryptHasher usa o custo padrão do bcrypt quando cost é zero.
func NewBcryptHasher(cost int) *BcryptHasher {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

func (h *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (h *BcryptHasher) Compare(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
