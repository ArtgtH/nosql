package events

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	sessionHTTP "nosql/internal/api/session"
	eventsService "nosql/internal/service/events"
	sessionService "nosql/internal/service/session"
	"nosql/internal/transport"
)

type Handler struct {
	events   *eventsService.Service
	sessions *sessionService.Service
	ttl      time.Duration
}

func NewHandler(
	events *eventsService.Service,
	sessions *sessionService.Service,
	ttl time.Duration,
) *Handler {
	return &Handler{
		events:   events,
		sessions: sessions,
		ttl:      ttl,
	}
}

type createEventRequest struct {
	Title       string `json:"title"`
	Address     string `json:"address"`
	StartedAt   string `json:"started_at"`
	FinishedAt  string `json:"finished_at"`
	Description string `json:"description"`
}

type createEventResponse struct {
	ID string `json:"id"`
}

type locationResponse struct {
	Address string `json:"address"`
}

type eventResponse struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
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

	session, found, err := h.sessions.Get(r.Context(), sid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !found || session.UserID == "" {
		h.refreshExistingSession(r, w, sid)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var req createEventRequest
	if err := decodeJSON(r, &req); err != nil {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusBadRequest, `invalid "body" field`)
		return
	}

	id, err := h.events.Create(
		r.Context(),
		session.UserID,
		req.Title,
		req.Address,
		req.StartedAt,
		req.FinishedAt,
		req.Description,
	)
	if err != nil {
		h.refreshExistingSession(r, w, sid)

		switch {
		case errors.Is(err, eventsService.ErrInvalidTitle),
			errors.Is(err, eventsService.ErrInvalidAddress),
			errors.Is(err, eventsService.ErrInvalidStartedAt),
			errors.Is(err, eventsService.ErrInvalidFinishedAt):
			transport.Message(w, r, http.StatusBadRequest, err.Error())
			return
		case errors.Is(err, eventsService.ErrEventExists):
			transport.Message(w, r, http.StatusConflict, err.Error())
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	h.refreshExistingSession(r, w, sid)
	transport.JSON(w, r, http.StatusCreated, createEventResponse{ID: id})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sid := sessionHTTP.ReadSID(r)

	limit, err := parseUintQuery(r, "limit")
	if err != nil {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusBadRequest, `invalid "limit" parameter`)
		return
	}

	offset, err := parseUintQuery(r, "offset")
	if err != nil {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusBadRequest, `invalid "offset" parameter`)
		return
	}

	events, err := h.events.List(
		r.Context(),
		r.URL.Query().Get("title"),
		limit,
		offset,
	)
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
			Description: event.Description,
			Location: locationResponse{
				Address: event.Location.Address,
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
