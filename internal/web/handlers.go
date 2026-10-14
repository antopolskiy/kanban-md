package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func (s *Server) handleBoard(w http.ResponseWriter, r *http.Request) {
	data, err := s.loadBoard(parseViewQuery(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, http.StatusOK, s.boardPage(data, s.csrfToken(w, r), "", r.URL.Query().Get("notice")))
}

func (s *Server) handleHelp(w http.ResponseWriter, r *http.Request) {
	data, err := s.loadBoard(parseViewQuery(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p := s.basePage(data, s.csrfToken(w, r), "", "")
	p.Kind = kindHelp
	p.Title = "Help"
	p.CreateHref = createHref(data, columns(data))
	s.render(w, http.StatusOK, p)
}

func (s *Server) handleNew(w http.ResponseWriter, r *http.Request) {
	s.renderForm(w, r, s.blankForm(r, "New task", "/tasks", true), nil)
}

func (s *Server) handleEditForm(w http.ResponseWriter, r *http.Request) {
	id, t, ok := s.lookup(w, r)
	if !ok {
		return
	}
	cfg, err := config.Load(s.dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if cfg.IsArchivedStatus(t.Status) {
		http.NotFound(w, r)
		return
	}
	form := s.blankForm(r, "Edit task", "/tasks/"+strconv.Itoa(id), false)
	form.Title = t.Title
	form.Body = t.Body
	form.Priority = t.Priority
	form.Tags = strings.Join(t.Tags, ", ")
	form.Status = t.Status
	s.renderForm(w, r, form, nil)
}

func (s *Server) handleArchiveForm(w http.ResponseWriter, r *http.Request) {
	id, t, ok := s.lookup(w, r)
	if !ok {
		return
	}
	data, err := s.loadBoard(parseViewQuery(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p := s.basePage(data, s.csrfToken(w, r), "", "")
	p.Kind = kindConfirm
	p.Title = "Archive task"
	p.Confirm = &confirmView{
		Heading: "Archive #" + strconv.Itoa(id) + " " + t.Title + "?",
		Action:  "/tasks/" + strconv.Itoa(id) + "/archive",
		Return:  safeReturn(r.URL.Query().Get("ret")),
	}
	s.render(w, http.StatusOK, p)
}

func (s *Server) handleDetail(w http.ResponseWriter, r *http.Request) {
	_, t, ok := s.lookup(w, r)
	if !ok {
		return
	}
	data, err := s.loadBoard(parseViewQuery(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p := s.detailPage(data, t, s.csrfToken(w, r), "", r.URL.Query().Get("notice"))
	if p.Detail != nil {
		p.Detail.Return = safeReturn(r.URL.RequestURI())
	}
	s.render(w, http.StatusOK, p)
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	warnings, err := s.createTask(r)
	if err != nil {
		s.renderForm(w, r, formFromRequest(r, "New task", "/tasks", s.priorities(), true), err)
		return
	}
	s.finish(w, r, warnings)
}

func (s *Server) handleEdit(w http.ResponseWriter, r *http.Request) {
	id, err := taskID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	warnings, err := s.editTask(r, id)
	if err != nil {
		s.renderForm(w, r, formFromRequest(r, "Edit task", r.URL.Path, s.priorities(), false), err)
		return
	}
	s.finish(w, r, warnings)
}

func (s *Server) handleMove(w http.ResponseWriter, r *http.Request) {
	s.postTask(w, r, s.moveTask)
}

func (s *Server) handlePriority(w http.ResponseWriter, r *http.Request) {
	s.postTask(w, r, s.priorityTask)
}

func (s *Server) handleArchive(w http.ResponseWriter, r *http.Request) {
	id, err := taskID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	warnings, err := s.archiveTask(id)
	if err != nil {
		s.failBoard(w, r, err)
		return
	}
	s.finish(w, r, warnings)
}

func (s *Server) postTask(w http.ResponseWriter, r *http.Request, fn func(*http.Request, int) ([]string, error)) {
	id, err := taskID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	warnings, err := fn(r, id)
	if err != nil {
		s.failBoard(w, r, err)
		return
	}
	s.finish(w, r, warnings)
}

func (s *Server) finish(w http.ResponseWriter, r *http.Request, warnings []string) {
	target := safeReturn(r.FormValue("return"))
	if len(warnings) > 0 {
		target = addQuery(target, "notice", strings.Join(warnings, " "))
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (s *Server) failBoard(w http.ResponseWriter, r *http.Request, err error) {
	data, loadErr := s.loadBoard(parseViewQuery(r))
	if loadErr != nil {
		http.Error(w, loadErr.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, statusFor(err), s.boardPage(data, s.csrfToken(w, r), err.Error(), ""))
}

func (s *Server) lookup(w http.ResponseWriter, r *http.Request) (int, *task.Task, bool) {
	id, err := taskID(r)
	if err != nil {
		http.NotFound(w, r)
		return 0, nil, false
	}
	cfg, err := config.Load(s.dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return 0, nil, false
	}
	t, err := readTask(cfg, id)
	if err != nil {
		http.NotFound(w, r)
		return 0, nil, false
	}
	return id, t, true
}

func (s *Server) blankForm(r *http.Request, heading, action string, isCreate bool) formView {
	cfg, err := config.Load(s.dir)
	priority := ""
	var priorities []string
	if err == nil {
		priorities = cfg.Priorities
		priority = cfg.Defaults.Priority
	}
	return formView{
		Heading:    heading,
		Action:     action,
		Priority:   priority,
		Priorities: priorities,
		Status:     r.URL.Query().Get("status"),
		Return:     safeReturn(r.URL.Query().Get("ret")),
		IsCreate:   isCreate,
	}
}

func (s *Server) priorities() []string {
	cfg, err := config.Load(s.dir)
	if err != nil {
		return nil
	}
	return cfg.Priorities
}

func (s *Server) renderForm(w http.ResponseWriter, r *http.Request, form formView, formErr error) {
	data, err := s.loadBoard(parseViewQuery(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status := http.StatusOK
	errText := ""
	if formErr != nil {
		errText = formErr.Error()
		status = statusFor(formErr)
	}
	p := s.basePage(data, s.csrfToken(w, r), errText, "")
	p.Kind = kindForm
	p.Title = form.Heading
	p.Form = &form
	p.CreateHref = newTaskHref(data.query, form.Status)
	s.render(w, status, p)
}
