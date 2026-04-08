package users

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	sessionHTTP "nosql/internal/api/session"
	eventsService "nosql/internal/service/events"
	sessionService "nosql/internal/service/session"
	usersService "nosql/internal/service/users"
	"nosql/internal/transport"
)

type Handler struct {
	users    *usersService.Service
	events   *eventsService.Service
	sessions *sessionService.Service
	ttl      time.Duration
}

func NewHandler(
	users *usersService.Service,
	events *eventsService.Service,
	sessions *sessionService.Service,
	ttl time.Duration,
) *Handler {
	return &Handler{
		users:    users,
		events:   events,
		sessions: sessions,
		ttl:      ttl,
	}
}

type createUserRequest struct {
	FullName string `json:"full_name"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type userResponse struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Username string `json:"username"`
}

type listUsersResponse struct {
	Users []userResponse `json:"users"`
	Count int            `json:"count"`
}

type locationResponse struct {
	Address string `json:"address"`
	City    string `json:"city,omitempty"`
}

type eventResponse struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Category    string           `json:"category"`
	Price       uint64           `json:"price"`
	Description string           `json:"description"`
	Location    locationResponse `json:"location"`
	CreatedAt   string           `json:"created_at"`
	CreatedBy   string           `json:"created_by"`
	StartedAt   string           `json:"started_at"`
	FinishedAt  string           `json:"finished_at"`
}

type listEventsResponse struct {
	Events []eventResponse `json:"events"`
	Count  int             `json:"count"`
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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sid := sessionHTTP.ReadSID(r)

	limit, err := parseUintQuery(r, "limit")
	if err != nil {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusBadRequest, `invalid "limit" field`)
		return
	}

	offset, err := parseUintQuery(r, "offset")
	if err != nil {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusBadRequest, `invalid "offset" field`)
		return
	}

	id := r.URL.Query().Get("id")
	if id != "" {
		if _, err := primitive.ObjectIDFromHex(id); err != nil {
			h.refreshExistingSession(r, w, sid)
			transport.Message(w, r, http.StatusBadRequest, `invalid "id" field`)
			return
		}
	}

	users, err := h.users.List(r.Context(), usersService.ListFilter{
		ID:     id,
		Name:   r.URL.Query().Get("name"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := listUsersResponse{
		Users: make([]userResponse, 0, len(users)),
		Count: len(users),
	}
	for _, user := range users {
		response.Users = append(response.Users, toUserResponse(user))
	}

	h.refreshExistingSession(r, w, sid)
	transport.JSON(w, r, http.StatusOK, response)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	sid := sessionHTTP.ReadSID(r)

	user, found, err := h.users.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !found {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusNotFound, "Not found")
		return
	}

	h.refreshExistingSession(r, w, sid)
	transport.JSON(w, r, http.StatusOK, toUserResponse(user))
}

func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	sid := sessionHTTP.ReadSID(r)

	user, found, err := h.users.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !found {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusNotFound, "User not found")
		return
	}

	events, err := h.events.List(r.Context(), eventsService.ListFilter{
		CreatedBy: user.ID.Hex(),
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := listEventsResponse{
		Events: make([]eventResponse, 0, len(events)),
		Count:  len(events),
	}
	for _, event := range events {
		response.Events = append(response.Events, eventResponse{
			ID:          event.ID.Hex(),
			Title:       event.Title,
			Category:    eventsService.NormalizeCategory(event.Category),
			Price:       event.Price,
			Description: event.Description,
			Location: locationResponse{
				Address: event.Location.Address,
				City:    event.Location.City,
			},
			CreatedAt:  event.CreatedAt,
			CreatedBy:  event.CreatedBy,
			StartedAt:  event.StartedAt,
			FinishedAt: event.FinishedAt,
		})
	}

	h.refreshExistingSession(r, w, sid)
	transport.JSON(w, r, http.StatusOK, response)
}

func (h *Handler) refreshExistingSession(r *http.Request, w http.ResponseWriter, sid string) {
	_, found, err := h.sessions.RefreshIfExists(r.Context(), sid)
	if err == nil && found {
		sessionHTTP.WriteSID(w, sid, h.ttl)
	}
}

func toUserResponse(user usersService.User) userResponse {
	return userResponse{
		ID:       user.ID.Hex(),
		FullName: user.FullName,
		Username: user.Username,
	}
}

func parseUintQuery(r *http.Request, name string) (uint64, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return 0, nil
	}
	return strconv.ParseUint(value, 10, 64)
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
