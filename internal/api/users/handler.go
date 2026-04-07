package users

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	sessionHTTP "nosql/internal/api/session"
	sessionService "nosql/internal/service/session"
	usersService "nosql/internal/service/users"
	"nosql/internal/transport"
)

type Handler struct {
	users    *usersService.Service
	sessions *sessionService.Service
	ttl      time.Duration
}

func NewHandler(
	users *usersService.Service,
	sessions *sessionService.Service,
	ttl time.Duration,
) *Handler {
	return &Handler{
		users:    users,
		sessions: sessions,
		ttl:      ttl,
	}
}

type createUserRequest struct {
	FullName string `json:"full_name"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	sid := sessionHTTP.ReadSID(r)

	var req createUserRequest
	if err := decodeJSON(r, &req); err != nil {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusBadRequest, `invalid "body" field`)
		return
	}

	userID, err := h.users.Create(r.Context(), req.FullName, req.Username, req.Password)
	if err != nil {
		h.refreshExistingSession(r, w, sid)

		switch {
		case errors.Is(err, usersService.ErrInvalidFullName),
			errors.Is(err, usersService.ErrInvalidUsername),
			errors.Is(err, usersService.ErrInvalidPassword):
			transport.Message(w, r, http.StatusBadRequest, err.Error())
			return
		case errors.Is(err, usersService.ErrUserExists):
			transport.Message(w, r, http.StatusConflict, err.Error())
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	session, err := h.sessions.Create(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sessionHTTP.WriteSID(w, session.ID, h.ttl)
	w.WriteHeader(http.StatusCreated)
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
