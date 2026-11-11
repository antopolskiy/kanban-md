package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/watcher"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 2 * time.Second
	loopbackHost      = "127.0.0.1"
)

// Options configures the local web server.
type Options struct {
	Port             int
	OpenBrowser      bool
	HideEmptyColumns bool
	Claimant         string
	Now              func() time.Time
	Stdout           io.Writer
	OnReady          func(string)
}

// Server serves one local board.
type Server struct {
	dir       string
	hideEmpty bool
	claim     string
	clock     func() time.Time
	hub       *hub
	tmpl      *templates
}

// New prepares a server for an already loaded board.
func New(cfg *config.Config, opts Options) (*Server, error) {
	if cfg == nil {
		return nil, errors.New("missing board config")
	}
	tmpl, err := parseTemplates()
	if err != nil {
		return nil, err
	}
	return &Server{
		dir:       cfg.Dir(),
		hideEmpty: opts.HideEmptyColumns,
		claim:     opts.Claimant,
		clock:     opts.Now,
		hub:       newHub(),
		tmpl:      tmpl,
	}, nil
}

// Handler is the HTTP handler, including CSRF checks on POST.
func (s *Server) Handler() http.Handler {
	return s.protect(s.routes())
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleBoard)
	mux.HandleFunc("GET /help", s.handleHelp)
	mux.HandleFunc("GET /events", s.handleEvents)
	mux.HandleFunc("GET /tasks/new", s.handleNew)
	mux.HandleFunc("POST /tasks", s.handleCreate)
	mux.HandleFunc("GET /tasks/{id}", s.handleDetail)
	mux.HandleFunc("GET /tasks/{id}/edit", s.handleEditForm)
	mux.HandleFunc("POST /tasks/{id}", s.handleEdit)
	mux.HandleFunc("POST /tasks/{id}/move", s.handleMove)
	mux.HandleFunc("POST /tasks/{id}/priority", s.handlePriority)
	mux.HandleFunc("GET /tasks/{id}/archive", s.handleArchiveForm)
	mux.HandleFunc("POST /tasks/{id}/archive", s.handleArchive)
	static, err := fs.Sub(assets, "static")
	if err != nil {
		return mux
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	return mux
}

// Serve listens on 127.0.0.1 until ctx is canceled.
func Serve(ctx context.Context, cfg *config.Config, opts Options) error {
	s, err := New(cfg, opts)
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", listenAddr(opts.Port))
	if err != nil {
		return fmt.Errorf("listening on %s: %w", listenAddr(opts.Port), err)
	}
	rawURL := "http://" + ln.Addr().String() + "/"
	out := opts.Stdout
	if out == nil {
		out = os.Stdout
	}
	fmt.Fprintln(out, rawURL)
	if opts.OnReady != nil {
		opts.OnReady(rawURL)
	}
	if opts.OpenBrowser {
		if openErr := Open(rawURL); openErr != nil {
			fmt.Fprintf(os.Stderr, "could not open browser: %v\n", openErr)
		}
	}

	watchCtx, watchCancel := context.WithCancel(ctx)
	defer watchCancel()
	go s.watch(watchCtx)

	srv := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	go shutdownOnCancel(ctx, srv)
	err = srv.Serve(ln)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func listenAddr(port int) string {
	return net.JoinHostPort(loopbackHost, strconv.Itoa(port))
}

func shutdownOnCancel(ctx context.Context, srv *http.Server) {
	<-ctx.Done()
	shutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
}

func (s *Server) watch(ctx context.Context) {
	cfg, err := config.Load(s.dir)
	if err != nil {
		return
	}
	paths := []string{cfg.TasksPath()}
	if cfg.Dir() != cfg.TasksPath() {
		paths = append(paths, cfg.Dir())
	}
	w, err := watcher.New(paths, s.hub.notify)
	if err != nil {
		return
	}
	defer func() { _ = w.Close() }()
	w.Run(ctx, nil)
}

func (s *Server) claimant() string {
	if s.claim != "" {
		return s.claim
	}
	return defaultClaimant()
}
