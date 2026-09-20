//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

func TestHealthLive(t *testing.T) {
	res := get(t, "/api/health", "")

	assertStatus(t, res, http.StatusOK)
	if got := res.String("status"); got != "ok" {
		t.Errorf("status = %q, esperava %q", got, "ok")
	}
}

func TestHealthReadyConfirmaOBanco(t *testing.T) {
	res := get(t, "/api/health/ready", "")

	assertStatus(t, res, http.StatusOK)
	if got := res.String("database"); got != "ok" {
		t.Errorf("database = %q, esperava %q", got, "ok")
	}
}

func TestRotaInexistenteResponde404EmJSON(t *testing.T) {
	res := get(t, "/api/nao-existe", "")

	assertStatus(t, res, http.StatusNotFound)
	if res.String("code") != "not_found" {
		t.Errorf("code = %q, esperava %q", res.String("code"), "not_found")
	}
}
