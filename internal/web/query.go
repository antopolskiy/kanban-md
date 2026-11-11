package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	sortPriority = "priority"
	sortCreated  = "created"
	sortUpdated  = "updated"
	sortTitle    = "title"

	dirAsc       = "asc"
	emptyHide    = "hide"
	emptyShow    = "show"
	levelAll     = "all"
	levelUnknown = "unknown"

	querySearch = "q"
	querySort   = "sort"
	queryDir    = "dir"
	queryLevel  = "level"
	queryTag    = "tag"
	queryColumn = "col"
	queryEmpty  = "empty"
	querySelect = "sel"
)

var sortFields = []string{sortPriority, sortCreated, sortUpdated, sortTitle}

// viewQuery is session view state. It is not stored on tasks.
type viewQuery struct {
	Search   string
	Sort     string
	Asc      bool
	Level    string
	Tag      string
	Column   string
	Empty    string
	Selected int
}

func parseViewQuery(r *http.Request) viewQuery {
	q := r.URL.Query()
	v := viewQuery{
		Search: q.Get(querySearch),
		Sort:   q.Get(querySort),
		Asc:    q.Get(queryDir) == dirAsc,
		Level:  q.Get(queryLevel),
		Tag:    q.Get(queryTag),
		Column: q.Get(queryColumn),
		Empty:  q.Get(queryEmpty),
	}
	if !validSort(v.Sort) {
		v.Sort = sortPriority
	}
	if id, err := strconv.Atoi(q.Get(querySelect)); err == nil && id > 0 {
		v.Selected = id
	}
	return v
}

func validSort(field string) bool {
	for _, candidate := range sortFields {
		if candidate == field {
			return true
		}
	}
	return false
}

func (q viewQuery) hideEmpty(fallback bool) bool {
	switch q.Empty {
	case emptyHide:
		return true
	case emptyShow:
		return false
	default:
		return fallback
	}
}

func (q viewQuery) values() url.Values {
	v := url.Values{}
	if q.Search != "" {
		v.Set(querySearch, q.Search)
	}
	if q.Sort != "" && q.Sort != sortPriority {
		v.Set(querySort, q.Sort)
	}
	if q.Asc {
		v.Set(queryDir, dirAsc)
	}
	if q.Level != "" && q.Level != levelAll {
		v.Set(queryLevel, q.Level)
	}
	if q.Tag != "" {
		v.Set(queryTag, q.Tag)
	}
	if q.Column != "" {
		v.Set(queryColumn, q.Column)
	}
	if q.Empty == emptyHide || q.Empty == emptyShow {
		v.Set(queryEmpty, q.Empty)
	}
	if q.Selected > 0 {
		v.Set(querySelect, strconv.Itoa(q.Selected))
	}
	return v
}

func (q viewQuery) href(path string) string {
	if enc := q.values().Encode(); enc != "" {
		return path + "?" + enc
	}
	return path
}

func (q viewQuery) with(key, val string) viewQuery {
	next := q
	switch key {
	case querySearch:
		next.Search = val
	case querySort:
		next.Sort = val
	case queryDir:
		next.Asc = val == dirAsc
	case queryLevel:
		next.Level = val
	case queryTag:
		next.Tag = val
	case queryColumn:
		next.Column = val
	case queryEmpty:
		next.Empty = val
	case querySelect:
		id, err := strconv.Atoi(val)
		if err != nil || id < 1 {
			next.Selected = 0
			break
		}
		next.Selected = id
	}
	return next
}

func formHref(path string, q viewQuery) string {
	return withReturn(q.href(path), q.href("/"))
}

func withReturn(href, ret string) string {
	sep := "?"
	if strings.Contains(href, "?") {
		sep = "&"
	}
	return href + sep + "ret=" + url.QueryEscape(safeReturn(ret))
}

func newTaskHref(q viewQuery, status string) string {
	href := formHref("/tasks/new", q)
	return href + "&status=" + url.QueryEscape(status)
}

func safeReturn(raw string) string {
	if raw == "" {
		return "/"
	}
	u, err := url.Parse(raw)
	if err != nil || u.IsAbs() || u.Host != "" || !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") {
		return "/"
	}
	return u.RequestURI()
}

func addQuery(raw, key, val string) string {
	u, err := url.Parse(safeReturn(raw))
	if err != nil {
		return "/"
	}
	q := u.Query()
	q.Set(key, val)
	u.RawQuery = q.Encode()
	return u.RequestURI()
}
