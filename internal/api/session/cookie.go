package session

import (
	"net/http"
	"time"

	sessionService "nosql/internal/service/session"
)

func ReadSID(r *http.Request) string {
	cookie, err := r.Cookie(sessionService.CookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func WriteSID(w http.ResponseWriter, sid string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionService.CookieName,
		Value:    sid,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(ttl.Seconds()),
	})
}
