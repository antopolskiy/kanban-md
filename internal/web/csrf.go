package web

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/antopolskiy/kanban-md/internal/clierr"
)

const (
	csrfCookie  = "kbmd_web"
	csrfBytes   = 32
	headerFetch = "Sec-Fetch-Site"
)

func (s *Server) protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		if !sameOrigin(r) || !csrfMatch(r) {
			http.Error(w, "cross-site request rejected", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func sameOrigin(r *http.Request) bool {
	switch r.Header.Get(headerFetch) {
	case "", "same-origin", "none":
	default:
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	return origin == "http://"+r.Host || origin == "https://"+r.Host
}

func csrfMatch(r *http.Request) bool {
	if err := r.ParseForm(); err != nil {
		return false
	}
	cookie, err := r.Cookie(csrfCookie)
	if err != nil || cookie.Value == "" {
		return false
	}
	return cookie.Value == r.PostForm.Get("csrf")
}

func (s *Server) csrfToken(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(csrfCookie); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	token := newToken()
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	return token
}

func newToken() string {
	buf := make([]byte, csrfBytes)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte("kanban-md-web-csrf-fallback-token"))
	}
	return hex.EncodeToString(buf)
}

func statusFor(err error) int {
	var cliErr *clierr.Error
	if errors.As(err, &cliErr) && cliErr.Code == clierr.TaskClaimed {
		return http.StatusConflict
	}
	return http.StatusBadRequest
}
