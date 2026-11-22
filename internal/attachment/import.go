package attachment

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
)

const (
	dirMode       = 0o750
	fileMode      = 0o600
	attachmentDir = "attachments"
)

// Import stores an image in boardDir/attachments and returns an ordinary
// [screenshot](relative/path) Markdown link from tasksDir. Both directories must
// exist; relative directory arguments are resolved from the working directory.
// Image bytes and format are revalidated, and successful imports are retained.
func Import(boardDir, tasksDir string, img Image) (string, error) {
	format, err := validate(img.Data)
	if err != nil {
		return "", err
	}
	if img.Format != format {
		return "", fmt.Errorf("image format %q does not match detected format %q", img.Format, format)
	}
	boardPath, err := directoryPath(boardDir)
	if err != nil {
		return "", fmt.Errorf("resolving board directory: %w", err)
	}
	_, err = directoryPath(tasksDir)
	if err != nil {
		return "", fmt.Errorf("resolving tasks directory: %w", err)
	}
	ext := "." + format
	if format == formatJPEG {
		ext = ".jpg"
	}
	name := "screenshot-" + rand.Text() + ext
	reference, err := relativeReference(boardDir, tasksDir, name)
	if err != nil {
		return "", fmt.Errorf("creating attachment reference: %w", err)
	}
	board, err := os.OpenRoot(boardPath)
	if err != nil {
		return "", fmt.Errorf("opening board directory: %w", err)
	}
	defer board.Close()
	attachments, err := openAttachments(board)
	if err != nil {
		return "", err
	}
	defer attachments.Close()
	f, err := attachments.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
	if err != nil {
		return "", fmt.Errorf("creating image attachment: %w", err)
	}
	if err := writeImage(attachments, name, f, img.Data); err != nil {
		return "", err
	}
	return reference, nil
}

// Link addresses use the caller-visible task path, just like task.File. Keep
// this separate from canonicalizing and rooting the storage directory.
func relativeReference(boardDir, tasksDir, name string) (string, error) {
	boardPath, err := filepath.Abs(boardDir)
	if err != nil {
		return "", err
	}
	tasksPath, err := filepath.Abs(tasksDir)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(tasksPath, filepath.Join(boardPath, attachmentDir, name))
	if err != nil {
		return "", err
	}
	destination := (&url.URL{Path: filepath.ToSlash(relative)}).String()
	return "[screenshot](" + destination + ")", nil
}

func directoryPath(name string) (string, error) {
	if name == "" {
		return "", errors.New("directory path is empty")
	}
	abs, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return resolved, nil
}

func openAttachments(board *os.Root) (*os.Root, error) {
	if err := board.Mkdir(attachmentDir, dirMode); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("creating attachments directory: %w", err)
	}
	info, err := board.Lstat(attachmentDir)
	if err != nil {
		return nil, fmt.Errorf("checking attachments directory: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("attachments path must be a directory, not a file or symbolic link")
	}
	root, err := board.OpenRoot(attachmentDir)
	if err != nil {
		return nil, fmt.Errorf("opening attachments directory: %w", err)
	}
	return root, nil
}

func writeImage(root *os.Root, name string, f io.WriteCloser, data []byte) error {
	n, writeErr := f.Write(data)
	if writeErr == nil && n != len(data) {
		writeErr = io.ErrShortWrite
	}
	if err := errors.Join(writeErr, f.Close()); err != nil {
		return fmt.Errorf("writing image attachment: %w", errors.Join(err, root.Remove(name)))
	}
	return nil
}
