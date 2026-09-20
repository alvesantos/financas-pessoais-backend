package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/alvesantos/financas-backend/internal/api/response"
	"github.com/alvesantos/financas-backend/internal/domain"
)

// Pinger é satisfeita pelo pool do Postgres. Declarada aqui, no consumidor,
// para que o controller não dependa do driver.
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthController responde aos checks de liveness e readiness.
type HealthController struct {
	db Pinger
}

func NewHealthController(db Pinger) *HealthController {
	return &HealthController{db: db}
}

// Live diz apenas que o processo está no ar. GET /api/health
func (c *HealthController) Live(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready confirma que as dependências respondem. GET /api/health/ready
func (c *HealthController) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := c.db.Ping(ctx); err != nil {
		response.Fail(w, r, domain.NewError(domain.CodeUnavailable, "banco de dados indisponível").Wrap(err))
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "ready", "database": "ok"})
}
