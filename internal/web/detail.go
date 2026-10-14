package web

import (
	"html/template"
	"strconv"
	"strings"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

const timeLayout = "2006-01-02 15:04"

func (s *Server) detailPage(data *boardData, t *task.Task, csrf, errText, notice string) page {
	p := s.basePage(data, csrf, errText, notice)
	p.Kind = kindDetail
	p.Title = t.Title
	p.CreateHref = newTaskHref(data.query, t.Status)
	p.Detail = buildDetail(data, t)
	return p
}

func buildDetail(data *boardData, t *task.Task) *detailView {
	body, err := renderMarkdown(t.Body)
	if err != nil {
		body = ""
	}
	idx := data.cfg.PriorityIndex(t.Priority)
	detail := &detailView{
		ID:          t.ID,
		Title:       t.Title,
		Priority:    t.Priority,
		Meta:        detailMeta(t),
		Body:        body,
		EmptyBody:   strings.TrimSpace(t.Body) == "" || err != nil,
		File:        t.File,
		Statuses:    statusChoices(data.cfg, t.Status),
		EditHref:    withReturn(data.query.href("/tasks/"+strconv.Itoa(t.ID)+"/edit"), data.query.href("/tasks/"+strconv.Itoa(t.ID))),
		ArchiveHref: withReturn(data.query.href("/tasks/"+strconv.Itoa(t.ID)+"/archive"), data.query.href("/")),
		Archived:    data.cfg.IsArchivedStatus(t.Status),
		CanRaise:    idx >= 0 && idx < len(data.cfg.Priorities)-1,
		CanLower:    idx > 0,
	}
	path := data.tree.ancestors(t)
	detail.AncestorNote = path.marker
	detail.Ancestors = relationLinks(data, path.tasks)
	children := board.SummarizeChildren(data.all, t.ID, data.cfg, false)
	detail.Children = childLinks(data, children)
	if children.Total() > 0 {
		detail.ChildRollup = strconv.Itoa(children.Done) + "/" + strconv.Itoa(children.Total()) + " done"
	}
	detail.Dependencies = dependencyLinks(data, t)
	return detail
}

func detailMeta(t *task.Task) []string {
	lines := []string{
		"Status: " + t.Status,
		"Priority: " + t.Priority,
	}
	if t.Class != "" {
		lines = append(lines, "Class: "+t.Class)
	}
	if t.Assignee != "" {
		lines = append(lines, "Assignee: "+t.Assignee)
	}
	if t.ClaimedBy != "" {
		lines = append(lines, "Claimed: "+t.ClaimedBy)
	}
	if t.Blocked {
		lines = append(lines, "Blocked: "+blockedText(t))
	}
	if t.Due != nil {
		lines = append(lines, "Due: "+t.Due.String())
	}
	if t.Estimate != "" {
		lines = append(lines, "Estimate: "+t.Estimate)
	}
	if len(t.Tags) > 0 {
		lines = append(lines, "Tags: "+strings.Join(t.Tags, ", "))
	}
	lines = append(lines, "Created: "+t.Created.Local().Format(timeLayout))
	lines = append(lines, "Updated: "+t.Updated.Local().Format(timeLayout))
	return lines
}

func statusChoices(cfg *config.Config, current string) []statusChoice {
	names := cfg.BoardStatuses()
	choices := make([]statusChoice, 0, len(names))
	for _, name := range names {
		choices = append(choices, statusChoice{Name: name, Current: name == current})
	}
	return choices
}

func relationLinks(data *boardData, tasks []*task.Task) []navLink {
	links := make([]navLink, 0, len(tasks))
	for _, t := range tasks {
		label := "#" + strconv.Itoa(t.ID) + " " + t.Title
		if data.cfg.IsArchivedStatus(t.Status) {
			links = append(links, navLink{Label: label + " (archived)"})
			continue
		}
		links = append(links, navLink{
			Href:  data.query.href("/tasks/" + strconv.Itoa(t.ID)),
			Label: label,
		})
	}
	return links
}

func childLinks(data *boardData, summary board.ChildSummary) []navLink {
	links := make([]navLink, 0, len(summary.Children))
	for _, child := range summary.Children {
		links = append(links, navLink{
			Href:  data.query.href("/tasks/" + strconv.Itoa(child.ID)),
			Label: "#" + strconv.Itoa(child.ID) + " " + child.Title + " (" + child.Status + ")",
		})
	}
	return links
}

func dependencyLinks(data *boardData, t *task.Task) []navLink {
	links := make([]navLink, 0, len(t.DependsOn))
	for _, id := range t.DependsOn {
		dep := data.tree.byID[id]
		if dep == nil {
			links = append(links, navLink{Label: "missing #" + strconv.Itoa(id)})
			continue
		}
		label := "#" + strconv.Itoa(dep.ID) + " " + dep.Title
		if data.cfg.IsArchivedStatus(dep.Status) {
			links = append(links, navLink{Label: label + " (archived)"})
			continue
		}
		links = append(links, navLink{
			Href:  data.query.href("/tasks/" + strconv.Itoa(dep.ID)),
			Label: label,
		})
	}
	return links
}

func renderMarkdown(src string) (template.HTML, error) {
	if strings.TrimSpace(src) == "" {
		return "", nil
	}
	safe, err := sanitizeMarkdown(src)
	if err != nil {
		return "", err
	}
	return template.HTML(safe), nil //nolint:gosec // sanitized by bluemonday after goldmark
}
