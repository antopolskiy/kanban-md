package web

import (
	"strconv"
	"strings"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/property"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func columns(data *boardData) []columnView {
	names := visibleStatuses(data)
	selectedID := selectedCardID(data, names)
	active := activeColumn(data.query.Column, names)
	cols := make([]columnView, 0, len(names))
	for _, name := range names {
		cols = append(cols, columnView{
			Name:       name,
			Heading:    columnHeading(data, name),
			Active:     name == active,
			OverWIP:    overWIP(data, name),
			CreateHref: newTaskHref(data.query.with(queryColumn, name), name),
			Cards:      cardsFor(data, name, selectedID),
		})
	}
	return cols
}

func visibleStatuses(data *boardData) []string {
	names := data.cfg.BoardStatuses()
	if !data.hideEmpty {
		return names
	}
	counts := map[string]int{}
	for _, t := range data.visible {
		counts[t.Status]++
	}
	filtered := make([]string, 0, len(names))
	for _, name := range names {
		if counts[name] > 0 {
			filtered = append(filtered, name)
		}
	}
	if len(filtered) == 0 {
		return names
	}
	return filtered
}

func activeColumn(want string, names []string) string {
	for _, name := range names {
		if name == want {
			return name
		}
	}
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

func columnHeading(data *boardData, name string) string {
	count := 0
	for _, t := range data.visible {
		if t.Status == name {
			count++
		}
	}
	heading := name + " (" + strconv.Itoa(count)
	if limit := data.cfg.WIPLimit(name); limit > 0 {
		heading += "/" + strconv.Itoa(limit)
	}
	return heading + ")"
}

func overWIP(data *boardData, name string) bool {
	limit := data.cfg.WIPLimit(name)
	if limit == 0 {
		return false
	}
	count := 0
	for _, t := range data.visible {
		if t.Status == name {
			count++
		}
	}
	return count >= limit
}

func cardsFor(data *boardData, status string, selectedID int) []cardView {
	var cards []cardView
	for _, t := range data.visible {
		if t.Status != status {
			continue
		}
		cards = append(cards, cardView{
			ID:       t.ID,
			Href:     data.query.with(querySelect, strconv.Itoa(t.ID)).href("/tasks/" + strconv.Itoa(t.ID)),
			Title:    t.Title,
			Body:     strings.TrimSpace(t.Body),
			IDLabel:  idLabel(data, t),
			Priority: t.Priority,
			Meta:     cardMeta(t, data.cfg),
			Tags:     tagLinks(data.query, t.Tags),
			Blocked:  blockedText(t),
			Overdue:  overdue(t, data.cfg, data.now),
			Selected: t.ID == selectedID,
		})
	}
	return cards
}

func selectedCardID(data *boardData, names []string) int {
	visible := map[int]bool{}
	var first int
	for _, name := range names {
		for _, t := range data.visible {
			if t.Status != name {
				continue
			}
			if first == 0 {
				first = t.ID
			}
			visible[t.ID] = true
		}
	}
	if visible[data.query.Selected] {
		return data.query.Selected
	}
	return first
}

func idLabel(data *boardData, t *task.Task) string {
	label := "#" + strconv.Itoa(t.ID)
	if data.tree.has {
		label += " L" + depthLabel(data.tree.depths[t.ID])
	}
	if data.cfg.StatusShowDuration(t.Status) {
		if age := ageLabel(t.Updated, data.now); age != "" {
			label += " · " + age
		}
	}
	return label
}

func cardMeta(t *task.Task, cfg *config.Config) string {
	var parts []string
	if t.Class != "" {
		parts = append(parts, t.Class)
	}
	if t.ClaimedBy != "" {
		parts = append(parts, "@"+t.ClaimedBy)
	}
	for _, field := range cfg.CardFields() {
		key, selected, err := property.SelectorKey(field)
		if !selected || err != nil {
			continue
		}
		parts = append(parts, output.TaskPropertyTokens(t, []string{key}, 0)...)
	}
	if t.Due != nil {
		parts = append(parts, "due "+t.Due.String())
	}
	return strings.Join(parts, " · ")
}

func blockedText(t *task.Task) string {
	if !t.Blocked {
		return ""
	}
	if t.BlockReason != "" {
		return "blocked: " + t.BlockReason
	}
	return "blocked"
}

func overdue(t *task.Task, cfg *config.Config, now time.Time) bool {
	return t.Due != nil && t.Due.Before(now) && !cfg.IsTerminalStatus(t.Status)
}

func tagLinks(q viewQuery, tags []string) []navLink {
	links := make([]navLink, 0, len(tags))
	for _, tag := range tags {
		links = append(links, navLink{
			Href:   q.with(queryTag, tag).href("/"),
			Label:  tag,
			Active: strings.EqualFold(q.Tag, tag),
		})
	}
	return links
}

func ageLabel(then, now time.Time) string {
	if then.IsZero() {
		return ""
	}
	d := now.Sub(then)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + "m"
	case d < hoursPerDay*time.Hour:
		return strconv.Itoa(int(d.Hours())) + "h"
	default:
		return strconv.Itoa(int(d.Hours())/hoursPerDay) + "d"
	}
}

func columnTabs(data *boardData, cols []columnView) []navLink {
	tabs := make([]navLink, 0, len(cols))
	for _, col := range cols {
		tabs = append(tabs, navLink{
			Href:   data.query.with(queryColumn, col.Name).href("/"),
			Label:  col.Heading,
			Active: col.Active,
		})
	}
	return tabs
}

func createHref(data *boardData, cols []columnView) string {
	for _, col := range cols {
		if col.Active {
			return col.CreateHref
		}
	}
	if len(data.cfg.BoardStatuses()) == 0 {
		return formHref("/tasks/new", data.query)
	}
	return newTaskHref(data.query, data.cfg.BoardStatuses()[0])
}
