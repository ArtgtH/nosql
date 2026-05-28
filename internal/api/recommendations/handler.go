package recommendations

import (
	"net/http"
	"time"

	sessionHTTP "nosql/internal/api/session"
	eventsService "nosql/internal/service/events"
	recommendationsService "nosql/internal/service/recommendations"
	sessionService "nosql/internal/service/session"
	"nosql/internal/transport"
)

type Handler struct {
	recommendations *recommendationsService.Service
	sessions        *sessionService.Service
	ttl             time.Duration
}

func NewHandler(
	recommendations *recommendationsService.Service,
	sessions *sessionService.Service,
	ttl time.Duration,
) *Handler {
	return &Handler{
		recommendations: recommendations,
		sessions:        sessions,
		ttl:             ttl,
	}
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

type listResponse struct {
	Events []eventResponse `json:"events"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
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

	events, err := h.recommendations.List(r.Context(), session.UserID)
	if err != nil {
		h.refreshExistingSession(r, w, sid)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := listResponse{Events: make([]eventResponse, 0, len(events))}
	for _, event := range events {
		response.Events = append(response.Events, toEventResponse(event))
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

func toEventResponse(event eventsService.Event) eventResponse {
	return eventResponse{
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
	}
}
