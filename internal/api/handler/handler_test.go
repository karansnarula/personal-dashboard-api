package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/karansnarula/personal-dashboard-api/internal/api"
	"github.com/karansnarula/personal-dashboard-api/internal/domain"
	"github.com/karansnarula/personal-dashboard-api/internal/service"
	"github.com/karansnarula/personal-dashboard-api/internal/widget"
)

// The handler tests exercise the real router, middleware and handlers with
// in-memory fakes standing in for the database and external APIs.

type memUsers struct {
	byEmail map[string]domain.User
	next    int64
}

func (m *memUsers) Create(_ context.Context, email, hash string) (domain.User, error) {
	if _, ok := m.byEmail[email]; ok {
		return domain.User{}, domain.ErrEmailTaken
	}
	m.next++
	u := domain.User{ID: m.next, Email: email, PasswordHash: hash}
	m.byEmail[email] = u
	return u, nil
}

func (m *memUsers) GetByEmail(_ context.Context, email string) (domain.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

type memWidgets struct {
	byID map[int64]domain.Widget
	next int64
}

func (m *memWidgets) Create(_ context.Context, userID int64, t domain.WidgetType, cfg json.RawMessage) (domain.Widget, error) {
	m.next++
	w := domain.Widget{ID: m.next, UserID: userID, Type: t, Config: cfg}
	m.byID[w.ID] = w
	return w, nil
}

func (m *memWidgets) ListByUser(_ context.Context, userID int64) ([]domain.Widget, error) {
	out := []domain.Widget{}
	for id := int64(1); id <= m.next; id++ {
		if w, ok := m.byID[id]; ok && w.UserID == userID {
			out = append(out, w)
		}
	}
	return out, nil
}

func (m *memWidgets) GetByID(_ context.Context, id, userID int64) (domain.Widget, error) {
	w, ok := m.byID[id]
	if !ok || w.UserID != userID {
		return domain.Widget{}, domain.ErrNotFound
	}
	return w, nil
}

func (m *memWidgets) UpdateConfig(ctx context.Context, id, userID int64, cfg json.RawMessage) (domain.Widget, error) {
	w, err := m.GetByID(ctx, id, userID)
	if err != nil {
		return domain.Widget{}, err
	}
	w.Config = cfg
	m.byID[id] = w
	return w, nil
}

func (m *memWidgets) Delete(ctx context.Context, id, userID int64) error {
	if _, err := m.GetByID(ctx, id, userID); err != nil {
		return err
	}
	delete(m.byID, id)
	return nil
}

// fakeTokens issues "user:<id>" tokens so tests need no signing key.
type fakeTokens struct{}

func (fakeTokens) Issue(userID int64) (string, error) {
	return "user:" + strconv.FormatInt(userID, 10), nil
}
func (fakeTokens) TTL() time.Duration { return time.Hour }
func (fakeTokens) Parse(token string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimPrefix(token, "user:"), 10, 64)
	if err != nil || !strings.HasPrefix(token, "user:") {
		return 0, errors.New("bad token")
	}
	return id, nil
}

type okPinger struct{}

func (okPinger) Ping(context.Context) error { return nil }

type stubClient struct{ data any }

func (s stubClient) Fetch(context.Context, json.RawMessage) (any, error) { return s.data, nil }

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	users := &memUsers{byEmail: map[string]domain.User{}}
	widgets := &memWidgets{byID: map[int64]domain.Widget{}}
	tokens := fakeTokens{}

	return api.NewRouter(api.Config{Development: false}, api.Deps{
		Logger:        logger,
		DB:            okPinger{},
		Tokens:        tokens,
		AuthService:   service.NewAuthService(users, tokens),
		WidgetService: service.NewWidgetService(widgets),
		DashboardService: service.NewDashboardService(widgets, widget.Registry{
			domain.WidgetCurrency: stubClient{data: map[string]float64{"rate": 0.9}},
		}, time.Second, logger),
	})
}

func do(t *testing.T, h http.Handler, method, path, token, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var out map[string]any
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("%s %s: non-JSON body: %s", method, path, rec.Body.String())
		}
	}
	return rec.Code, out
}

func errCode(out map[string]any) string {
	e, _ := out["error"].(map[string]any)
	code, _ := e["code"].(string)
	return code
}

func login(t *testing.T, h http.Handler, email string) string {
	t.Helper()
	creds := `{"email":"` + email + `","password":"hunter2hunter2"}`
	if code, out := do(t, h, "POST", "/api/v1/auth/register", "", creds); code != 201 {
		t.Fatalf("register: %d %v", code, out)
	}
	code, out := do(t, h, "POST", "/api/v1/auth/login", "", creds)
	if code != 200 {
		t.Fatalf("login: %d %v", code, out)
	}
	return out["token"].(string)
}

func TestHealth(t *testing.T) {
	h := newTestRouter(t)
	if code, out := do(t, h, "GET", "/api/v1/healthz", "", ""); code != 200 || out["status"] != "ok" {
		t.Fatalf("health: %d %v", code, out)
	}
}

func TestAuthFlow(t *testing.T) {
	h := newTestRouter(t)

	cases := []struct {
		name string
		body string
		code int
		err  string
	}{
		{"empty body", ``, 400, "bad_request"},
		{"missing password", `{"email":"a@b.com"}`, 400, "bad_request"},
		{"unknown field", `{"email":"a@b.com","password":"hunter2hunter2","x":1}`, 400, "bad_request"},
		{"short password", `{"email":"a@b.com","password":"short"}`, 400, "validation_error"},
		{"bad email", `{"email":"nope","password":"hunter2hunter2"}`, 400, "validation_error"},
		{"ok", `{"email":"A@B.com","password":"hunter2hunter2"}`, 201, ""},
		{"duplicate", `{"email":"a@b.com","password":"hunter2hunter2"}`, 409, "conflict"},
	}
	for _, tc := range cases {
		code, out := do(t, h, "POST", "/api/v1/auth/register", "", tc.body)
		if code != tc.code || errCode(out) != tc.err {
			t.Errorf("register %s: got %d %v, want %d %s", tc.name, code, out, tc.code, tc.err)
		}
	}

	if code, out := do(t, h, "POST", "/api/v1/auth/login", "", `{"email":"a@b.com","password":"wrongwrong"}`); code != 401 || errCode(out) != "unauthorized" {
		t.Fatalf("wrong password: %d %v", code, out)
	}
	code, out := do(t, h, "POST", "/api/v1/auth/login", "", `{"email":"a@b.com","password":"hunter2hunter2"}`)
	if code != 200 || out["token"] != "user:1" || out["token_type"] != "Bearer" || out["expires_in"] != float64(3600) {
		t.Fatalf("login: %d %v", code, out)
	}

	if code, _ := do(t, h, "GET", "/api/v1/widgets", "", ""); code != 401 {
		t.Fatalf("no token: %d", code)
	}
	if code, _ := do(t, h, "GET", "/api/v1/widgets", "garbage", ""); code != 401 {
		t.Fatalf("bad token: %d", code)
	}
	if code, out := do(t, h, "POST", "/api/v1/auth/logout", "", ""); code != 200 || out["message"] == "" {
		t.Fatalf("logout: %d %v", code, out)
	}
}

func TestWidgetCRUD(t *testing.T) {
	h := newTestRouter(t)
	token := login(t, h, "a@b.com")

	for _, body := range []string{
		`{"type":"weather","config":{"city":""}}`,
		`{"type":"news","config":{"keyword":""}}`,
		`{"type":"stock","config":{}}`,
		`{"type":"currency","config":{"base":"USD","target":"USD"}}`,
		`{"type":"crypto","config":{"coin":"BTC"}}`,
	} {
		if code, out := do(t, h, "POST", "/api/v1/widgets", token, body); code != 400 || errCode(out) != "validation_error" {
			t.Errorf("%s: got %d %v, want 400 validation_error", body, code, out)
		}
	}
	if code, out := do(t, h, "POST", "/api/v1/widgets", token, `{"type":"weather"}`); code != 400 || errCode(out) != "bad_request" {
		t.Errorf("missing config: got %d %v", code, out)
	}

	code, out := do(t, h, "POST", "/api/v1/widgets", token, `{"type":"stock","config":{"ticker":"aapl"}}`)
	if code != 201 {
		t.Fatalf("create: %d %v", code, out)
	}
	id := strconv.Itoa(int(out["id"].(float64)))
	if out["config"].(map[string]any)["ticker"] != "AAPL" {
		t.Fatalf("ticker not normalized: %v", out["config"])
	}

	if code, out := do(t, h, "GET", "/api/v1/widgets", token, ""); code != 200 || len(out["widgets"].([]any)) != 1 {
		t.Fatalf("list: %d %v", code, out)
	}
	if code, out := do(t, h, "GET", "/api/v1/widgets/"+id, token, ""); code != 200 || out["type"] != "stock" {
		t.Fatalf("get: %d %v", code, out)
	}
	if code, _ := do(t, h, "GET", "/api/v1/widgets/abc", token, ""); code != 400 {
		t.Fatalf("non-numeric id: %d", code)
	}

	if code, out := do(t, h, "PATCH", "/api/v1/widgets/"+id, token, `{"config":{"ticker":"msft"}}`); code != 200 || out["config"].(map[string]any)["ticker"] != "MSFT" {
		t.Fatalf("patch: %d %v", code, out)
	}
	if code, out := do(t, h, "PATCH", "/api/v1/widgets/"+id, token, `{"config":{"city":"Oslo"}}`); code != 400 || errCode(out) != "validation_error" {
		t.Fatalf("patch wrong shape: %d %v", code, out)
	}

	other := login(t, h, "c@d.com")
	if code, _ := do(t, h, "GET", "/api/v1/widgets/"+id, other, ""); code != 404 {
		t.Fatalf("cross-user get: %d", code)
	}
	if code, _ := do(t, h, "DELETE", "/api/v1/widgets/"+id, other, ""); code != 404 {
		t.Fatalf("cross-user delete: %d", code)
	}

	if code, _ := do(t, h, "DELETE", "/api/v1/widgets/"+id, token, ""); code != 204 {
		t.Fatalf("delete: %d", code)
	}
	if code, _ := do(t, h, "GET", "/api/v1/widgets/"+id, token, ""); code != 404 {
		t.Fatalf("get after delete: %d", code)
	}
}

func TestDashboard(t *testing.T) {
	h := newTestRouter(t)
	token := login(t, h, "a@b.com")

	do(t, h, "POST", "/api/v1/widgets", token, `{"type":"currency","config":{"base":"USD","target":"EUR"}}`)
	do(t, h, "POST", "/api/v1/widgets", token, `{"type":"weather","config":{"city":"Oslo"}}`) // no client registered

	code, out := do(t, h, "GET", "/api/v1/dashboard", token, "")
	if code != 200 || out["mode"] != "sequential" || out["widget_count"] != float64(2) {
		t.Fatalf("dashboard: %d %v", code, out)
	}
	results := out["widgets"].([]any)
	first, second := results[0].(map[string]any), results[1].(map[string]any)
	if first["status"] != "ok" || first["data"] == nil {
		t.Errorf("currency: %v", first)
	}
	if second["status"] != "error" || second["error"] == "" {
		t.Errorf("weather: %v", second)
	}
}
