package session

import (
	"net/http"
	"nosql/internal/transport"
	"time"

	sessionService "nosql/internal/service/session"
)

type Handler struct {
	service *sessionService.Service
	ttl     time.Duration
}

func NewHandler(service *sessionService.Service, ttl time.Duration) *Handler {
	return &Handler{
		service: service,
		ttl:     ttl,
	}
}

func (h *Handler) CreateOrRefresh(w http.ResponseWriter, r *http.Request) {
	sid := ReadSID(r)

	result, err := h.service.Upsert(r.Context(), sid)
	if err != nil {
		transport.ERROR(w, r, http.StatusInternalServerError, err)
		return
	}

	WriteSID(w, result.Session.ID, h.ttl)

	if result.Created {
		w.WriteHeader(http.StatusCreated)
		return
	}

	w.WriteHeader(http.StatusOK)
}
