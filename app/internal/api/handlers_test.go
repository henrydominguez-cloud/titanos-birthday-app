package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/titanos/birthday-app/internal/store"
)

// newTestServer returns a Server whose clock is frozen at 2026-09-06.
func newTestServer() *Server {
	s := NewServer(store.NewMemory(), nil)
	s.now = func() time.Time { return time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC) }
	return s
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestPutThenGet(t *testing.T) {
	h := newTestServer().Routes()

	// Save a user whose birthday is 4 days away.
	rec := do(t, h, http.MethodPut, "/hello/jdoe", `{"dateOfBirth":"1990-09-10"}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PUT status = %d, want 204 (body: %s)", rec.Code, rec.Body)
	}

	rec = do(t, h, http.MethodGet, "/hello/jdoe", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", rec.Code)
	}
	var got messageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := "Hello, jdoe! Your birthday is in 4 day(s)"
	if got.Message != want {
		t.Errorf("message = %q, want %q", got.Message, want)
	}
}

func TestPutValidation(t *testing.T) {
	h := newTestServer().Routes()
	cases := []struct {
		name, path, body string
		want             int
	}{
		{"non-letter username", "/hello/jdoe123", `{"dateOfBirth":"1990-09-10"}`, http.StatusBadRequest},
		{"future date", "/hello/jdoe", `{"dateOfBirth":"3000-01-01"}`, http.StatusBadRequest},
		{"today is not before today", "/hello/jdoe", `{"dateOfBirth":"2026-09-06"}`, http.StatusBadRequest},
		{"bad date format", "/hello/jdoe", `{"dateOfBirth":"06-09-1990"}`, http.StatusBadRequest},
		{"invalid json", "/hello/jdoe", `not-json`, http.StatusBadRequest},
		{"unknown field", "/hello/jdoe", `{"foo":"bar"}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if rec := do(t, h, http.MethodPut, tc.path, tc.body); rec.Code != tc.want {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, tc.want, rec.Body)
			}
		})
	}
}

func TestPutAcceptsSpecTypoField(t *testing.T) {
	h := newTestServer().Routes()
	// The take-home spells the field "dateOfBrith"; it must still work.
	rec := do(t, h, http.MethodPut, "/hello/typo", `{"dateOfBrith":"1990-01-01"}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", rec.Code, rec.Body)
	}
}

func TestGetUnknownUser(t *testing.T) {
	h := newTestServer().Routes()
	if rec := do(t, h, http.MethodGet, "/hello/ghost", ""); rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestGetHappyBirthday(t *testing.T) {
	h := newTestServer().Routes()
	do(t, h, http.MethodPut, "/hello/today", `{"dateOfBirth":"1990-09-06"}`)
	rec := do(t, h, http.MethodGet, "/hello/today", "")
	var got messageResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Message != "Hello, today! Happy birthday!" {
		t.Errorf("message = %q", got.Message)
	}
}

func TestHealthAndReady(t *testing.T) {
	h := newTestServer().Routes()
	for _, p := range []string{"/healthz", "/readyz"} {
		if rec := do(t, h, http.MethodGet, p, ""); rec.Code != http.StatusOK {
			t.Errorf("%s status = %d, want 200", p, rec.Code)
		}
	}
}
