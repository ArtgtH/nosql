package events

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
	reviewsService "nosql/internal/service/reviews"
	sessionService "nosql/internal/service/session"
	usersService "nosql/internal/service/users"
	"nosql/internal/transport"
)

type Handler struct {
	events    *eventsService.Service
	reactions *reactionsService.Service
	reviews   *reviewsService.Service
	users     *usersService.Service
	sessions  *sessionService.Service
	ttl       time.Duration
}

func NewHandler(
	events *eventsService.Service,
	reactions *reactionsService.Service,
	reviews *reviewsService.Service,
	users *usersService.Service,
	sessions *sessionService.Service,
	ttl time.Duration,
) *Handler {
	return &Handler{
		events:    events,
		reactions: reactions,
		reviews:   reviews,
		users:     users,
		sessions:  sessions,
		ttl:       ttl,
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
	City    string `json:"city,omitempty"`
}

type reactionsResponse struct {
	Likes    int `json:"likes"`
	Dislikes int `json:"dislikes"`
}

type reviewsSummaryResponse struct {
	Count  int     `json:"count"`
	Rating float64 `json:"rating"`
}

type eventResponse struct {
	ID          string                  `json:"id"`
	Title       string                  `json:"title"`
	Category    string                  `json:"category"`
	Price       uint64                  `json:"price"`
	Description string                  `json:"description"`
	Location    locationResponse        `json:"location"`
	CreatedAt   string                  `json:"created_at"`
	CreatedBy   string                  `json:"created_by"`
	StartedAt   string                  `json:"started_at"`
	FinishedAt  string                  `json:"finished_at"`
	Reactions   *reactionsResponse      `json:"reactions,omitempty"`
	Reviews     *reviewsSummaryResponse `json:"reviews,omitempty"`
}

type listEventsResponse struct {
	Events []eventResponse `json:"events"`
	Count  int             `json:"count"`
}

type createReviewRequest struct {
	Comment string `json:"comment"`
	Rating  *int8  `json:"rating"`
}

type createReviewResponse struct {
	ID string `json:"id"`
}

type reviewResponse struct {
	ID        string `json:"id"`
	EventID   string `json:"event_id"`
	Comment   string `json:"comment"`
	CreatedAt string `json:"created_at"`
	CreatedBy string `json:"created_by"`
	Rating    int8   `json:"rating"`
	UpdatedAt string `json:"updated_at"`
}

type listReviewsResponse struct {
	Reviews []reviewResponse `json:"reviews"`
	Count   int              `json:"count"`
}

func (h *Handler) HasReactions() bool {
	return h.reactions != nil
}

func (h *Handler) HasReviews() bool {
	return h.reviews != nil
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
	if err := decodeJSONStrict(r, &req); err != nil {
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
	includeReactions := includesReactions(r)
	includeReviews := includesReviews(r)

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

	createdBy := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if createdBy != "" {
		if _, err := primitive.ObjectIDFromHex(createdBy); err != nil {
			h.refreshExistingSession(r, w, sid)
			transport.Message(w, r, http.StatusBadRequest, `invalid "user_id" field`)
			return
		}
	}

	username := strings.TrimSpace(r.URL.Query().Get("user"))
	if username != "" {
		user, found, err := h.users.FindByUsername(r.Context(), username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !found {
			h.refreshExistingSession(r, w, sid)
			transport.JSON(w, r, http.StatusOK, listEventsResponse{Events: []eventResponse{}, Count: 0})
			return
		}
		if createdBy != "" && createdBy != user.ID.Hex() {
			h.refreshExistingSession(r, w, sid)
			transport.JSON(w, r, http.StatusOK, listEventsResponse{Events: []eventResponse{}, Count: 0})
			return
		}
		createdBy = user.ID.Hex()
	}

	events, err := h.events.List(r.Context(), eventsService.ListFilter{
		ID:              id,
		Title:           strings.TrimSpace(r.URL.Query().Get("title")),
		Category:        category,
		City:            strings.TrimSpace(r.URL.Query().Get("city")),
		CreatedBy:       createdBy,
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

	reactionsByTitle, err := h.resolveReactionsByTitle(r, events, includeReactions)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	reviewsByTitle, err := h.resolveReviewsByTitle(r, events, includeReviews)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := listEventsResponse{Events: make([]eventResponse, 0, len(events)), Count: len(events)}
	for _, event := range events {
		response.Events = append(response.Events, toEventResponse(event, includeReactions, reactionsByTitle[event.Title], includeReviews, reviewsByTitle[event.Title]))
	}

	h.refreshExistingSession(r, w, sid)
	transport.JSON(w, r, http.StatusOK, response)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	sid := sessionHTTP.ReadSID(r)
	includeReactions := includesReactions(r)
	includeReviews := includesReviews(r)

	event, found, err := h.events.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !found {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusNotFound, "Not found")
		return
	}

	counts := reactionsService.Counts{}
	if includeReactions && h.reactions != nil {
		counts, err = h.reactions.GetByTitle(r.Context(), event.Title)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	reviewCounts := reviewsService.Counts{}
	if includeReviews && h.reviews != nil {
		reviewCounts, err = h.reviews.GetByTitle(r.Context(), event.Title)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	h.refreshExistingSession(r, w, sid)
	transport.JSON(w, r, http.StatusOK, toEventResponse(event, includeReactions, counts, includeReviews, reviewCounts))
}

func (h *Handler) Patch(w http.ResponseWriter, r *http.Request) {
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

	patch, err := decodePatchRequest(r)
	if err != nil {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusBadRequest, `invalid "body" field`)
		return
	}

	err = h.events.Patch(r.Context(), chi.URLParam(r, "id"), session.UserID, patch)
	if err != nil {
		h.refreshExistingSession(r, w, sid)
		switch {
		case errors.Is(err, eventsService.ErrInvalidCategory),
			errors.Is(err, eventsService.ErrInvalidPrice),
			errors.Is(err, eventsService.ErrInvalidCity):
			transport.Message(w, r, http.StatusBadRequest, err.Error())
			return
		case errors.Is(err, eventsService.ErrEventNotFoundOrNotOrganizer):
			transport.Message(w, r, http.StatusNotFound, err.Error())
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	h.refreshExistingSession(r, w, sid)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Like(w http.ResponseWriter, r *http.Request) {
	h.react(w, r, true)
}

func (h *Handler) Dislike(w http.ResponseWriter, r *http.Request) {
	h.react(w, r, false)
}

func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) {
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
	if h.reviews == nil {
		h.refreshExistingSession(r, w, sid)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	var req createReviewRequest
	if err := decodeJSONStrict(r, &req); err != nil || req.Rating == nil {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusBadRequest, `invalid "body" field`)
		return
	}

	id, err := h.reviews.Create(r.Context(), chi.URLParam(r, "id"), session.UserID, req.Comment, *req.Rating)
	if err != nil {
		h.refreshExistingSession(r, w, sid)
		switch {
		case errors.Is(err, reviewsService.ErrInvalidComment):
			transport.Message(w, r, http.StatusBadRequest, err.Error())
			return
		case errors.Is(err, reviewsService.ErrInvalidRating):
			transport.Message(w, r, http.StatusBadRequest, err.Error())
			return
		case errors.Is(err, reviewsService.ErrAlreadyExists):
			transport.Message(w, r, http.StatusConflict, err.Error())
			return
		case errors.Is(err, reviewsService.ErrEventNotFound):
			transport.Message(w, r, http.StatusNotFound, err.Error())
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	h.refreshExistingSession(r, w, sid)
	transport.JSON(w, r, http.StatusCreated, createReviewResponse{ID: id})
}

func (h *Handler) ListReviews(w http.ResponseWriter, r *http.Request) {
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
	if h.reviews == nil {
		h.refreshExistingSession(r, w, sid)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	reviews, err := h.reviews.ListByEventID(r.Context(), chi.URLParam(r, "id"), limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := listReviewsResponse{Reviews: make([]reviewResponse, 0, len(reviews)), Count: len(reviews)}
	for _, review := range reviews {
		response.Reviews = append(response.Reviews, toReviewResponse(review))
	}

	h.refreshExistingSession(r, w, sid)
	transport.JSON(w, r, http.StatusOK, response)
}

func (h *Handler) UpdateReview(w http.ResponseWriter, r *http.Request) {
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
	if h.reviews == nil {
		h.refreshExistingSession(r, w, sid)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	patch, err := decodeReviewPatchRequest(r)
	if err != nil {
		h.refreshExistingSession(r, w, sid)
		transport.Message(w, r, http.StatusBadRequest, `invalid "body" field`)
		return
	}

	err = h.reviews.Update(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "review_id"), session.UserID, patch)
	if err != nil {
		h.refreshExistingSession(r, w, sid)
		switch {
		case errors.Is(err, reviewsService.ErrInvalidComment):
			transport.Message(w, r, http.StatusBadRequest, err.Error())
			return
		case errors.Is(err, reviewsService.ErrInvalidRating):
			transport.Message(w, r, http.StatusBadRequest, err.Error())
			return
		case errors.Is(err, reviewsService.ErrEventNotFound):
			transport.Message(w, r, http.StatusNotFound, err.Error())
			return
		case errors.Is(err, reviewsService.ErrReviewNotFound):
			transport.Message(w, r, http.StatusNotFound, err.Error())
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	h.refreshExistingSession(r, w, sid)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) react(w http.ResponseWriter, r *http.Request, like bool) {
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
	if h.reactions == nil {
		h.refreshExistingSession(r, w, sid)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	eventID := chi.URLParam(r, "id")
	if like {
		err = h.reactions.Like(r.Context(), eventID, session.UserID)
	} else {
		err = h.reactions.Dislike(r.Context(), eventID, session.UserID)
	}
	if err != nil {
		h.refreshExistingSession(r, w, sid)
		if errors.Is(err, reactionsService.ErrEventNotFound) {
			transport.Message(w, r, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.refreshExistingSession(r, w, sid)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) resolveReactionsByTitle(r *http.Request, events []eventsService.Event, includeReactions bool) (map[string]reactionsService.Counts, error) {
	result := make(map[string]reactionsService.Counts, len(events))
	if !includeReactions {
		return result, nil
	}
	if h.reactions == nil {
		return result, nil
	}

	titles := make([]string, 0, len(events))
	for _, event := range events {
		titles = append(titles, event.Title)
	}

	return h.reactions.GetByTitles(r.Context(), titles)
}

func (h *Handler) resolveReviewsByTitle(r *http.Request, events []eventsService.Event, includeReviews bool) (map[string]reviewsService.Counts, error) {
	result := make(map[string]reviewsService.Counts, len(events))
	if !includeReviews {
		return result, nil
	}
	if h.reviews == nil {
		return result, nil
	}

	titles := make([]string, 0, len(events))
	for _, event := range events {
		titles = append(titles, event.Title)
	}

	return h.reviews.GetByTitles(r.Context(), titles)
}

func (h *Handler) refreshExistingSession(r *http.Request, w http.ResponseWriter, sid string) {
	_, found, err := h.sessions.RefreshIfExists(r.Context(), sid)
	if err == nil && found {
		sessionHTTP.WriteSID(w, sid, h.ttl)
	}
}

func toEventResponse(event eventsService.Event, includeReactions bool, counts reactionsService.Counts, includeReviews bool, reviewCounts reviewsService.Counts) eventResponse {
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
	if includeReviews {
		response.Reviews = &reviewsSummaryResponse{Count: reviewCounts.Count, Rating: reviewCounts.Rating}
	}
	return response
}

func toReviewResponse(review reviewsService.Review) reviewResponse {
	return reviewResponse{
		ID:        review.ID,
		EventID:   review.EventID,
		Comment:   review.Comment,
		CreatedAt: review.CreatedAt.UTC().Format(time.RFC3339),
		CreatedBy: review.CreatedBy,
		Rating:    review.Rating,
		UpdatedAt: review.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func includesReactions(r *http.Request) bool {
	for _, part := range strings.Split(r.URL.Query().Get("include"), ",") {
		if strings.TrimSpace(part) == "reactions" {
			return true
		}
	}
	return false
}

func includesReviews(r *http.Request) bool {
	for _, part := range strings.Split(r.URL.Query().Get("include"), ",") {
		if strings.TrimSpace(part) == "reviews" {
			return true
		}
	}
	return false
}

func decodeJSONStrict(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func decodePatchRequest(r *http.Request) (eventsService.EventPatch, error) {
	raw := map[string]json.RawMessage{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&raw); err != nil {
		return eventsService.EventPatch{}, err
	}

	var patch eventsService.EventPatch

	if value, ok := raw["category"]; ok {
		var category string
		if err := json.Unmarshal(value, &category); err != nil {
			return eventsService.EventPatch{}, eventsService.ErrInvalidCategory
		}
		patch.Category = &category
	}

	if value, ok := raw["price"]; ok {
		var price uint64
		if err := json.Unmarshal(value, &price); err != nil {
			return eventsService.EventPatch{}, eventsService.ErrInvalidPrice
		}
		patch.Price = &price
	}

	if value, ok := raw["city"]; ok {
		var city string
		if err := json.Unmarshal(value, &city); err != nil {
			return eventsService.EventPatch{}, eventsService.ErrInvalidCity
		}
		patch.City = &city
	}

	for key := range raw {
		switch key {
		case "category", "price", "city":
		default:
			return eventsService.EventPatch{}, errors.New("unknown field")
		}
	}

	return patch, nil
}

func decodeReviewPatchRequest(r *http.Request) (reviewsService.Patch, error) {
	raw := map[string]json.RawMessage{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&raw); err != nil {
		return reviewsService.Patch{}, err
	}

	var patch reviewsService.Patch
	if value, ok := raw["comment"]; ok {
		var comment string
		if err := json.Unmarshal(value, &comment); err != nil {
			return reviewsService.Patch{}, reviewsService.ErrInvalidComment
		}
		patch.Comment = &comment
	}
	if value, ok := raw["rating"]; ok {
		var rating int8
		if err := json.Unmarshal(value, &rating); err != nil {
			return reviewsService.Patch{}, reviewsService.ErrInvalidRating
		}
		patch.Rating = &rating
	}
	for key := range raw {
		switch key {
		case "comment", "rating":
		default:
			return reviewsService.Patch{}, errors.New("unknown field")
		}
	}
	return patch, nil
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
