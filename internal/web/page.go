package web

import (
	"fmt"
	"html/template"
	"strconv"
	"time"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

const (
	kindBoard   = "board"
	kindDetail  = "detail"
	kindForm    = "form"
	kindConfirm = "confirm"
	kindHelp    = "help"

	activityLimit = 8
	hoursPerDay   = 24
)

type navLink struct {
	Href   string
	Label  string
	Active bool
}

type field struct {
	Name  string
	Value string
}

type cardView struct {
	ID       int
	Href     string
	Title    string
	Body     string
	IDLabel  string
	Priority string
	Meta     string
	Tags     []navLink
	Blocked  string
	Overdue  bool
	Selected bool
}

type columnView struct {
	Name       string
	Heading    string
	Active     bool
	OverWIP    bool
	CreateHref string
	Cards      []cardView
}

type formView struct {
	Heading    string
	Action     string
	Title      string
	Body       string
	Priority   string
	Priorities []string
	Tags       string
	Status     string
	Return     string
	IsCreate   bool
}

type statusChoice struct {
	Name    string
	Current bool
}

type detailView struct {
	ID           int
	Title        string
	Priority     string
	Meta         []string
	Body         template.HTML
	EmptyBody    bool
	File         string
	Ancestors    []navLink
	AncestorNote string
	Children     []navLink
	ChildRollup  string
	Dependencies []navLink
	Statuses     []statusChoice
	EditHref     string
	ArchiveHref  string
	Return       string
	Archived     bool
	CanRaise     bool
	CanLower     bool
}

type confirmView struct {
	Heading string
	Action  string
	Return  string
}

type page struct {
	Kind         string
	Title        string
	BoardName    string
	CSRF         string
	Error        string
	Notice       string
	Search       string
	Sort         string
	Dir          string
	Level        string
	LevelValues  string
	CreateHref   string
	HelpHref     string
	BoardHref    string
	Hidden       []field
	SortLinks    []navLink
	DirLink      navLink
	LevelLinks   []navLink
	Tag          string
	ClearTagHref string
	EmptyLink    navLink
	Summary      []string
	Tabs         []navLink
	Columns      []columnView
	Activity     []string
	Detail       *detailView
	Form         *formView
	Confirm      *confirmView
}

type boardData struct {
	cfg       *config.Config
	all       []*task.Task
	active    []*task.Task
	visible   []*task.Task
	tree      *hierarchy
	query     viewQuery
	hideEmpty bool
	now       time.Time
}

func (s *Server) loadBoard(q viewQuery) (*boardData, error) {
	cfg, err := config.Load(s.dir)
	if err != nil {
		return nil, err
	}
	all, _, err := task.ReadAllLenient(cfg.TasksPath())
	if err != nil {
		return nil, err
	}
	tree := newHierarchy(all)
	active := activeTasks(cfg, all)
	visible := filterVisible(active, tree, q)
	board.Sort(visible, q.Sort, !q.Asc, cfg)
	return &boardData{
		cfg:       cfg,
		all:       all,
		active:    active,
		visible:   visible,
		tree:      tree,
		query:     q,
		hideEmpty: q.hideEmpty(s.hideEmpty),
		now:       s.now(),
	}, nil
}

func activeTasks(cfg *config.Config, all []*task.Task) []*task.Task {
	active := make([]*task.Task, 0, len(all))
	for _, t := range all {
		if !cfg.IsArchivedStatus(t.Status) {
			active = append(active, t)
		}
	}
	return active
}

func filterVisible(active []*task.Task, tree *hierarchy, q viewQuery) []*task.Task {
	visible := make([]*task.Task, 0, len(active))
	for _, t := range active {
		if !matchesBoardSearch(t, q.Search) || !hasTag(t, q.Tag) || !levelMatch(tree.depths[t.ID], q.Level) {
			continue
		}
		visible = append(visible, t)
	}
	return visible
}

func (s *Server) boardPage(data *boardData, csrf, errText, notice string) page {
	p := s.basePage(data, csrf, errText, notice)
	p.Kind = kindBoard
	p.Title = data.cfg.Board.Name
	p.Summary = summaryLines(board.Summary(data.cfg, data.visible, data.now))
	p.Columns = columns(data)
	p.Tabs = columnTabs(data, p.Columns)
	p.Activity = activityLines(data.cfg.Dir())
	p.LevelLinks = levelLinks(data)
	p.CreateHref = createHref(data, p.Columns)
	if p.Tag != "" {
		p.ClearTagHref = data.query.with(queryTag, "").href("/")
	}
	return p
}

func (s *Server) basePage(data *boardData, csrf, errText, notice string) page {
	q := data.query
	dir := "desc"
	dirLabel := "Ascending"
	dirValue := dirAsc
	if q.Asc {
		dir = dirAsc
		dirLabel = "Descending"
		dirValue = "desc"
	}
	return page{
		BoardName:   data.cfg.Board.Name,
		CSRF:        csrf,
		Error:       errText,
		Notice:      notice,
		Search:      q.Search,
		Sort:        q.Sort,
		Dir:         dir,
		Level:       currentLevel(q.Level),
		HelpHref:    "/help",
		BoardHref:   q.href("/"),
		Hidden:      hiddenFields(q),
		SortLinks:   sortLinks(q),
		DirLink:     navLink{Href: q.with(queryDir, dirValue).href("/"), Label: dirLabel},
		Tag:         q.Tag,
		EmptyLink:   emptyLink(q, data.hideEmpty),
		LevelValues: levelValueList(data),
	}
}

func currentLevel(level string) string {
	if level == "" {
		return levelAll
	}
	return level
}

func hiddenFields(q viewQuery) []field {
	var fields []field
	if q.Sort != "" && q.Sort != sortPriority {
		fields = append(fields, field{Name: querySort, Value: q.Sort})
	}
	if q.Asc {
		fields = append(fields, field{Name: queryDir, Value: dirAsc})
	}
	if q.Level != "" && q.Level != levelAll {
		fields = append(fields, field{Name: queryLevel, Value: q.Level})
	}
	if q.Tag != "" {
		fields = append(fields, field{Name: queryTag, Value: q.Tag})
	}
	if q.Column != "" {
		fields = append(fields, field{Name: queryColumn, Value: q.Column})
	}
	if q.Empty == emptyHide || q.Empty == emptyShow {
		fields = append(fields, field{Name: queryEmpty, Value: q.Empty})
	}
	return fields
}

func sortLinks(q viewQuery) []navLink {
	links := make([]navLink, 0, len(sortFields))
	for _, field := range sortFields {
		links = append(links, navLink{
			Href:   q.with(querySort, field).href("/"),
			Label:  field,
			Active: q.Sort == field,
		})
	}
	return links
}

func emptyLink(q viewQuery, hiding bool) navLink {
	if hiding {
		return navLink{Href: q.with(queryEmpty, emptyShow).href("/"), Label: "Show empty"}
	}
	return navLink{Href: q.with(queryEmpty, emptyHide).href("/"), Label: "Hide empty"}
}

func summaryLines(overview board.Overview) []string {
	lines := []string{strconv.Itoa(overview.TotalTasks) + " tasks"}
	blocked := 0
	overdue := 0
	for _, status := range overview.Statuses {
		blocked += status.Blocked
		overdue += status.Overdue
		label := status.Status + " " + strconv.Itoa(status.Count)
		if status.WIPLimit > 0 {
			label += "/" + strconv.Itoa(status.WIPLimit)
		}
		lines = append(lines, label)
	}
	if blocked > 0 {
		lines = append(lines, strconv.Itoa(blocked)+" blocked")
	}
	if overdue > 0 {
		lines = append(lines, strconv.Itoa(overdue)+" overdue")
	}
	return lines
}

func activityLines(dir string) []string {
	entries, err := board.ReadLog(dir, board.LogFilterOptions{Limit: activityLimit})
	if err != nil || len(entries) == 0 {
		return nil
	}
	lines := make([]string, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		lines = append(lines, fmt.Sprintf("%s %s #%d %s",
			entry.Timestamp.Local().Format("15:04"), entry.Action, entry.TaskID, entry.Detail))
	}
	return lines
}
