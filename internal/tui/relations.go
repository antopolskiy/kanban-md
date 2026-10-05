package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

const relationGutter = 2

type detailFrame struct {
	taskID    int
	scrollOff int
	focusID   int
}

type relationSpan struct {
	taskID int
	start  int
	end    int
}

type detailDocument struct {
	lines     []string
	relations []relationSpan
}

func (b *Board) activeTask(id int) *task.Task {
	if b.activeByID != nil {
		return b.activeByID[id]
	}
	for _, t := range b.unfilteredTasks {
		if t.ID == id {
			return t
		}
	}
	return nil
}

func (b *Board) detailDocument(t *task.Task) detailDocument {
	var doc detailDocument
	header := fmt.Sprintf("Task #%d: %s", t.ID, t.Title)
	bold := lipgloss.NewStyle().Bold(true)
	for _, line := range wrapTitle(header, b.width, noLineLimit) {
		doc.lines = append(doc.lines, bold.Render(line))
	}
	doc.lines = append(doc.lines, strings.Repeat("─", min(lipgloss.Width(header), max(b.width, 0))), "")
	doc.lines = append(doc.lines, detailLabelStyle.Render("Status:")+"  "+t.Status)
	doc.lines = append(doc.lines, detailLabelStyle.Render("Priority:")+"  "+t.Priority)
	doc.lines = append(doc.lines, detailMetadataLines(t)...)
	for _, line := range propertyDetailLines(t, b.cfg) {
		doc.lines = append(doc.lines, strings.Split(ansi.Hardwrap(line, max(b.width, 1), true), "\n")...)
	}
	doc.lines = append(doc.lines, detailTimestampLines(t)...)
	if t.Blocked {
		doc.lines = append(doc.lines, "", errorStyle.Render("BLOCKED: "+t.BlockReason))
	}
	if t.Parent != nil {
		path := b.hierarchyIndex().ancestors(t)
		doc.lines = append(doc.lines, "", bold.Render("Ancestry"))
		if path.marker != "" {
			doc.appendRelationRow(0, path.marker, b.width, 0)
		}
		for _, ancestor := range path.tasks {
			id := 0
			if b.activeTask(ancestor.ID) != nil {
				id = ancestor.ID
			}
			doc.appendRelationRow(id, fmt.Sprintf("↑ #%d [%s] %s", ancestor.ID, ancestor.Status, ancestor.Title),
				b.width, b.detailFocusID)
		}
		doc.appendRelationRow(0, fmt.Sprintf("#%d [%s] %s (current)", t.ID, t.Status, t.Title), b.width, 0)
	}
	children := board.SummarizeChildren(b.unfilteredTasks, t.ID, b.cfg, false)
	board.SelectChildProperties(&children, b.unfilteredTasks, propertyViewKeys(b.cfg))
	if children.Total() > 0 {
		doc.lines = append(doc.lines, "", bold.Render(fmt.Sprintf("Children (%d/%d done)", children.Done, children.Total())))
		for i, child := range children.Children {
			branch := "├─"
			if i == len(children.Children)-1 {
				branch = "└─"
			}
			id := child.ID
			// A malformed cycle can put one task in both the ancestor path and
			// direct children. Keep the row, with one focus stop per task ID.
			if doc.relation(id) != nil {
				id = 0
			}
			text := fmt.Sprintf("%s #%d [%s] %s", branch, child.ID, child.Status, child.Title)
			if tokens := output.PropertyTokens(child.PropertyKeys, child.Properties, 0); len(tokens) > 0 {
				text += " (" + strings.Join(tokens, " ") + ")"
			}
			doc.appendRelationRow(id, text,
				b.width, b.detailFocusID)
		}
	}
	if t.Body != "" {
		doc.lines = append(doc.lines, "")
		doc.lines = append(doc.lines, strings.Split(renderMarkdown(unescapeBody(t.Body), b.width), "\n")...)
	}
	return doc
}

func (d *detailDocument) appendRelationRow(id int, text string, width, focusID int) {
	start := len(d.lines)
	for i, line := range strings.Split(ansi.Wrap(text, max(width-relationGutter, 1), ""), "\n") {
		gutter := "  "
		if i == 0 && id != 0 && id == focusID {
			gutter = "> "
		}
		line = truncate(gutter+line, max(width, 1))
		if id != 0 {
			line = lipgloss.NewStyle().Underline(true).Render(line)
		}
		d.lines = append(d.lines, line)
	}
	if id != 0 {
		d.relations = append(d.relations, relationSpan{taskID: id, start: start, end: len(d.lines)})
	}
}

func (d detailDocument) relation(id int) *relationSpan {
	for i := range d.relations {
		if d.relations[i].taskID == id {
			return &d.relations[i]
		}
	}
	return nil
}

func (b *Board) focusRelation(reverse bool) {
	if b.detailTask == nil {
		return
	}
	doc := b.detailDocument(b.detailTask)
	if len(doc.relations) == 0 {
		b.detailFocusID = 0
		return
	}
	idx := -1
	for i, relation := range doc.relations {
		if relation.taskID == b.detailFocusID {
			idx = i
			break
		}
	}
	if reverse {
		if idx < 0 {
			idx = 0
		}
		idx = (idx + len(doc.relations) - 1) % len(doc.relations)
	} else {
		idx = (idx + 1) % len(doc.relations)
	}
	relation := doc.relations[idx]
	b.detailFocusID = relation.taskID
	viewHeight := b.detailViewportHeight(len(doc.lines))
	if relation.end-relation.start > viewHeight || relation.start < b.detailScrollOff {
		b.detailScrollOff = relation.start
	} else if relation.end > b.detailScrollOff+viewHeight {
		b.detailScrollOff = relation.end - viewHeight
	}
	b.clampDetailScroll()
}

func (b *Board) openRelation(id int) {
	if b.detailTask == nil || id == b.detailTask.ID {
		return
	}
	target := b.activeTask(id)
	if target == nil || b.detailDocument(b.detailTask).relation(id) == nil {
		return
	}
	b.clampDetailScroll()
	b.detailHistory = append(b.detailHistory, detailFrame{
		taskID: b.detailTask.ID, scrollOff: b.detailScrollOff, focusID: id,
	})
	b.detailTask = target
	b.detailScrollOff = 0
	b.detailFocusID = 0
	b.invalidatePointerState()
}

func (b *Board) backDetail() {
	for len(b.detailHistory) > 0 {
		last := len(b.detailHistory) - 1
		frame := b.detailHistory[last]
		b.detailHistory = b.detailHistory[:last]
		if t := b.activeTask(frame.taskID); t != nil {
			b.detailTask = t
			b.detailScrollOff = frame.scrollOff
			b.detailFocusID = frame.focusID
			b.refreshDetailFocus()
			b.invalidatePointerState()
			return
		}
	}
	b.closeDetail()
}

func (b *Board) closeDetail() {
	b.view = viewBoard
	b.detailTask = nil
	b.detailScrollOff = 0
	b.detailFocusID = 0
	b.detailHistory = nil
	if b.detailOriginID != 0 {
		b.selectTaskByID(b.detailOriginID)
	}
	b.detailOriginID = 0
	b.invalidatePointerState()
}

func (b *Board) refreshDetailFocus() {
	if b.detailTask != nil && b.detailDocument(b.detailTask).relation(b.detailFocusID) == nil {
		b.detailFocusID = 0
	}
	b.clampDetailScroll()
}

func (b *Board) detailViewportHeight(lineCount int) int {
	if height := b.height - detailChrome; height > 0 {
		return height
	}
	return max(lineCount, 1)
}
