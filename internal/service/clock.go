package service

import (
	"time"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// SystemClock é o relógio de verdade. Os testes trocam por um relógio fixo.
type SystemClock struct{}

var _ domain.Clock = SystemClock{}

func (SystemClock) Today() time.Time {
	return domain.Day(time.Now().UTC())
}
