package web

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/filelock"
	"github.com/antopolskiy/kanban-md/internal/task"
)

const (
	statusNext = "next"
	statusPrev = "prev"
	priorityUp = "up"
)

func (s *Server) createTask(r *http.Request) ([]string, error) {
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		return nil, errors.New("title is required")
	}
	unlock, err := filelock.Lock(filepath.Join(s.dir, ".lock"))
	if err != nil {
		return nil, fmt.Errorf("acquiring lock: %w", err)
	}
	defer func() { _ = unlock() }()

	cfg, err := config.Load(s.dir)
	if err != nil {
		return nil, err
	}
	_, err = board.Create(cfg, board.CreateParams{
		Title:    title,
		Status:   r.FormValue("status"),
		Priority: r.FormValue("priority"),
		Tags:     parseTags(r.FormValue("tags")),
		Body:     strings.TrimSpace(r.FormValue("body")),
	}, s.now())
	return nil, err
}

func (s *Server) editTask(r *http.Request, id int) ([]string, error) {
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		return nil, errors.New("title is required")
	}
	body := strings.TrimSpace(r.FormValue("body"))
	priority := r.FormValue("priority")
	tags := parseTags(r.FormValue("tags"))
	cfg, err := config.Load(s.dir)
	if err != nil {
		return nil, err
	}
	_, err = board.Edit(cfg, id, s.claimant(), false, func(t *task.Task) (bool, error) {
		if t.Title == title && t.Body == body && t.Priority == priority && slices.Equal(t.Tags, tags) {
			return false, nil
		}
		t.Title = title
		t.Body = body
		t.Priority = priority
		t.Tags = tags
		return true, nil
	}, s.now())
	if isNoChanges(err) {
		return nil, nil
	}
	return nil, err
}

func (s *Server) moveTask(r *http.Request, id int) ([]string, error) {
	cfg, err := config.Load(s.dir)
	if err != nil {
		return nil, err
	}
	current, err := readTask(cfg, id)
	if err != nil {
		return nil, err
	}
	status, err := resolveStatus(cfg, current.Status, r.FormValue("status"))
	if err != nil {
		return nil, err
	}
	params := board.MoveParams{ID: id, NewStatus: status}
	if current.ClaimedBy != "" {
		params.Claimant = current.ClaimedBy
	} else if cfg.StatusRequiresClaim(status) {
		params.Claimant = s.claimant()
		params.SetClaim = true
	}
	result, err := board.Move(cfg, params, s.now())
	if err != nil || result == nil {
		return nil, err
	}
	return result.Warnings, nil
}

func (s *Server) priorityTask(r *http.Request, id int) ([]string, error) {
	cfg, err := config.Load(s.dir)
	if err != nil {
		return nil, err
	}
	current, err := readTask(cfg, id)
	if err != nil {
		return nil, err
	}
	next, err := stepPriority(cfg, current.Priority, r.FormValue("direction"))
	if err != nil {
		return nil, err
	}
	_, err = board.Edit(cfg, id, s.claimant(), false, func(t *task.Task) (bool, error) {
		if t.Priority == next {
			return false, nil
		}
		t.Priority = next
		return true, nil
	}, s.now())
	if isNoChanges(err) {
		return nil, nil
	}
	return nil, err
}

func (s *Server) archiveTask(id int) ([]string, error) {
	cfg, err := config.Load(s.dir)
	if err != nil {
		return nil, err
	}
	result, err := board.Delete(cfg, id, s.claimant(), s.now())
	if err != nil || result == nil {
		return nil, err
	}
	return result.Warnings, nil
}

func readTask(cfg *config.Config, id int) (*task.Task, error) {
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		return nil, err
	}
	return task.Read(path)
}

func resolveStatus(cfg *config.Config, current, requested string) (string, error) {
	if requested != statusNext && requested != statusPrev {
		return requested, nil
	}
	names := cfg.BoardStatuses()
	idx := -1
	for i, name := range names {
		if name == current {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", fmt.Errorf("status %q is not on the board", current)
	}
	if requested == statusNext {
		if idx >= len(names)-1 {
			return "", errors.New("task is already at the last status")
		}
		return names[idx+1], nil
	}
	if idx == 0 {
		return "", errors.New("task is already at the first status")
	}
	return names[idx-1], nil
}

func stepPriority(cfg *config.Config, current, direction string) (string, error) {
	idx := cfg.PriorityIndex(current)
	if direction == priorityUp {
		if idx < 0 || idx >= len(cfg.Priorities)-1 {
			return "", errors.New("task is already at the highest priority")
		}
		return cfg.Priorities[idx+1], nil
	}
	if idx <= 0 {
		return "", errors.New("task is already at the lowest priority")
	}
	return cfg.Priorities[idx-1], nil
}

func parseTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var tags []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			tags = append(tags, part)
		}
	}
	return tags
}

func isNoChanges(err error) bool {
	var cliErr *clierr.Error
	return errors.As(err, &cliErr) && cliErr.Code == clierr.NoChanges
}

func taskID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		return 0, errors.New("invalid task id")
	}
	return id, nil
}

func formFromRequest(r *http.Request, heading, action string, priorities []string, isCreate bool) formView {
	ret := r.URL.Query().Get("ret")
	if r.FormValue("return") != "" {
		ret = r.FormValue("return")
	}
	return formView{
		Heading:    heading,
		Action:     action,
		Title:      r.FormValue("title"),
		Body:       r.FormValue("body"),
		Priority:   r.FormValue("priority"),
		Priorities: priorities,
		Tags:       r.FormValue("tags"),
		Status:     r.FormValue("status"),
		Return:     safeReturn(ret),
		IsCreate:   isCreate,
	}
}

func (s *Server) now() time.Time {
	if s.clock != nil {
		return s.clock()
	}
	return time.Now()
}
