package web

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func fixedNow() time.Time {
	return time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
}

func newTestServer(t *testing.T, hideEmpty bool) (*Server, *config.Config) {
	t.Helper()
	dir := t.TempDir()
	cfg, err := config.Init(filepath.Join(dir, "kanban"), "Web")
	if err != nil {
		t.Fatal(err)
	}
	srv, err := New(cfg, Options{
		HideEmptyColumns: hideEmpty,
		Claimant:         "web@test",
		Now:              fixedNow,
	})
	if err != nil {
		t.Fatal(err)
	}
	return srv, cfg
}

func mustCreate(t *testing.T, cfg *config.Config, title, status string, tags []string) *task.Task {
	t.Helper()
	result, err := board.Create(cfg, board.CreateParams{
		Title:  title,
		Status: status,
		Tags:   tags,
	}, fixedNow())
	if err != nil {
		t.Fatal(err)
	}
	return result.Task
}

func testClient(t *testing.T, handler http.Handler) (*httptest.Server, *http.Client) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := ts.Client()
	client.Jar = jar
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return ts, client
}

func readBody(t *testing.T, res *http.Response) string {
	t.Helper()
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func csrfFrom(t *testing.T, client *http.Client, rawURL string) string {
	t.Helper()
	res, err := client.Get(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(t, res)
	const needle = `name="csrf" value="`
	i := strings.Index(body, needle)
	if i < 0 {
		t.Fatalf("csrf field missing in %s", body)
	}
	rest := body[i+len(needle):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		t.Fatal("csrf value not closed")
	}
	return rest[:end]
}

func postForm(t *testing.T, client *http.Client, rawURL, token string, form url.Values) *http.Response {
	t.Helper()
	form.Set("csrf", token)
	res, err := client.PostForm(rawURL, form)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestBoardRendersTaskAndSearch(t *testing.T) {
	srv, cfg := newTestServer(t, false)
	created := mustCreate(t, cfg, "Alpha unique", "todo", []string{"web"})
	created.Body = "See the whole note"
	if err := task.Write(created.File, created); err != nil {
		t.Fatal(err)
	}
	mustCreate(t, cfg, "Beta", "todo", nil)
	ts, client := testClient(t, srv.Handler())

	res, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	if res.Header.Get("Referrer-Policy") != "same-origin" {
		t.Fatalf("Referrer-Policy = %q", res.Header.Get("Referrer-Policy"))
	}
	body := readBody(t, res)
	if !strings.Contains(body, "Alpha unique") || !strings.Contains(body, "Beta") {
		t.Fatalf("board missing tasks: %s", body)
	}
	if !strings.Contains(body, `title="Alpha unique"`) || !strings.Contains(body, `title="See the whole note"`) {
		t.Fatal("card title and description are missing hover text")
	}
	if !strings.Contains(body, "Recent activity") {
		t.Fatal("missing activity section")
	}
	if !strings.Contains(body, `id="theme-toggle"`) || !strings.Contains(body, "/static/theme.js") {
		t.Fatal("missing theme toggle")
	}

	res, err = client.Get(ts.URL + "/?q=unique")
	if err != nil {
		t.Fatal(err)
	}
	body = readBody(t, res)
	if !strings.Contains(body, ">Alpha unique</a>") || strings.Contains(body, ">Beta</a>") {
		t.Fatalf("search filter failed: %s", body)
	}

	res, err = client.Get(ts.URL + "/?tag=web")
	if err != nil {
		t.Fatal(err)
	}
	body = readBody(t, res)
	if !strings.Contains(body, "Alpha unique") || strings.Contains(body, ">Beta<") {
		t.Fatalf("tag filter failed: %s", body)
	}
}

func TestCreateAndArchive(t *testing.T) {
	srv, cfg := newTestServer(t, false)
	ts, client := testClient(t, srv.Handler())
	token := csrfFrom(t, client, ts.URL+"/")

	res := postForm(t, client, ts.URL+"/tasks", token, url.Values{
		"title":    {"Created card"},
		"status":   {"todo"},
		"priority": {"high"},
		"tags":     {"alpha, beta"},
		"return":   {"/"},
	})
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("create status = %d body %s", res.StatusCode, readBody(t, res))
	}
	res.Body.Close()

	created := findByTitle(t, cfg, "Created card")
	if created.Status != "todo" || created.Priority != "high" || len(created.Tags) != 2 {
		t.Fatalf("created = %+v", created)
	}

	res = postForm(t, client, ts.URL+"/tasks/"+strconv.Itoa(created.ID)+"/archive", token, url.Values{
		"return": {"/"},
	})
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("archive status = %d body %s", res.StatusCode, readBody(t, res))
	}
	res.Body.Close()
	archived := reread(t, cfg, created.ID)
	if archived.Status != "archived" {
		t.Fatalf("status = %s", archived.Status)
	}
}

func TestMovePreservesClaimAndAutoClaims(t *testing.T) {
	_, cfg := newTestServer(t, false)
	srv, err := New(cfg, Options{Claimant: "web@test", Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	ts, client := testClient(t, srv.Handler())
	token := csrfFrom(t, client, ts.URL+"/")

	claimed := mustCreate(t, cfg, "Claimed", "todo", nil)
	stampClaim(t, claimed, "agent")
	res := postForm(t, client, ts.URL+"/tasks/"+strconv.Itoa(claimed.ID)+"/move", token, url.Values{
		"status": {"backlog"},
		"return": {"/"},
	})
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("move status = %d body %s", res.StatusCode, readBody(t, res))
	}
	res.Body.Close()
	moved := reread(t, cfg, claimed.ID)
	if moved.Status != "backlog" || moved.ClaimedBy != "agent" {
		t.Fatalf("move preserved claim: status %s claim %s", moved.Status, moved.ClaimedBy)
	}

	open := mustCreate(t, cfg, "Needs claim", "backlog", nil)
	res = postForm(t, client, ts.URL+"/tasks/"+strconv.Itoa(open.ID)+"/move", token, url.Values{
		"status": {"in-progress"},
		"return": {"/"},
	})
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("auto-claim move status = %d body %s", res.StatusCode, readBody(t, res))
	}
	res.Body.Close()
	progress := reread(t, cfg, open.ID)
	if progress.ClaimedBy != "web@test" {
		t.Fatalf("claimed_by = %q", progress.ClaimedBy)
	}
}

func TestEditRejectsForeignClaim(t *testing.T) {
	srv, cfg := newTestServer(t, false)
	ts, client := testClient(t, srv.Handler())
	token := csrfFrom(t, client, ts.URL+"/")
	claimed := mustCreate(t, cfg, "Claimed", "todo", nil)
	stampClaim(t, claimed, "agent")

	res := postForm(t, client, ts.URL+"/tasks/"+strconv.Itoa(claimed.ID), token, url.Values{
		"title":    {"Stolen"},
		"priority": {claimed.Priority},
		"return":   {"/"},
	})
	body := readBody(t, res)
	if res.StatusCode != http.StatusConflict || !strings.Contains(body, "claimed") {
		t.Fatalf("edit claim status = %d body %s", res.StatusCode, body)
	}
}

func TestRejectsCrossSitePost(t *testing.T) {
	srv, _ := newTestServer(t, false)
	ts, client := testClient(t, srv.Handler())

	res, err := client.PostForm(ts.URL+"/tasks", url.Values{"title": {"Nope"}, "return": {"/"}})
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(t, res)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("missing csrf status = %d body %s", res.StatusCode, body)
	}

	token := csrfFrom(t, client, ts.URL+"/")
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/tasks", strings.NewReader(url.Values{
		"csrf":   {token},
		"title":  {"Nope"},
		"return": {"/"},
	}.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "null")
	res, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body = readBody(t, res)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("null origin status = %d body %s", res.StatusCode, body)
	}

	req, err = http.NewRequest(http.MethodPost, ts.URL+"/tasks", strings.NewReader(url.Values{
		"csrf":   {token},
		"title":  {"Nope"},
		"return": {"/"},
	}.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.example")
	res, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body = readBody(t, res)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("evil origin status = %d body %s", res.StatusCode, body)
	}
}

func TestDetailSanitizesMarkdown(t *testing.T) {
	srv, cfg := newTestServer(t, false)
	created := mustCreate(t, cfg, "Docs", "todo", nil)
	created.Body = "Hello <script>alert(1)</script>\n\n# Heading"
	if err := task.Write(created.File, created); err != nil {
		t.Fatal(err)
	}
	ts, client := testClient(t, srv.Handler())
	res, err := client.Get(ts.URL + "/tasks/" + strconv.Itoa(created.ID))
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(t, res)
	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Fatalf("script was not sanitized: %s", body)
	}
	if !strings.Contains(body, "Hello") || !strings.Contains(body, "<h1") {
		t.Fatalf("markdown missing: %s", body)
	}
}

func TestHideEmptyColumns(t *testing.T) {
	srv, cfg := newTestServer(t, true)
	mustCreate(t, cfg, "Only todo", "todo", nil)
	ts, client := testClient(t, srv.Handler())
	res, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(t, res)
	if strings.Contains(body, `data-column="backlog"`) || !strings.Contains(body, `data-column="todo"`) {
		t.Fatalf("empty columns were not hidden: %s", body)
	}
}

func TestServeBindsLoopback(t *testing.T) {
	_, cfg := newTestServer(t, false)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ready := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- Serve(ctx, cfg, Options{
			Port:        0,
			OpenBrowser: false,
			OnReady:     func(rawURL string) { ready <- rawURL },
		})
	}()
	select {
	case rawURL := <-ready:
		if !strings.HasPrefix(rawURL, "http://127.0.0.1:") {
			t.Fatalf("url = %s", rawURL)
		}
	case err := <-errCh:
		t.Fatal(err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not become ready")
	}
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop")
	}
}

func findByTitle(t *testing.T, cfg *config.Config, title string) *task.Task {
	t.Helper()
	tasks, _, err := task.ReadAllLenient(cfg.TasksPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, tk := range tasks {
		if tk.Title == title {
			return tk
		}
	}
	t.Fatalf("task %q not found", title)
	return nil
}

func reread(t *testing.T, cfg *config.Config, id int) *task.Task {
	t.Helper()
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		t.Fatal(err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	return tk
}

func stampClaim(t *testing.T, tk *task.Task, claimant string) {
	t.Helper()
	now := time.Now()
	tk.ClaimedBy = claimant
	tk.ClaimedAt = &now
	if err := task.Write(tk.File, tk); err != nil {
		t.Fatal(err)
	}
}
