package health

import (
	"net/http"
	sessionHTTP "nosql/internal/api/session"
	sessionService "nosql/internal/service/session"
	"nosql/internal/transport"
	"time"
)

type Handler struct {
	ttl time.Duration
}

func NewHandler(ttl time.Duration) *Handler {
	return &Handler{ttl: ttl}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	sid := sessionHTTP.ReadSID(r)
	if sessionService.IsValidSID(sid) {
		sessionHTTP.WriteSID(w, sid, h.ttl)
	}

	transport.JSON(w, r, http.StatusOK, map[string]string{
		"status": "ok",
	})
}
