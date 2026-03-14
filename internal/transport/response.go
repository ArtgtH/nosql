package transport

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

func JSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	render.Status(r, status)
	render.JSON(w, r, v)
}

func ERROR(w http.ResponseWriter, r *http.Request, status int, err error) {
	reqID := middleware.GetReqID(r.Context())

	render.Status(r, status)
	render.JSON(w, r, map[string]any{
		"error": map[string]any{
			"message":    err.Error(),
			"request_id": reqID,
		},
	})
}
