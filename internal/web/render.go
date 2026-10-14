package web

import (
	"embed"
	"html/template"
	"net/http"
)

//go:embed templates static
var assets embed.FS

type templates struct {
	page *template.Template
}

func parseTemplates() (*templates, error) {
	page, err := template.ParseFS(assets, "templates/page.html")
	if err != nil {
		return nil, err
	}
	return &templates{page: page}, nil
}

func (s *Server) render(w http.ResponseWriter, status int, p page) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	// same-origin keeps POST Origin equal to this host. no-referrer makes
	// browsers send Origin: null, which the CSRF check rejects.
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; img-src 'self' data: https:; style-src 'self'; script-src 'self'")
	w.WriteHeader(status)
	if err := s.tmpl.page.ExecuteTemplate(w, "page", p); err != nil {
		return
	}
}
