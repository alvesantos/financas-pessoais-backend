package controller

import (
	"net/http"

	"github.com/alvesantos/financas-backend/internal/api/dto"
	"github.com/alvesantos/financas-backend/internal/api/request"
	"github.com/alvesantos/financas-backend/internal/api/response"
	"github.com/alvesantos/financas-backend/internal/domain"
)

// DashboardController expõe as métricas do painel.
type DashboardController struct {
	dashboard domain.DashboardService
}

func NewDashboardController(dashboard domain.DashboardService) *DashboardController {
	return &DashboardController{dashboard: dashboard}
}

// Overview devolve o painel. GET /api/dashboard?year=&month=
func (c *DashboardController) Overview(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	year, month, err := request.YearMonth(r)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	overview, err := c.dashboard.Overview(r.Context(), userID, year, month)
	if err != nil {
		response.Fail(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewDashboardResponse(*overview))
}
