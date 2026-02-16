package health

import (
	"errors"
	"net/http"
	"nosql/internal/transport"
)

type HealthHandler struct{}

func (h HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	transport.JSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

func (h HealthHandler) Unhealth(w http.ResponseWriter, r *http.Request) {
	transport.ERROR(w, r, http.StatusBadRequest, errors.New("unhealthy"))
}
