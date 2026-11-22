package attachment

import (
	"bytes"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func makeDirectory(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(path, dirMode); err != nil {
		t.Fatal(err)
	}
	return path
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // Test-owned paths under t.TempDir.
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func referenceFile(t *testing.T, tasksDir, reference string) string {
	t.Helper()
	const prefix = "[screenshot]("
	if !strings.HasPrefix(reference, prefix) || !strings.HasSuffix(reference, ")") {
		t.Fatalf("invalid Markdown reference %q", reference)
	}
	destination := strings.TrimSuffix(strings.TrimPrefix(reference, prefix), ")")
	parsed, err := url.Parse(destination)
	if err != nil || parsed.IsAbs() || parsed.RawQuery != "" || parsed.Fragment != "" {
		t.Fatalf("invalid relative destination %q: %v", destination, err)
	}
	if strings.ContainsAny(destination, "\\ \t\r\n()") {
		t.Fatalf("unescaped destination %q", destination)
	}
	return filepath.Clean(filepath.Join(tasksDir, filepath.FromSlash(parsed.Path)))
}

func TestImportSupportedFormats(t *testing.T) {
	for _, format := range []string{formatPNG, formatJPEG, formatGIF} {
		t.Run(format, func(t *testing.T) {
			board := t.TempDir()
			tasks := makeDirectory(t, filepath.Join(board, "tasks"))
			data := encodedImage(t, format)
			img := Image{Data: data, Format: format}
			reference, err := Import(board, tasks, img)
			if err != nil {
				t.Fatal(err)
			}
			path := referenceFile(t, tasks, reference)
			if filepath.Dir(path) != filepath.Join(board, attachmentDir) {
				t.Fatalf("attachment stored outside attachments directory: %s", path)
			}
			ext := "." + format
			if format == formatJPEG {
				ext = ".jpg"
			}
			if filepath.Ext(path) != ext || !bytes.Equal(readFile(t, path), data) {
				t.Fatal("image bytes or extension changed")
			}
			if runtime.GOOS != "windows" {
				info, statErr := os.Stat(path)
				if statErr != nil || info.Mode().Perm() != fileMode {
					t.Fatalf("image permissions: %v, %v", info, statErr)
				}
			}
		})
	}
}

func TestImportPreservesFileBytes(t *testing.T) {
	board := t.TempDir()
	tasks := makeDirectory(t, filepath.Join(board, "tasks"))
	data := append(encodedImage(t, formatPNG), []byte("original trailing metadata\r\n")...)
	// The source suffix deliberately differs from the encoded format.
	source := filepath.Join(t.TempDir(), "source.gif")
	if err := os.WriteFile(source, data, fileMode); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(source) //nolint:gosec // Test-owned source file under t.TempDir.
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := Read(f)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := Import(board, tasks, img)
	if err != nil {
		t.Fatal(err)
	}
	path := referenceFile(t, tasks, reference)
	if filepath.Ext(path) != ".png" || !bytes.Equal(readFile(t, path), data) {
		t.Fatal("file import changed source bytes or trusted its extension")
	}
}

func TestImportRelativePathsAndRelocation(t *testing.T) {
	for _, tasksName := range []string{"tasks", "custom/nested/task files", "outside", "tasks (#%[])/nested"} {
		t.Run(tasksName, func(t *testing.T) {
			base := t.TempDir()
			board := makeDirectory(t, filepath.Join(base, "board"))
			tasks := filepath.Join(board, tasksName)
			if tasksName == "outside" {
				tasks = filepath.Join(base, "custom-tasks")
			}
			makeDirectory(t, tasks)
			data := encodedImage(t, formatPNG)
			reference, err := Import(board, tasks, Image{Data: data, Format: formatPNG})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(readFile(t, referenceFile(t, tasks, reference)), data) {
				t.Fatal("relative reference does not resolve to original bytes")
			}
			relocated := filepath.Join(t.TempDir(), "relocated")
			if err = os.Rename(base, relocated); err != nil {
				t.Fatal(err)
			}
			relativeTasks, err := filepath.Rel(base, tasks)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(readFile(t, referenceFile(t, filepath.Join(relocated, relativeTasks), reference)), data) {
				t.Fatal("reference broke when board tree moved")
			}
		})
	}
}

func TestImportRelativeDirectoryArguments(t *testing.T) {
	t.Chdir(t.TempDir())
	makeDirectory(t, "board/nested/tasks")
	img := Image{Data: encodedImage(t, formatPNG), Format: formatPNG}
	reference, err := Import("board", "board/nested/tasks", img)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(readFile(t, referenceFile(t, "board/nested/tasks", reference)), img.Data) {
		t.Fatal("relative directory arguments produced broken reference")
	}
}

func TestImportReferenceUsesReportedSymlinkTasksPath(t *testing.T) {
	board := t.TempDir()
	actualTasks := makeDirectory(t, filepath.Join(board, "actual", "deep"))
	tasks := filepath.Join(board, "tasks")
	if err := os.Symlink(actualTasks, tasks); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	img := Image{Data: encodedImage(t, formatPNG), Format: formatPNG}
	reference, err := Import(board, tasks, img)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(readFile(t, referenceFile(t, tasks, reference)), img.Data) {
		t.Fatal("reference did not resolve from the task's reported file directory")
	}
}

func TestImportRejectsInvalidImagesWithoutFiles(t *testing.T) {
	pngData := encodedImage(t, formatPNG)
	tests := []struct {
		name string
		img  Image
	}{
		{"empty", Image{}},
		{"non-image", Image{Data: []byte("text"), Format: formatPNG}},
		{"missing format", Image{Data: pngData}},
		{"incorrect format", Image{Data: pngData, Format: formatJPEG}},
		{"format injection", Image{Data: pngData, Format: "../../task.md"}},
		{"uppercase format", Image{Data: pngData, Format: "PNG"}},
		{"malformed image", Image{Data: pngData[:33], Format: formatPNG}},
		{"oversized", Image{Data: make([]byte, MaxBytes+1), Format: formatPNG}},
		{"dimensions", Image{Data: pngDimensions(t, maxDimension+1, 1), Format: formatPNG}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board := t.TempDir()
			tasks := makeDirectory(t, filepath.Join(board, "tasks"))
			if reference, err := Import(board, tasks, tt.img); err == nil || reference != "" {
				t.Fatalf("Import invalid image: %q, %v", reference, err)
			}
			if _, err := os.Stat(filepath.Join(board, attachmentDir)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("invalid input created storage: %v", err)
			}
		})
	}
}

func TestImportDirectoryFailures(t *testing.T) {
	board := t.TempDir()
	tasks := makeDirectory(t, filepath.Join(board, "tasks"))
	blocker := filepath.Join(board, "not-directory")
	if err := os.WriteFile(blocker, []byte("keep"), fileMode); err != nil {
		t.Fatal(err)
	}
	img := Image{Data: encodedImage(t, formatPNG), Format: formatPNG}
	for _, dirs := range [][2]string{
		{"", tasks},
		{board, ""},
		{blocker, tasks},
		{board, blocker},
		{filepath.Join(board, "missing"), tasks},
		{board, filepath.Join(tasks, "missing")},
		{"bad\x00board", tasks},
		{board, "bad\x00tasks"},
	} {
		if reference, err := Import(dirs[0], dirs[1], img); err == nil || reference != "" {
			t.Fatalf("Import(%q, %q): %q, %v", dirs[0], dirs[1], reference, err)
		}
	}
	if err := os.WriteFile(filepath.Join(board, attachmentDir), []byte("keep"), fileMode); err != nil {
		t.Fatal(err)
	}
	if reference, err := Import(board, tasks, img); err == nil || reference != "" {
		t.Fatalf("Import with attachments blocker: %q, %v", reference, err)
	}
	if string(readFile(t, filepath.Join(board, attachmentDir))) != "keep" {
		t.Fatal("Import changed pre-existing file")
	}
}

func TestImportRetainsSuccessfulFilesAndTaskData(t *testing.T) {
	board := t.TempDir()
	tasks := makeDirectory(t, filepath.Join(board, "tasks"))
	for _, path := range []string{filepath.Join(board, "config.yml"), filepath.Join(tasks, "001-test.md")} {
		if err := os.WriteFile(path, []byte("original"), fileMode); err != nil {
			t.Fatal(err)
		}
	}
	img := Image{Data: encodedImage(t, formatPNG), Format: formatPNG}
	first, err := Import(board, tasks, img)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Import(board, tasks, img)
	if err != nil || first == second {
		t.Fatalf("identical imports should have distinct references: %q, %q, %v", first, second, err)
	}
	if _, err := Import(board, tasks, Image{Data: img.Data, Format: "invalid"}); err == nil {
		t.Fatal("invalid later import succeeded")
	}
	for _, reference := range []string{first, second} {
		if !bytes.Equal(readFile(t, referenceFile(t, tasks, reference)), img.Data) {
			t.Fatal("successful import was removed or changed")
		}
	}
	for _, path := range []string{filepath.Join(board, "config.yml"), filepath.Join(tasks, "001-test.md")} {
		if string(readFile(t, path)) != "original" {
			t.Fatalf("Import changed %s", path)
		}
	}
}

func TestImportConcurrentUniqueFiles(t *testing.T) {
	board := t.TempDir()
	tasks := makeDirectory(t, filepath.Join(board, "tasks"))
	img := Image{Data: encodedImage(t, formatPNG), Format: formatPNG}
	const count = 32
	type result struct {
		reference string
		err       error
	}
	results := make(chan result, count)
	var workers sync.WaitGroup
	for range count {
		workers.Go(func() {
			reference, err := Import(board, tasks, img)
			results <- result{reference, err}
		})
	}
	workers.Wait()
	close(results)
	seen := make(map[string]bool)
	for got := range results {
		if got.err != nil {
			t.Fatal(got.err)
		}
		if seen[got.reference] {
			t.Fatalf("duplicate reference %q", got.reference)
		}
		seen[got.reference] = true
		if !bytes.Equal(readFile(t, referenceFile(t, tasks, got.reference)), img.Data) {
			t.Fatal("concurrent import has incomplete data")
		}
	}
	entries, err := os.ReadDir(filepath.Join(board, attachmentDir))
	if err != nil || len(entries) != count {
		t.Fatalf("got %d files, want %d: %v", len(entries), count, err)
	}
}

type failingFile struct {
	*os.File
	writeErr error
	closeErr error
}

func (f failingFile) Write(data []byte) (int, error) {
	n, err := f.File.Write(data[:1])
	return n, errors.Join(err, f.writeErr)
}

func (f failingFile) Close() error { return errors.Join(f.File.Close(), f.closeErr) }

type closeFailingFile struct{ failingFile }

func (f closeFailingFile) Write(data []byte) (int, error) { return f.File.Write(data) }

func TestWriteFailureCleanup(t *testing.T) {
	want := errors.New("disk failure")
	const shortWrite = "short write"
	for _, failure := range []string{"write", shortWrite, "close"} {
		t.Run(failure, func(t *testing.T) {
			root, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			file, err := root.OpenFile("partial.png", os.O_CREATE|os.O_EXCL|os.O_WRONLY, fileMode)
			if err != nil {
				t.Fatal(err)
			}
			var writer io.WriteCloser = failingFile{File: file, writeErr: want}
			if failure == shortWrite {
				writer = failingFile{File: file}
			}
			if failure == "close" {
				writer = closeFailingFile{failingFile{File: file, closeErr: want}}
			}
			err = writeImage(root, "partial.png", writer, []byte("image bytes"))
			expected := want
			if failure == shortWrite {
				expected = io.ErrShortWrite
			}
			if !errors.Is(err, expected) {
				t.Fatalf("writeImage returned %v, want %v", err, expected)
			}
			if _, err := root.Stat("partial.png"); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("incomplete image remains: %v", err)
			}
			if _, err := file.Write([]byte("closed")); !errors.Is(err, os.ErrClosed) {
				t.Fatalf("file was not closed: %v", err)
			}
		})
	}
}

func TestImportColonInFirstRelativeComponent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows directory names cannot contain colons")
	}
	tasks := t.TempDir()
	board := makeDirectory(t, filepath.Join(tasks, "board:images"))
	img := Image{Data: encodedImage(t, formatPNG), Format: formatPNG}
	reference, err := Import(board, tasks, img)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(readFile(t, referenceFile(t, tasks, reference)), img.Data) {
		t.Fatal("colon in board path created a broken reference")
	}
}

func TestImportRejectsAttachmentSymlinks(t *testing.T) {
	for _, inside := range []bool{true, false} {
		t.Run(map[bool]string{true: "inside board", false: "outside board"}[inside], func(t *testing.T) {
			board := t.TempDir()
			tasks := makeDirectory(t, filepath.Join(board, "tasks"))
			target := t.TempDir()
			link := target
			if inside {
				target, link = tasks, "tasks"
			}
			if err := os.Symlink(link, filepath.Join(board, attachmentDir)); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			img := Image{Data: encodedImage(t, formatPNG), Format: formatPNG}
			if reference, err := Import(board, tasks, img); err == nil || reference != "" {
				t.Fatalf("Import followed attachments symlink: %q, %v", reference, err)
			}
			entries, err := os.ReadDir(target)
			if err != nil || len(entries) != 0 {
				t.Fatalf("Import wrote through symlink: %v, %v", entries, err)
			}
		})
	}
}

func TestImportWithSymlinkedDirectoryArguments(t *testing.T) {
	base := t.TempDir()
	board := makeDirectory(t, filepath.Join(base, "board"))
	tasks := makeDirectory(t, filepath.Join(base, "real-tasks"))
	boardLink, tasksLink := filepath.Join(base, "board-link"), filepath.Join(board, "tasks-link")
	for alias, target := range map[string]string{boardLink: board, tasksLink: tasks} {
		if err := os.Symlink(target, alias); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}
	img := Image{Data: encodedImage(t, formatPNG), Format: formatPNG}
	reference, err := Import(boardLink, tasksLink, img)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(readFile(t, referenceFile(t, tasksLink, reference)), img.Data) {
		t.Fatal("symlink aliases produced a broken relative reference")
	}
}
