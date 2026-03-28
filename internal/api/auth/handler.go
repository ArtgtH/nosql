package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	sessionHTTP "nosql/internal/api/session"
	authService "nosql/internal/service/auth"
	sessionService "nosql/internal/service/session"
	usersService "nosql/internal/service/users"
	"nosql/internal/transport"
)

type Handler struct {
	auth     *authService.Service
	sessions *sessionService.Service
	ttl      time.Duration
}

func NewHandler(
	auth *authService.Service,
	sessions *sessionService.Service,
	ttl time.Duration,
) *Handler {
	return &Handler{
		auth:     auth,
		sessions: sessions,
		ttl:      ttl,
	}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	sid := sessionHTTP.ReadSID(r)

	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusBadRequest, `invalid "body" field`)
		return
	}

	userID, err := h.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		h.refreshExistingSession(r, w, sid)

		switch {
		case errors.Is(err, usersService.ErrInvalidUsername),
			errors.Is(err, usersService.ErrInvalidPassword):
			transport.Message(w, r, http.StatusBadRequest, err.Error())
			return
		case errors.Is(err, authService.ErrInvalidCredentials):
			transport.Message(w, r, http.StatusUnauthorized, err.Error())
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	session, err := h.sessions.AttachUser(r.Context(), sid, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sessionHTTP.WriteSID(w, session.ID, h.ttl)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	sid := sessionHTTP.ReadSID(r)

	if err := h.sessions.Delete(r.Context(), sid); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sessionHTTP.DeleteSID(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) refreshExistingSession(r *http.Request, w http.ResponseWriter, sid string) {
	_, found, err := h.sessions.RefreshIfExists(r.Context(), sid)
	if err == nil && found {
		sessionHTTP.WriteSID(w, sid, h.ttl)
	}
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
