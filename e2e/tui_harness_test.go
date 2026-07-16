//go:build !windows

package e2e_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/creack/pty"
)

const (
	tuiRows           = 40
	tuiCols           = 120
	tuiStartupTimeout = 3 * time.Second
	tuiExitTimeout    = 3 * time.Second
	tuiKeyDelay       = 12 * time.Millisecond
	tuiTaskTimeout    = 2 * time.Second
)

var (
	ansiCSIRe = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
	ansiOSCRe = regexp.MustCompile(`\x1b\][^\x07]*\x07`)
)

type tuiOutputBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *tuiOutputBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *tuiOutputBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (b *tuiOutputBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Len()
}

func (b *tuiOutputBuffer) StringFrom(offset int) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	raw := b.buf.String()
	if offset < 0 {
		offset = 0
	}
	if offset > len(raw) {
		offset = len(raw)
	}
	return raw[offset:]
}

type tuiSession struct {
	t       *testing.T
	cmd     *exec.Cmd
	ptmx    *os.File
	out     tuiOutputBuffer
	done    chan struct{}
	doneErr chan error
	cleanup sync.Once
}

type tuiProcessOptions struct {
	args []string
	rows uint16
	cols uint16
}

func startTUIProcess(t *testing.T, dir string) *tuiSession {
	t.Helper()
	return startTUIProcessWithOptions(t, dir, tuiProcessOptions{})
}

func startTUIProcessWithOptions(t *testing.T, dir string, options tuiProcessOptions) *tuiSession {
	t.Helper()

	rows := options.rows
	if rows == 0 {
		rows = tuiRows
	}
	cols := options.cols
	if cols == 0 {
		cols = tuiCols
	}
	args := append([]string{"--dir", dir, "tui"}, options.args...)
	cmd := exec.Command(binPath, args...) //nolint:gosec,noctx // command uses test-built binary path
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "TERM=dumb")

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		t.Fatalf("starting TUI process: %v", err)
	}

	session := &tuiSession{
		t:       t,
		cmd:     cmd,
		ptmx:    ptmx,
		done:    make(chan struct{}),
		doneErr: make(chan error, 1),
	}

	go func() {
		_, _ = io.Copy(&session.out, ptmx)
		session.doneErr <- cmd.Wait()
		close(session.done)
	}()

	t.Cleanup(session.close)
	return session
}

func (s *tuiSession) close() {
	s.cleanup.Do(func() {
		_ = s.pressKey("q")
		timer := time.NewTimer(tuiExitTimeout)
		select {
		case <-s.done:
			if !timer.Stop() {
				<-timer.C
			}
		case <-timer.C:
			_ = s.pressKey("ctrl+c")
			select {
			case <-s.done:
			case <-time.After(150 * time.Millisecond):
				if s.cmd.Process != nil {
					_ = s.cmd.Process.Kill()
				}
			}
		}

		select {
		case err := <-s.doneErr:
			_ = err
		case <-time.After(time.Second):
			if s.cmd.Process != nil {
				_ = s.cmd.Process.Kill()
			}
			select {
			case err := <-s.doneErr:
				_ = err
			default:
			}
		}
		_ = s.ptmx.Close()
	})
}

func (s *tuiSession) output() string {
	return sanitizeTTYOutput(s.out.String())
}

func (s *tuiSession) checkpoint() int {
	return s.out.Len()
}

func (s *tuiSession) outputSince(checkpoint int) string {
	return sanitizeTTYOutput(s.out.StringFrom(checkpoint))
}

func (s *tuiSession) rawOutputSince(checkpoint int) string {
	return s.out.StringFrom(checkpoint)
}

func (s *tuiSession) writeRaw(input []byte) error {
	s.t.Helper()
	_, err := s.ptmx.Write(input)
	if err == nil {
		time.Sleep(tuiKeyDelay)
	}
	return err
}

func (s *tuiSession) pressKey(name string) error {
	s.t.Helper()
	return s.writeRaw([]byte(encodeKey(name)))
}

func (s *tuiSession) pressKeys(names ...string) {
	s.t.Helper()
	for _, name := range names {
		if err := s.pressKey(name); err != nil {
			s.t.Fatalf("pressing key %q: %v", name, err)
		}
	}
}

func (s *tuiSession) typeText(text string) {
	s.t.Helper()
	for _, r := range text {
		if err := s.pressKey(string(r)); err != nil {
			s.t.Fatalf("typing text %q: %v", text, err)
		}
	}
}

func (s *tuiSession) pressBackspace(count int) {
	s.t.Helper()
	for range count {
		if err := s.pressKey("backspace"); err != nil {
			s.t.Fatalf("backspacing %d times: %v", count, err)
		}
	}
}

func (s *tuiSession) pressBackspaceRunes(value string) {
	s.pressBackspace(utf8.RuneCountInString(value))
}

func (s *tuiSession) waitForOutput(needle string) {
	s.t.Helper()
	deadline := time.Now().Add(tuiStartupTimeout)
	for time.Now().Before(deadline) {
		if strings.Contains(s.output(), needle) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	s.t.Fatalf("timed out waiting for output containing %q; got %q", needle, s.output())
}

func (s *tuiSession) waitForOutputSince(checkpoint int, needle string) {
	s.t.Helper()
	deadline := time.Now().Add(tuiStartupTimeout)
	for time.Now().Before(deadline) {
		if strings.Contains(s.outputSince(checkpoint), needle) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	s.t.Fatalf("timed out waiting for new output containing %q; got %q", needle, s.outputSince(checkpoint))
}

func (s *tuiSession) waitForRawOutput(needle string) {
	s.t.Helper()
	s.waitForRawOutputSince(0, needle)
}

func (s *tuiSession) waitForRawOutputSince(checkpoint int, needle string) {
	s.t.Helper()
	deadline := time.Now().Add(tuiStartupTimeout)
	for time.Now().Before(deadline) {
		if strings.Contains(s.rawOutputSince(checkpoint), needle) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	s.t.Fatalf("timed out waiting for raw output containing %q; got %q", needle, s.rawOutputSince(checkpoint))
}

func (s *tuiSession) resize(cols, rows uint16) {
	s.t.Helper()
	if err := pty.Setsize(s.ptmx, &pty.Winsize{Cols: cols, Rows: rows}); err != nil {
		s.t.Fatalf("resizing PTY to %dx%d: %v", cols, rows, err)
	}
	time.Sleep(tuiKeyDelay)
}

func (s *tuiSession) mouseSGR(code, x, y int, release bool) {
	s.t.Helper()
	final := byte('M')
	if release {
		final = 'm'
	}
	sequence := []byte(fmt.Sprintf("\x1b[<%d;%d;%d%c", code, x+1, y+1, final))
	if err := s.writeRaw(sequence); err != nil {
		s.t.Fatalf("writing SGR mouse event: %v", err)
	}
}

func (s *tuiSession) mouseX10(code, x, y int) {
	s.t.Helper()
	if code < 0 || code > 223 || x < 0 || x > 222 || y < 0 || y > 222 {
		s.t.Fatalf("X10 mouse coordinate/code out of range: code=%d x=%d y=%d", code, x, y)
	}
	sequence := []byte{
		0x1b, '[', 'M',
		byte(32 + code),  // #nosec G115 -- range checked above
		byte(32 + x + 1), // #nosec G115 -- range checked above
		byte(32 + y + 1), // #nosec G115 -- range checked above
	}
	if err := s.writeRaw(sequence); err != nil {
		s.t.Fatalf("writing X10 mouse event: %v", err)
	}
}

func (s *tuiSession) clickSGR(x, y int) {
	s.t.Helper()
	s.mouseSGR(0, x, y, false)
	s.mouseSGR(0, x, y, true)
}

func (s *tuiSession) clickX10(x, y int) {
	s.t.Helper()
	s.mouseX10(0, x, y)
	s.mouseX10(3, x, y)
}

func (s *tuiSession) dragSGR(sourceX, destinationX int) {
	s.t.Helper()
	s.mouseSGR(0, sourceX, 2, false)
	s.mouseSGR(32, destinationX, 0, false)
	s.mouseSGR(0, destinationX, 0, true)
}

func (s *tuiSession) dragX10(sourceX, destinationX int) {
	s.t.Helper()
	s.mouseX10(0, sourceX, 2)
	s.mouseX10(32, destinationX, 0)
	s.mouseX10(3, destinationX, 0)
}

func (s *tuiSession) wheelSGR(x, y, direction int) {
	s.t.Helper()
	code := 64
	if direction > 0 {
		code = 65
	}
	s.mouseSGR(code, x, y, false)
}

func (s *tuiSession) waitForExit() {
	s.t.Helper()
	select {
	case <-s.done:
		_ = s.waitErr()
	case <-time.After(tuiExitTimeout):
		s.t.Fatalf("timed out waiting for TUI process to exit")
	}
}

func (s *tuiSession) waitErr() error {
	select {
	case err := <-s.doneErr:
		return err
	default:
		return nil
	}
}

func encodeKey(name string) string {
	switch name {
	case "enter", "return":
		return "\r"
	case "tab":
		return "\t"
	case "esc":
		return "\x1b"
	case "shift+tab":
		return "\x1b[Z"
	case "up":
		return "\x1b[A"
	case "down":
		return "\x1b[B"
	case "left":
		return "\x1b[D"
	case "right":
		return "\x1b[C"
	case "backspace", "delete":
		return "\x7f"
	case "ctrl+c":
		return "\x03"
	default:
		return name
	}
}

func sanitizeTTYOutput(raw string) string {
	raw = strings.ReplaceAll(raw, "\r", "")
	raw = ansiCSIRe.ReplaceAllString(raw, "")
	raw = ansiOSCRe.ReplaceAllString(raw, "")
	return raw
}

func initBoardWithSeededTasks(t *testing.T) string {
	t.Helper()

	dir := initBoard(t)
	mustCreateTask(t, dir, "Task A", "--priority", "high")
	mustCreateTask(t, dir, "Task B", "--priority", "medium")
	mustCreateTask(t, dir, "Task C", "--status", "in-progress", "--priority", "high")
	mustCreateTask(t, dir, "Task D", "--status", "done", "--priority", "low")

	return dir
}

func waitForTask(t *testing.T, kanbanDir string, id int, check func(taskJSON) bool) {
	t.Helper()

	deadline := time.Now().Add(tuiTaskTimeout)
	for {
		var tk taskJSON
		r := runKanbanJSON(t, kanbanDir, &tk, "show", strconv.Itoa(id))
		if r.exitCode == 0 && check(tk) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for task update (id=%d), last seen: %#v", id, tk)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func containsTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}
