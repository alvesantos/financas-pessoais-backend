package request

import (
	"net/http"
	"strconv"
	"time"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// anoMinimo e anoMaximo evitam períodos absurdos, que só fariam o servidor
// trabalhar à toa.
const (
	anoMinimo = 2000
	anoMaximo = 2200
)

// PathID lê um identificador numérico da rota.
func PathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, domain.ErrValidation.WithFields(map[string]string{name: "identificador inválido"})
	}

	return id, nil
}

// YearMonth lê ?year= e ?month= da query. Sem eles, usa o mês corrente.
// Abrir a tela sem parâmetro nenhum mostra o mês de hoje.
func YearMonth(r *http.Request) (int, time.Month, error) {
	now := time.Now().UTC()
	query := r.URL.Query()

	year, err := intParam(query.Get("year"), now.Year())
	if err != nil || year < anoMinimo || year > anoMaximo {
		return 0, 0, domain.ErrValidation.WithFields(map[string]string{"year": "ano inválido"})
	}

	month, err := intParam(query.Get("month"), int(now.Month()))
	if err != nil || month < 1 || month > 12 {
		return 0, 0, domain.ErrValidation.WithFields(map[string]string{"month": "mês inválido"})
	}

	return year, time.Month(month), nil
}

func intParam(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}
