package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/transport"
	"github.com/google/uuid"
	v1 "planstar/admin/api/kratos/admin/v1"
)

// errorBody mirrors the wire shape of kratos errors.Status.
type errorBody struct {
	Code    int32  `json:"code"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

func testConfig() Config {
	return Config{
		Secret:       "test-secret-with-enough-entropy",
		CookieName:   "psAdminToken",
		CookieDomain: "example.com",
		CookieSecure: true,
		TokenTTL:     time.Hour,
	}
}

// fakeRevocation is an in-memory RevocationStore.
type fakeRevocation struct {
	revoked map[string]bool
	err     error
}

func (f *fakeRevocation) Revoke(context.Context, string, time.Duration) error { return f.err }

func (f *fakeRevocation) Revoked(_ context.Context, jti string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.revoked[jti], nil
}

// serve runs the middleware over a handler that records whether it was reached.
func serve(t *testing.T, cfg Config, store RevocationStore, req *http.Request) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	reached := false
	handler := Middleware(cfg, store)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		reached = true
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec, reached
}

// The middleware runs outside the kratos handler chain, so it has to invoke the
// error encoder itself. These tests pin the body down as structured protojson:
// writing plain text here would leave clients unable to read code / reason.
func TestMiddlewareRejectsMissingCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/admins/current", nil)
	req.Header.Set("Accept", "application/protojson")

	rec, reached := serve(t, testConfig(), nil, req)

	if reached {
		t.Fatal("handler was reached without a credential")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/protojson" {
		t.Fatalf("Content-Type = %q, want application/protojson", ct)
	}
	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON (%v): %s", err, rec.Body.String())
	}
	if body.Code != http.StatusUnauthorized {
		t.Errorf("body code = %d, want %d", body.Code, http.StatusUnauthorized)
	}
	if want := v1.ErrorReason_UNAUTHENTICATED.String(); body.Reason != want {
		t.Errorf("body reason = %q, want %q", body.Reason, want)
	}
}

func TestMiddlewareRejectsInvalidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/admins/current", nil)
	req.Header.Set("Accept", "application/protojson")
	req.AddCookie(&http.Cookie{Name: testConfig().CookieName, Value: "not-a-jwt"})

	rec, reached := serve(t, testConfig(), nil, req)

	if reached {
		t.Fatal("handler was reached with an invalid token")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/protojson" {
		t.Fatalf("Content-Type = %q, want application/protojson", ct)
	}
	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON (%v): %s", err, rec.Body.String())
	}
	if body.Code != http.StatusUnauthorized {
		t.Errorf("body code = %d, want %d", body.Code, http.StatusUnauthorized)
	}
	if want := v1.ErrorReason_UNAUTHENTICATED.String(); body.Reason != want {
		t.Errorf("body reason = %q, want %q", body.Reason, want)
	}
}

// The reason constants in this package are declared locally so `pkg` stays free
// of any domain's generated proto. That freedom is only safe if the two stay in
// step, which is what this test enforces.
func TestReasonsMatchAPIEnum(t *testing.T) {
	if want := v1.ErrorReason_UNAUTHENTICATED.String(); reasonUnauthenticated != want {
		t.Errorf("reasonUnauthenticated = %q, want %q", reasonUnauthenticated, want)
	}
	if want := v1.ErrorReason_PERMISSION_DENIED.String(); reasonPermissionDenied != want {
		t.Errorf("reasonPermissionDenied = %q, want %q", reasonPermissionDenied, want)
	}
	if want := v1.ErrorReason_PASSWORD_CHANGE_REQUIRED.String(); reasonPasswordChangeRequired != want {
		t.Errorf("reasonPasswordChangeRequired = %q, want %q", reasonPasswordChangeRequired, want)
	}
}

func TestMiddlewareAllowsValidToken(t *testing.T) {
	cfg := testConfig()
	token, err := GenerateToken(uuid.New(), RoleAdmin, cfg.Secret, time.Now().Add(time.Hour), false)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/admins/current", nil)
	req.AddCookie(&http.Cookie{Name: cfg.CookieName, Value: token})

	rec, reached := serve(t, cfg, nil, req)

	if !reached {
		t.Fatalf("handler was not reached, status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestMiddlewareSkipsNoAuthPaths(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/admins/login", nil)

	_, reached := serve(t, testConfig(), nil, req)

	if !reached {
		t.Fatal("login path should not require authentication")
	}
}

func TestMiddlewarePassesOptions(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/v1/admins/list", nil)

	_, reached := serve(t, testConfig(), nil, req)

	if !reached {
		t.Fatal("preflight should not require authentication")
	}
}

func TestMiddlewareRejectsRevokedToken(t *testing.T) {
	cfg := testConfig()
	token, err := GenerateToken(uuid.New(), RoleAdmin, cfg.Secret, time.Now().Add(time.Hour), false)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	claims, err := ParseToken(token, cfg.Secret)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	store := &fakeRevocation{revoked: map[string]bool{claims.ID: true}}
	req := httptest.NewRequest(http.MethodGet, "/v1/admins/current", nil)
	req.AddCookie(&http.Cookie{Name: cfg.CookieName, Value: token})

	rec, reached := serve(t, cfg, store, req)

	if reached {
		t.Fatal("handler was reached with a revoked token")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMiddlewarePwdChangeGate(t *testing.T) {
	cfg := testConfig()
	token, err := GenerateToken(uuid.New(), RoleAdmin, cfg.Secret, time.Now().Add(time.Hour), true)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	cases := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/v1/admins/current", http.StatusOK},
		{http.MethodPut, "/v1/admins/password", http.StatusOK},
		{http.MethodPost, "/v1/admins/logout", http.StatusOK},
		{http.MethodGet, "/v1/admins/list", http.StatusForbidden},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.AddCookie(&http.Cookie{Name: cfg.CookieName, Value: token})
		rec := httptest.NewRecorder()
		reached := false
		Middleware(cfg, nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			reached = true
		})).ServeHTTP(rec, req)
		if tc.want == http.StatusOK && !reached {
			t.Errorf("%s %s: handler not reached, status %d", tc.method, tc.path, rec.Code)
		}
		if tc.want == http.StatusForbidden {
			if reached {
				t.Errorf("%s %s: handler reached despite PwdChange", tc.method, tc.path)
			}
			if rec.Code != http.StatusForbidden {
				t.Errorf("%s %s: status = %d, want %d", tc.method, tc.path, rec.Code, http.StatusForbidden)
			}
		}
	}
}

func TestRoleMatrix(t *testing.T) {
	cases := []struct {
		access string
		viewer bool
		op     bool
		admin  bool
	}{
		{RoleViewer, true, false, false},
		{RoleOperator, true, true, false},
		{RoleAdmin, true, true, true},
		{"", false, false, false},
		{"unknown", false, false, false},
	}
	for _, tc := range cases {
		a := &Auth{Access: tc.access}
		if got := a.HasViewerAccess(); got != tc.viewer {
			t.Errorf("access %q: HasViewerAccess = %v, want %v", tc.access, got, tc.viewer)
		}
		if got := a.HasOperatorAccess(); got != tc.op {
			t.Errorf("access %q: HasOperatorAccess = %v, want %v", tc.access, got, tc.op)
		}
		if got := a.HasAdminAccess(); got != tc.admin {
			t.Errorf("access %q: HasAdminAccess = %v, want %v", tc.access, got, tc.admin)
		}
	}
}

// fakeTransport captures reply headers so cookie attributes can be asserted
// without spinning up a Kratos server.
type fakeTransport struct {
	header http.Header
}

func (f *fakeTransport) Kind() transport.Kind            { return transport.KindHTTP }
func (f *fakeTransport) Endpoint() string                { return "" }
func (f *fakeTransport) Operation() string               { return "" }
func (f *fakeTransport) RequestHeader() transport.Header { return fakeHeader{f.header} }
func (f *fakeTransport) ReplyHeader() transport.Header   { return fakeHeader{f.header} }

type fakeHeader struct{ http.Header }

func (f fakeHeader) Keys() []string {
	keys := make([]string, 0, len(f.Header))
	for k := range f.Header {
		keys = append(keys, k)
	}
	return keys
}

func TestSetCookieAttributes(t *testing.T) {
	cfg := testConfig()
	tr := &fakeTransport{header: make(http.Header)}
	ctx := transport.NewServerContext(context.Background(), tr)

	if err := SetCookie(ctx, cfg, uuid.New(), RoleAdmin, true, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("SetCookie: %v", err)
	}
	raw := tr.header.Get("Set-Cookie")
	cookie, err := http.ParseSetCookie(raw)
	if err != nil {
		t.Fatalf("ParseSetCookie(%q): %v", raw, err)
	}
	if cookie.Name != cfg.CookieName {
		t.Errorf("Name = %q, want %q", cookie.Name, cfg.CookieName)
	}
	if !cookie.HttpOnly {
		t.Error("HttpOnly must be set")
	}
	if !cookie.Secure {
		t.Error("Secure must follow config")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("SameSite = %v, want Strict", cookie.SameSite)
	}
	if cookie.Path != "/" {
		t.Errorf("Path = %q, want /", cookie.Path)
	}
	if cookie.Domain != cfg.CookieDomain {
		t.Errorf("Domain = %q, want %q", cookie.Domain, cfg.CookieDomain)
	}
	claims, err := ParseToken(cookie.Value, cfg.Secret)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if !claims.PwdChange {
		t.Error("PwdChange claim must survive the round trip")
	}
}

func TestDeleteCookieAttributes(t *testing.T) {
	cfg := testConfig()
	tr := &fakeTransport{header: make(http.Header)}
	ctx := transport.NewServerContext(context.Background(), tr)

	if err := DeleteCookie(ctx, cfg); err != nil {
		t.Fatalf("DeleteCookie: %v", err)
	}
	cookie, err := http.ParseSetCookie(tr.header.Get("Set-Cookie"))
	if err != nil {
		t.Fatalf("ParseSetCookie: %v", err)
	}
	if cookie.MaxAge >= 0 {
		t.Errorf("MaxAge = %d, want a negative value", cookie.MaxAge)
	}
	if !cookie.HttpOnly || cookie.Path != "/" {
		t.Errorf("cookie = %#v, want HttpOnly and Path=/", cookie)
	}
}

func TestCORSPreflight(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/v1/admins/list", nil)
	req.Header.Set("Origin", "http://localhost:8000")
	rec := httptest.NewRecorder()
	CORS([]string{"http://localhost:8000"})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("preflight must not reach the handler")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:8000" {
		t.Errorf("Allow-Origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Allow-Credentials = %q, want true", got)
	}
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Methods"), "PUT") {
		t.Errorf("Allow-Methods = %q, want PUT", rec.Header().Get("Access-Control-Allow-Methods"))
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/admins/login", nil)
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Accept", "application/protojson")
	rec := httptest.NewRecorder()
	CORS([]string{"http://localhost:8000"})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("unknown origin must not reach the handler")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %s", rec.Body.String())
	}
	if body.Reason != v1.ErrorReason_PERMISSION_DENIED.String() {
		t.Errorf("reason = %q, want PERMISSION_DENIED", body.Reason)
	}
}

func TestCORSAllowsSameOriginWithoutHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/admins/current", nil)
	rec := httptest.NewRecorder()
	reached := false
	CORS(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		reached = true
	})).ServeHTTP(rec, req)
	if !reached {
		t.Fatal("a request without Origin must pass through")
	}
}
