package users

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	sessionHTTP "nosql/internal/api/session"
	eventsService "nosql/internal/service/events"
	reactionsService "nosql/internal/service/reactions"
	sessionService "nosql/internal/service/session"
	usersService "nosql/internal/service/users"
	"nosql/internal/transport"
)

type Handler struct {
	users     *usersService.Service
	events    *eventsService.Service
	reactions *reactionsService.Service
	sessions  *sessionService.Service
	ttl       time.Duration
}

func NewHandler(
	users *usersService.Service,
	events *eventsService.Service,
	reactions *reactionsService.Service,
	sessions *sessionService.Service,
	ttl time.Duration,
) *Handler {
	return &Handler{
		users:     users,
		events:    events,
		reactions: reactions,
		sessions:  sessions,
		ttl:       ttl,
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

type reactionsResponse struct {
	Likes    int `json:"likes"`
	Dislikes int `json:"dislikes"`
}

type eventResponse struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Category    string             `json:"category"`
	Price       uint64             `json:"price"`
	Description string             `json:"description"`
	Location    locationResponse   `json:"location"`
	CreatedAt   string             `json:"created_at"`
	CreatedBy   string             `json:"created_by"`
	StartedAt   string             `json:"started_at"`
	FinishedAt  string             `json:"finished_at"`
	Reactions   *reactionsResponse `json:"reactions,omitempty"`
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

	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id != "" {
		if _, err := primitive.ObjectIDFromHex(id); err != nil {
			h.refreshExistingSession(r, w, sid)
			transport.Message(w, r, http.StatusBadRequest, `invalid "id" field`)
			return
		}
	}

	users, err := h.users.List(r.Context(), usersService.ListFilter{
		ID:     id,
		Name:   strings.TrimSpace(r.URL.Query().Get("name")),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := listUsersResponse{Users: make([]userResponse, 0, len(users)), Count: len(users)}
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
	includeReactions := includesReactions(r)

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

	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id != "" {
		if _, err := primitive.ObjectIDFromHex(id); err != nil {
			h.refreshExistingSession(r, w, sid)
			transport.Message(w, r, http.StatusBadRequest, `invalid "id" field`)
			return
		}
	}

	category := strings.TrimSpace(r.URL.Query().Get("category"))
	if category != "" && !isValidCategory(category) {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusBadRequest, `invalid "category" field`)
		return
	}

	priceFrom, err := parseOptionalUintQuery(r, "price_from")
	if err != nil {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusBadRequest, `invalid "price_from" field`)
		return
	}

	priceTo, err := parseOptionalUintQuery(r, "price_to")
	if err != nil {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusBadRequest, `invalid "price_to" field`)
		return
	}

	dateFrom, err := parseOptionalDateQuery(r, "date_from", "started_date_from")
	if err != nil {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusBadRequest, `invalid "date_from" field`)
		return
	}

	dateToExclusive, err := parseOptionalDateToExclusiveQuery(r, "date_to", "started_date_to")
	if err != nil {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusBadRequest, `invalid "date_to" field`)
		return
	}

	events, err := h.events.List(r.Context(), eventsService.ListFilter{
		ID:              id,
		Title:           strings.TrimSpace(r.URL.Query().Get("title")),
		Category:        category,
		City:            strings.TrimSpace(r.URL.Query().Get("city")),
		CreatedBy:       user.ID.Hex(),
		DateFrom:        dateFrom,
		DateToExclusive: dateToExclusive,
		PriceFrom:       priceFrom,
		PriceTo:         priceTo,
		Limit:           limit,
		Offset:          offset,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	reactionsByTitle := map[string]reactionsService.Counts{}
	if includeReactions && h.reactions != nil {
		titles := make([]string, 0, len(events))
		for _, event := range events {
			titles = append(titles, event.Title)
		}
		reactionsByTitle, err = h.reactions.GetByTitles(r.Context(), titles)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	response := listEventsResponse{Events: make([]eventResponse, 0, len(events)), Count: len(events)}
	for _, event := range events {
		response.Events = append(response.Events, toEventResponse(event, includeReactions, reactionsByTitle[event.Title]))
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
	return userResponse{ID: user.ID.Hex(), FullName: user.FullName, Username: user.Username}
}

func toEventResponse(event eventsService.Event, includeReactions bool, counts reactionsService.Counts) eventResponse {
	response := eventResponse{
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
	if includeReactions {
		response.Reactions = &reactionsResponse{Likes: counts.Likes, Dislikes: counts.Dislikes}
	}
	return response
}

func includesReactions(r *http.Request) bool {
	for _, part := range strings.Split(r.URL.Query().Get("include"), ",") {
		if strings.TrimSpace(part) == "reactions" {
			return true
		}
	}
	return false
}

func parseUintQuery(r *http.Request, name string) (uint64, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return 0, nil
	}
	return strconv.ParseUint(value, 10, 64)
}

func parseOptionalUintQuery(r *http.Request, name string) (*uint64, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func parseOptionalDateQuery(r *http.Request, names ...string) (string, error) {
	value := firstQueryValue(r, names...)
	if value == "" {
		return "", nil
	}

	parsed, err := time.Parse("20060102", value)
	if err != nil {
		return "", err
	}

	return parsed.Format("2006-01-02"), nil
}

func parseOptionalDateToExclusiveQuery(r *http.Request, names ...string) (string, error) {
	value := firstQueryValue(r, names...)
	if value == "" {
		return "", nil
	}

	parsed, err := time.Parse("20060102", value)
	if err != nil {
		return "", err
	}

	return parsed.AddDate(0, 0, 1).Format("2006-01-02"), nil
}

func firstQueryValue(r *http.Request, names ...string) string {
	for _, name := range names {
		value := strings.TrimSpace(r.URL.Query().Get(name))
		if value != "" {
			return value
		}
	}
	return ""
}

func isValidCategory(value string) bool {
	switch value {
	case "meetup", "concert", "exhibition", "party", "other":
		return true
	default:
		return false
	}
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
