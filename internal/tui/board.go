// Package tui implements an interactive terminal UI for kanban-md boards.
package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// view represents the current screen state.
type view int

const (
	viewBoard view = iota
	viewDetail
	viewMove
	viewConfirmDelete
	viewHelp
)

// Key and layout constants.
const (
	keyEsc  = "esc"
	keyDown = "down"
	keyUp   = "up"

	tagMaxFraction = 2 // tags get at most 1/N of card width
	boardChrome    = 2 // blank line + status bar below the column area
	maxScrollOff   = 1<<31 - 1
	tickInterval   = 30 * time.Second // how often durations refresh
)

// Board is the top-level bubbletea model.
type Board struct {
	cfg       *config.Config
	tasks     []*task.Task
	columns   []column
	activeCol int
	activeRow int
	view      view
	width     int
	height    int
	err       error
	now       func() time.Time // clock for duration display; defaults to time.Now

	// Detail view.
	detailTask      *task.Task
	detailScrollOff int

	// Move view.
	moveStatuses []string
	moveCursor   int

	// Delete confirmation.
	deleteID    int
	deleteTitle string
}

// column groups tasks belonging to a single status.
type column struct {
	status    string
	tasks     []*task.Task
	scrollOff int // first visible row index
}

// NewBoard creates a new Board model from a config.
func NewBoard(cfg *config.Config) *Board {
	b := &Board{cfg: cfg, now: time.Now}
	b.loadTasks()
	return b
}

// SetNow overrides the clock function used for duration display (for testing).
func (b *Board) SetNow(fn func() time.Time) {
	b.now = fn
}

// Init implements tea.Model.
func (b *Board) Init() tea.Cmd {
	return tickCmd()
}

// Update implements tea.Model.
func (b *Board) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return b.handleKey(msg)
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
		return b, nil
	case ReloadMsg:
		b.loadTasks()
		return b, nil
	case TickMsg:
		return b, tickCmd()
	case errMsg:
		b.err = msg.err
		return b, nil
	}
	return b, nil
}

// View implements tea.Model.
func (b *Board) View() string {
	if b.width == 0 {
		return "Loading..."
	}

	switch b.view {
	case viewDetail:
		return b.viewDetail()
	case viewMove:
		return b.viewMoveDialog()
	case viewConfirmDelete:
		return b.viewDeleteConfirm()
	case viewHelp:
		return b.viewHelp()
	default:
		return b.viewBoard()
	}
}

func (b *Board) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys.
	if key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))) {
		return b, tea.Quit
	}

	switch b.view {
	case viewBoard:
		return b.handleBoardKey(msg)
	case viewDetail:
		return b.handleDetailKey(msg)
	case viewMove:
		return b.handleMoveKey(msg)
	case viewConfirmDelete:
		return b.handleDeleteKey(msg)
	case viewHelp:
		return b.handleHelpKey(msg)
	}

	return b, nil
}

func (b *Board) handleBoardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", keyEsc:
		return b, tea.Quit
	case "?":
		b.view = viewHelp
	case "h", "left", "l", "right", "j", keyDown, "k", keyUp:
		b.handleNavigation(msg.String())
	case "enter":
		b.handleEnter()
	case "m":
		b.handleMoveStart()
	case "N":
		return b.moveNext()
	case "P":
		return b.movePrev()
	case "d":
		b.handleDeleteStart()
	case "r":
		b.loadTasks()
	}
	return b, nil
}

func (b *Board) handleNavigation(k string) {
	switch k {
	case "h", "left":
		if b.activeCol > 0 {
			b.activeCol--
			b.clampRow()
		}
	case "l", "right":
		if b.activeCol < len(b.columns)-1 {
			b.activeCol++
			b.clampRow()
		}
	case "j", keyDown:
		col := b.currentColumn()
		if col != nil && b.activeRow < len(col.tasks)-1 {
			b.activeRow++
			b.ensureVisible()
		}
	case "k", keyUp:
		if b.activeRow > 0 {
			b.activeRow--
			b.ensureVisible()
		}
	}
}

func (b *Board) handleEnter() {
	if t := b.selectedTask(); t != nil {
		b.detailTask = t
		b.detailScrollOff = 0
		b.view = viewDetail
	}
}

func (b *Board) handleMoveStart() {
	if t := b.selectedTask(); t != nil {
		b.moveStatuses = b.cfg.StatusNames()
		b.moveCursor = b.cfg.StatusIndex(t.Status)
		if b.moveCursor < 0 {
			b.moveCursor = 0
		}
		b.view = viewMove
	}
}

func (b *Board) handleDeleteStart() {
	if t := b.selectedTask(); t != nil {
		b.deleteID = t.ID
		b.deleteTitle = t.Title
		b.view = viewConfirmDelete
	}
}

func (b *Board) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", keyEsc, "backspace":
		b.view = viewBoard
		b.detailTask = nil
		b.detailScrollOff = 0
	case "j", keyDown:
		b.detailScrollOff++
	case "k", keyUp:
		if b.detailScrollOff > 0 {
			b.detailScrollOff--
		}
	case "g":
		b.detailScrollOff = 0
	case "G":
		// Set to large value; viewDetail will clamp it.
		b.detailScrollOff = maxScrollOff
	}
	return b, nil
}

func (b *Board) handleMoveKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc, "q":
		b.view = viewBoard
	case "j", keyDown:
		if b.moveCursor < len(b.moveStatuses)-1 {
			b.moveCursor++
		}
	case "k", keyUp:
		if b.moveCursor > 0 {
			b.moveCursor--
		}
	case "enter":
		return b.executeMove(b.moveStatuses[b.moveCursor])
	}
	return b, nil
}

func (b *Board) handleDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return b.executeDelete()
	case "n", "N", keyEsc, "q":
		b.view = viewBoard
	}
	return b, nil
}

func (b *Board) handleHelpKey(_ tea.KeyMsg) (tea.Model, tea.Cmd) {
	b.view = viewBoard
	return b, nil
}

// loadTasks reads all tasks and organizes them into columns.
func (b *Board) loadTasks() {
	tasks, _, err := task.ReadAllLenient(b.cfg.TasksPath())
	if err != nil {
		b.err = err
		return
	}
	b.err = nil

	// Filter out archived tasks from TUI display.
	var visibleTasks []*task.Task
	for _, t := range tasks {
		if !b.cfg.IsArchivedStatus(t.Status) {
			visibleTasks = append(visibleTasks, t)
		}
	}
	b.tasks = visibleTasks

	// Sort tasks by priority (higher priority first).
	board.Sort(visibleTasks, "priority", true, b.cfg)

	// Build columns from board statuses (excludes archived).
	displayStatuses := b.cfg.BoardStatuses()
	b.columns = make([]column, len(displayStatuses))
	for i, status := range displayStatuses {
		b.columns[i] = column{status: status}
	}

	for _, t := range visibleTasks {
		for i := range b.columns {
			if b.columns[i].status == t.Status {
				b.columns[i].tasks = append(b.columns[i].tasks, t)
				break
			}
		}
	}

	b.clampRow()
}

func (b *Board) currentColumn() *column {
	if b.activeCol >= 0 && b.activeCol < len(b.columns) {
		return &b.columns[b.activeCol]
	}
	return nil
}

func (b *Board) selectedTask() *task.Task {
	col := b.currentColumn()
	if col == nil || len(col.tasks) == 0 {
		return nil
	}
	if b.activeRow >= 0 && b.activeRow < len(col.tasks) {
		return col.tasks[b.activeRow]
	}
	return nil
}

func (b *Board) clampRow() {
	col := b.currentColumn()
	if col == nil || len(col.tasks) == 0 {
		b.activeRow = 0
		return
	}
	if b.activeRow >= len(col.tasks) {
		b.activeRow = len(col.tasks) - 1
	}
	b.ensureVisible()
}

// visibleCardsForColumn returns the number of cards that fit in the column,
// accounting for scroll indicator lines ("↑ N more" / "↓ N more") that
// consume vertical space.
func (b *Board) visibleCardsForColumn(col *column, width int) int {
	budget := b.height - boardChrome
	if budget < 1 {
		return 1
	}

	// Always need 1 line for column header.
	avail := budget - 1

	// Check if up indicator is needed.
	if col.scrollOff > 0 {
		avail--
	}

	// Compute cards assuming no down indicator.
	n := b.fitCardsInHeight(col, avail, width)

	// Check if down indicator is needed.
	if col.scrollOff+n < len(col.tasks) {
		// Re-compute with 1 fewer line for the down indicator.
		n = b.fitCardsInHeight(col, avail-1, width)
		if n < 1 {
			n = 1
		}
	}

	return n
}

// ensureVisible adjusts the active column's scroll offset so the
// selected row is within the visible window.
func (b *Board) ensureVisible() {
	col := b.currentColumn()
	if col == nil {
		return
	}
	maxVis := b.visibleCardsForColumn(col, b.columnWidth())

	// Scroll down if active row is below visible window.
	if b.activeRow >= col.scrollOff+maxVis {
		col.scrollOff = b.activeRow - maxVis + 1
	}
	// Scroll up if active row is above visible window.
	if b.activeRow < col.scrollOff {
		col.scrollOff = b.activeRow
	}
}

func (b *Board) fitCardsInHeight(col *column, avail, width int) int {
	if len(col.tasks) == 0 {
		return 1
	}
	if avail < 1 {
		return 1
	}

	used := 0
	count := 0
	for i := col.scrollOff; i < len(col.tasks); i++ {
		cardLines := b.cardHeight(col.tasks[i], width)
		if count > 0 && used+cardLines > avail {
			break
		}
		count++
		used += cardLines
		if used >= avail {
			break
		}
	}

	if count < 1 {
		return 1
	}
	return count
}

// moveNext moves the selected task to the next board status (excludes archived).
func (b *Board) moveNext() (tea.Model, tea.Cmd) {
	t := b.selectedTask()
	if t == nil {
		return b, nil
	}

	boardStatuses := b.cfg.BoardStatuses()
	idx := indexOf(boardStatuses, t.Status)
	if idx < 0 || idx >= len(boardStatuses)-1 {
		b.err = fmt.Errorf("task #%d is already at the last status", t.ID)
		return b, nil
	}

	return b.executeMove(boardStatuses[idx+1])
}

// movePrev moves the selected task to the previous board status (excludes archived).
func (b *Board) movePrev() (tea.Model, tea.Cmd) {
	t := b.selectedTask()
	if t == nil {
		return b, nil
	}

	boardStatuses := b.cfg.BoardStatuses()
	idx := indexOf(boardStatuses, t.Status)
	if idx <= 0 {
		b.err = fmt.Errorf("task #%d is already at the first status", t.ID)
		return b, nil
	}

	return b.executeMove(boardStatuses[idx-1])
}

// indexOf returns the index of item in slice, or -1.
func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

func (b *Board) executeMove(targetStatus string) (tea.Model, tea.Cmd) {
	t := b.selectedTask()
	if t == nil {
		b.view = viewBoard
		return b, nil
	}

	if t.Status == targetStatus {
		b.view = viewBoard
		return b, nil
	}

	oldStatus := t.Status
	t.Status = targetStatus
	task.UpdateTimestamps(t, oldStatus, targetStatus, b.cfg)

	if err := task.Write(t.File, t); err != nil {
		b.err = fmt.Errorf("moving task #%d: %w", t.ID, err)
		t.Status = oldStatus // revert
	} else {
		board.LogMutation(b.cfg.Dir(), "move", t.ID, oldStatus+" -> "+targetStatus)
	}

	b.view = viewBoard
	b.loadTasks()
	return b, nil
}

func (b *Board) executeDelete() (tea.Model, tea.Cmd) {
	path, err := task.FindByID(b.cfg.TasksPath(), b.deleteID)
	if err != nil {
		b.err = fmt.Errorf("finding task #%d: %w", b.deleteID, err)
		b.view = viewBoard
		return b, nil
	}

	if err := os.Remove(path); err != nil {
		b.err = fmt.Errorf("deleting task #%d: %w", b.deleteID, err)
	} else {
		board.LogMutation(b.cfg.Dir(), "delete", b.deleteID, b.deleteTitle)
	}

	b.view = viewBoard
	b.loadTasks()
	return b, nil
}

// WatchPaths returns the paths that should be watched for file changes.
func (b *Board) WatchPaths() []string {
	paths := []string{b.cfg.TasksPath()}
	if b.cfg.Dir() != b.cfg.TasksPath() {
		paths = append(paths, b.cfg.Dir())
	}
	return paths
}

// --- Messages ---

// ReloadMsg is sent by the file watcher to trigger a board refresh.
type ReloadMsg struct{}

type errMsg struct{ err error }

// TickMsg is sent periodically to refresh duration displays.
type TickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(time.Time) tea.Msg { return TickMsg{} })
}

// --- Styles ---

var (
	columnHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("252")).
				Background(lipgloss.Color("236")).
				Padding(0, 1)

	activeColumnHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("62")).
				Padding(0, 1)

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			MarginBottom(0)

	activeCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1).
			MarginBottom(0)

	blockedCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("196")).
				Padding(0, 1).
				MarginBottom(0)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	priorityStyles = map[string]lipgloss.Style{
		"critical": lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		"high":     lipgloss.NewStyle().Foreground(lipgloss.Color("208")),
		"medium":   lipgloss.NewStyle().Foreground(lipgloss.Color("226")),
		"low":      lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
	}

	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	claimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("44")).Bold(true)

	detailLabelStyle = lipgloss.NewStyle().Bold(true).Width(14) //nolint:mnd // label column width

	dialogPadY = 1
	dialogPadX = 2

	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(dialogPadY, dialogPadX)
)

// ageStyle returns a lipgloss style for the duration label based on the
// configured age thresholds. Thresholds are walked in reverse order (longest
// first) so the first match wins.
func (b *Board) ageStyle(d time.Duration) lipgloss.Style {
	thresholds := b.cfg.AgeThresholdsDuration()
	// Walk backwards: pick the highest threshold that the duration exceeds.
	for i := len(thresholds) - 1; i >= 0; i-- {
		if d >= thresholds[i].After {
			return lipgloss.NewStyle().Foreground(lipgloss.Color(thresholds[i].Color))
		}
	}
	return dimStyle
}

// --- View rendering ---

func (b *Board) viewBoard() string {
	if len(b.columns) == 0 {
		return "No statuses configured."
	}

	// Calculate column width.
	colWidth := b.columnWidth()

	// Render columns.
	renderedCols := make([]string, len(b.columns))
	for i, col := range b.columns {
		renderedCols[i] = b.renderColumn(i, col, colWidth)
	}

	boardView := lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...)
	statusBar := b.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, boardView, "", statusBar)
}

func (b *Board) columnWidth() int {
	if b.width == 0 || len(b.columns) == 0 {
		return 30 //nolint:mnd // default column width
	}
	// Total rendered width = w * numColumns (JoinHorizontal adds no gaps).
	w := b.width / len(b.columns)
	const maxColWidth = 50
	if w > maxColWidth {
		w = maxColWidth
	}
	return w
}

func (b *Board) renderColumn(colIdx int, col column, width int) string {
	// Header.
	headerText := fmt.Sprintf("%s (%d)", col.status, len(col.tasks))
	wip := b.cfg.WIPLimit(col.status)
	if wip > 0 {
		headerText = fmt.Sprintf("%s (%d/%d)", col.status, len(col.tasks), wip)
	}
	// Truncate to fit within padding (1 left + 1 right).
	const headerPad = 2
	headerText = truncate(headerText, width-headerPad)

	var header string
	if colIdx == b.activeCol {
		header = activeColumnHeaderStyle.Width(width).Render(headerText)
	} else {
		header = columnHeaderStyle.Width(width).Render(headerText)
	}

	// Determine visible card range.
	maxVis := b.visibleCardsForColumn(&col, width)
	start := col.scrollOff
	end := start + maxVis
	if end > len(col.tasks) {
		end = len(col.tasks)
	}
	if start > len(col.tasks) {
		start = len(col.tasks)
	}

	parts := []string{header}

	// Show "↑ N more" indicator if scrolled down.
	if start > 0 {
		indicator := fmt.Sprintf("  ↑ %d more", start)
		parts = append(parts, dimStyle.Width(width).Render(truncate(indicator, width)))
	}

	// Render visible cards.
	if len(col.tasks) == 0 {
		parts = append(parts, dimStyle.Width(width).Render("  (empty)"))
	} else {
		for rowIdx := start; rowIdx < end; rowIdx++ {
			t := col.tasks[rowIdx]
			active := colIdx == b.activeCol && rowIdx == b.activeRow
			parts = append(parts, b.renderCard(t, active, width))
		}
	}

	// Show "↓ N more" indicator if more cards below.
	if end < len(col.tasks) {
		remaining := len(col.tasks) - end
		indicator := fmt.Sprintf("  ↓ %d more", remaining)
		parts = append(parts, dimStyle.Width(width).Render(truncate(indicator, width)))
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (b *Board) renderCard(t *task.Task, active bool, width int) string {
	contentLines := b.cardContentLines(t, width)
	content := strings.Join(contentLines, "\n")

	// Pick style.
	style := cardStyle
	if t.Blocked {
		style = blockedCardStyle
	}
	if active {
		style = activeCardStyle
	}

	return style.Width(width - 2).Render(content) //nolint:mnd // border width
}

func (b *Board) cardHeight(t *task.Task, width int) int {
	contentLines := b.cardContentLines(t, width)
	return len(contentLines) + 2 //nolint:mnd // top and bottom borders
}

func (b *Board) cardContentLines(t *task.Task, width int) []string {
	// Card content.
	const cardChrome = 4 // border (2) + padding (2)
	cardWidth := width - cardChrome
	if cardWidth < 1 {
		cardWidth = 1
	}

	titleLines := b.cfg.TitleLines()
	idStr := dimStyle.Render("#" + strconv.Itoa(t.ID))
	idLen := len(strconv.Itoa(t.ID)) + 1    // "#" + digits
	firstLineWidth := cardWidth - idLen - 1 // space after id
	if firstLineWidth < 1 {
		firstLineWidth = 1
	}

	var contentLines []string
	if titleLines == 1 {
		title := truncate(t.Title, firstLineWidth)
		contentLines = append(contentLines, idStr+" "+title)
	} else {
		wrapped := wrapTitle2(t.Title, firstLineWidth, cardWidth, titleLines)
		contentLines = append(contentLines, idStr+" "+wrapped[0])
		for i := 1; i < len(wrapped); i++ {
			contentLines = append(contentLines, wrapped[i])
		}
	}

	// Priority + tags line.
	var details []string
	pStyle, ok := priorityStyles[t.Priority]
	if !ok {
		pStyle = dimStyle
	}
	details = append(details, pStyle.Render(t.Priority))

	if len(t.Tags) > 0 {
		tagStr := strings.Join(t.Tags, ",")
		tagMaxLen := cardWidth / tagMaxFraction
		if len(tagStr) > tagMaxLen {
			tagStr = tagStr[:tagMaxLen-3] + "..."
		}
		details = append(details, dimStyle.Render(tagStr))
	}

	if t.Due != nil {
		details = append(details, dimStyle.Render("due:"+t.Due.String()))
	}

	ageDur := b.now().Sub(t.Updated)
	age := humanDuration(ageDur)
	details = append(details, b.ageStyle(ageDur).Render(age))

	contentLines = append(contentLines, strings.Join(details, " "))

	// Claim info on a dedicated line only for claimed tasks.
	if t.ClaimedBy != "" {
		contentLines = append(contentLines, claimStyle.Render("@"+t.ClaimedBy))
	}

	return contentLines
}

// wrapTitle2 splits a title across maxLines lines with different widths:
// firstWidth for the first line (shares space with the ID prefix),
// restWidth for continuation lines (uses full card width).
func wrapTitle2(title string, firstWidth, restWidth, maxLines int) []string {
	if maxLines < 1 {
		maxLines = 1
	}
	if len(title) <= firstWidth || maxLines == 1 {
		return []string{truncate(title, firstWidth)}
	}

	words := strings.Fields(title)
	lines := make([]string, 0, maxLines)
	var current strings.Builder

	for i, word := range words {
		lineWidth := restWidth
		if len(lines) == 0 {
			lineWidth = firstWidth
		}

		if current.Len() == 0 {
			current.WriteString(word)
			continue
		}
		if current.Len()+1+len(word) <= lineWidth {
			current.WriteByte(' ')
			current.WriteString(word)
		} else {
			lines = append(lines, truncate(current.String(), lineWidth))
			current.Reset()
			current.WriteString(word)
			if len(lines) == maxLines-1 {
				// Last line: append all remaining words.
				for _, w := range words[i+1:] {
					current.WriteByte(' ')
					current.WriteString(w)
				}
				break
			}
		}
	}
	if current.Len() > 0 {
		w := restWidth
		if len(lines) == 0 {
			w = firstWidth
		}
		lines = append(lines, truncate(current.String(), w))
	}
	return lines
}

// wrapTitle splits a title across maxLines lines, word-wrapping at word
// boundaries. Each line is at most maxWidth characters.
func wrapTitle(title string, maxWidth, maxLines int) []string {
	if maxLines < 1 {
		maxLines = 1
	}
	if len(title) <= maxWidth || maxLines == 1 {
		return []string{truncate(title, maxWidth)}
	}

	words := strings.Fields(title)
	lines := make([]string, 0, maxLines)
	var current strings.Builder

	for i, word := range words {
		if current.Len() == 0 {
			current.WriteString(word)
			continue
		}
		if current.Len()+1+len(word) <= maxWidth {
			current.WriteByte(' ')
			current.WriteString(word)
		} else {
			lines = append(lines, truncate(current.String(), maxWidth))
			current.Reset()
			current.WriteString(word)
			if len(lines) == maxLines-1 {
				// Last line: append all remaining words.
				for _, w := range words[i+1:] {
					current.WriteByte(' ')
					current.WriteString(w)
				}
				break
			}
		}
	}
	if current.Len() > 0 {
		lines = append(lines, truncate(current.String(), maxWidth))
	}
	return lines
}

func (b *Board) renderStatusBar() string {
	total := len(b.tasks)
	status := fmt.Sprintf(" %s | %d tasks | ←↓↑→/hjkl:navigate enter:detail m:move N:next P:prev d:delete ?:help esc/q:quit",
		b.cfg.Board.Name, total)
	status = truncate(status, b.width)

	if b.err != nil {
		errStr := errorStyle.Render(truncate("Error: "+b.err.Error(), b.width))
		return errStr + "\n" + statusBarStyle.Render(status)
	}

	return statusBarStyle.Render(status)
}

func (b *Board) viewDetail() string {
	t := b.detailTask
	if t == nil {
		return "No task selected."
	}

	lines := detailLines(t, b.width)

	// Reserve the last line for the fixed status hint.
	viewHeight := b.height - 1
	if viewHeight < 1 {
		viewHeight = len(lines)
	}

	// Build the status hint (always visible at bottom).
	hint := "q/esc:back"
	if len(lines) > viewHeight {
		hint += "  j/k:scroll  g/G:top/bottom"
	}

	// Apply viewport scrolling.
	off := b.detailScrollOff
	maxOff := len(lines) - viewHeight
	if maxOff < 0 {
		maxOff = 0
	}
	if off > maxOff {
		off = maxOff
	}

	end := off + viewHeight
	if end > len(lines) {
		end = len(lines)
	}

	return strings.Join(lines[off:end], "\n") + "\n" + dimStyle.Render(hint)
}

func detailLines(t *task.Task, width int) []string {
	var lines []string
	titleLine := lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("Task #%d: %s", t.ID, t.Title))
	lines = append(lines, titleLine)
	lines = append(lines, strings.Repeat("─", lipgloss.Width(titleLine)))
	lines = append(lines, "")
	lines = append(lines, detailLabelStyle.Render("Status:")+"  "+t.Status)
	lines = append(lines, detailLabelStyle.Render("Priority:")+"  "+t.Priority)

	if t.Assignee != "" {
		lines = append(lines, detailLabelStyle.Render("Assignee:")+"  "+t.Assignee)
	}
	if len(t.Tags) > 0 {
		lines = append(lines, detailLabelStyle.Render("Tags:")+"  "+strings.Join(t.Tags, ", "))
	}
	if t.Due != nil {
		lines = append(lines, detailLabelStyle.Render("Due:")+"  "+t.Due.String())
	}
	if t.Estimate != "" {
		lines = append(lines, detailLabelStyle.Render("Estimate:")+"  "+t.Estimate)
	}
	lines = append(lines, detailLabelStyle.Render("Created:")+"  "+t.Created.Format("2006-01-02 15:04"))
	lines = append(lines, detailLabelStyle.Render("Updated:")+"  "+t.Updated.Format("2006-01-02 15:04"))

	if t.ClaimedBy != "" {
		lines = append(lines, detailLabelStyle.Render("Claimed:")+"  "+claimStyle.Render(t.ClaimedBy))
	}
	if t.Started != nil {
		lines = append(lines, detailLabelStyle.Render("Started:")+"  "+t.Started.Format("2006-01-02 15:04"))
	}
	if t.Completed != nil {
		lines = append(lines, detailLabelStyle.Render("Completed:")+"  "+t.Completed.Format("2006-01-02 15:04"))
	}
	if t.Started != nil && t.Completed != nil {
		lines = append(lines, detailLabelStyle.Render("Duration:")+"  "+humanDuration(t.Completed.Sub(*t.Started)))
	}
	if t.Blocked {
		lines = append(lines, "")
		lines = append(lines, errorStyle.Render("BLOCKED: "+t.BlockReason))
	}
	if t.Body != "" {
		lines = append(lines, "")
		wrapped := lipgloss.NewStyle().Width(width).Render(t.Body)
		lines = append(lines, strings.Split(wrapped, "\n")...)
	}
	return lines
}

func (b *Board) viewMoveDialog() string {
	t := b.selectedTask()
	title := "Move task"
	if t != nil {
		title = fmt.Sprintf("Move #%d to:", t.ID)
	}

	var items []string
	for i, s := range b.moveStatuses {
		cursor := "  "
		if i == b.moveCursor {
			cursor = "> "
		}
		line := cursor + s
		if t != nil && s == t.Status {
			line += " (current)"
		}
		items = append(items, line)
	}

	content := lipgloss.NewStyle().Bold(true).Render(title) + "\n\n" +
		strings.Join(items, "\n") + "\n\n" +
		dimStyle.Render("enter:select  esc:cancel")

	return dialogStyle.Render(content)
}

func (b *Board) viewDeleteConfirm() string {
	content := errorStyle.Render("Delete task?") + "\n\n" +
		fmt.Sprintf("  #%d: %s", b.deleteID, b.deleteTitle) + "\n\n" +
		dimStyle.Render("y:yes  n:no")

	return dialogStyle.Render(content)
}

func (b *Board) viewHelp() string {
	help := []struct{ key, desc string }{
		{"h/←", "Move to left column"},
		{"l/→", "Move to right column"},
		{"j/↓", "Move cursor down"},
		{"k/↑", "Move cursor up"},
		{"enter", "Show task detail"},
		{"m", "Move task (status picker)"},
		{"N", "Move task to next status"},
		{"P", "Move task to previous status"},
		{"d", "Delete task"},
		{"r", "Refresh board"},
		{"?", "Show this help"},
		{"esc/q", "Quit"},
		{"ctrl+c", "Force quit"},
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Keyboard Shortcuts"))
	lines = append(lines, "")

	for _, h := range help {
		keyStyle := lipgloss.NewStyle().Bold(true).Width(12) //nolint:mnd // key column width
		lines = append(lines, keyStyle.Render(h.key)+"  "+h.desc)
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("Press any key to close"))

	return dialogStyle.Render(strings.Join(lines, "\n"))
}

func truncate(s string, maxLen int) string {
	if maxLen < 4 { //nolint:mnd // minimum length for truncation
		maxLen = 4
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// humanDuration formats a duration as a compact human-readable string.
// Examples: "<1m", "5m", "2h", "3d", "2w", "3mo", "1y".
func humanDuration(d time.Duration) string {
	const (
		day   = 24 * time.Hour
		week  = 7 * day
		month = 30 * day
		year  = 365 * day
	)

	switch {
	case d < time.Minute:
		return "<1m"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + "m"
	case d < day:
		return strconv.Itoa(int(d.Hours())) + "h"
	case d < week:
		return strconv.Itoa(int(d/day)) + "d"
	case d < month:
		return strconv.Itoa(int(d/week)) + "w"
	case d < year:
		return strconv.Itoa(int(d/month)) + "mo"
	default:
		return strconv.Itoa(int(d/year)) + "y"
	}
}
