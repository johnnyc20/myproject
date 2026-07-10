package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/johnnyc20/myproject/internal/store"
)

func newTestAPI(t *testing.T) *API {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return New(s)
}

func TestHealthz(t *testing.T) {
	a := newTestAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	a.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCreateAndGetItem(t *testing.T) {
	a := newTestAPI(t)

	createReq := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{"name":"widget"}`))
	createRec := httptest.NewRecorder()
	a.Routes().ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/items/1", nil)
	getRec := httptest.NewRecorder()
	a.Routes().ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getRec.Code, getRec.Body.String())
	}
	if !strings.Contains(getRec.Body.String(), "widget") {
		t.Fatalf("expected body to contain widget, got %s", getRec.Body.String())
	}
}
