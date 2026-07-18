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

func TestCreateAndGetWidget(t *testing.T) {
	a := newTestAPI(t)

	createReq := httptest.NewRequest(http.MethodPost, "/widgets", strings.NewReader(`{"name":"gizmo","price":1999}`))
	createRec := httptest.NewRecorder()
	a.Routes().ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/widgets/1", nil)
	getRec := httptest.NewRecorder()
	a.Routes().ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getRec.Code, getRec.Body.String())
	}
	if !strings.Contains(getRec.Body.String(), "gizmo") {
		t.Fatalf("expected body to contain gizmo, got %s", getRec.Body.String())
	}
	if !strings.Contains(getRec.Body.String(), "1999") {
		t.Fatalf("expected body to contain price 1999, got %s", getRec.Body.String())
	}
}

func TestUpdateAndDeleteWidget(t *testing.T) {
	a := newTestAPI(t)

	createReq := httptest.NewRequest(http.MethodPost, "/widgets", strings.NewReader(`{"name":"gizmo","price":1999}`))
	createRec := httptest.NewRecorder()
	a.Routes().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/widgets/1", strings.NewReader(`{"name":"gadget","price":2999}`))
	updateRec := httptest.NewRecorder()
	a.Routes().ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", updateRec.Code, updateRec.Body.String())
	}
	if !strings.Contains(updateRec.Body.String(), "gadget") {
		t.Fatalf("expected body to contain gadget, got %s", updateRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/widgets/1", nil)
	deleteRec := httptest.NewRecorder()
	a.Routes().ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/widgets/1", nil)
	getRec := httptest.NewRecorder()
	a.Routes().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", getRec.Code, getRec.Body.String())
	}
}

func createMemory(t *testing.T, a *API, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/memories", strings.NewReader(body))
	rec := httptest.NewRecorder()
	a.Routes().ServeHTTP(rec, req)
	return rec
}

func TestCreateAndGetMemory(t *testing.T) {
	a := newTestAPI(t)

	createRec := createMemory(t, a, `{"name":"testing-style","type":"feedback","description":"prefers table-driven tests","content":"the user prefers table-driven Go tests over repeated near-identical test funcs"}`)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/memories/1", nil)
	getRec := httptest.NewRecorder()
	a.Routes().ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getRec.Code, getRec.Body.String())
	}
	if !strings.Contains(getRec.Body.String(), "testing-style") {
		t.Fatalf("expected body to contain testing-style, got %s", getRec.Body.String())
	}
}

func TestListMemoriesFilteredByType(t *testing.T) {
	a := newTestAPI(t)

	createMemory(t, a, `{"name":"a","type":"feedback","description":"d","content":"c"}`)
	createMemory(t, a, `{"name":"b","type":"project","description":"d","content":"c"}`)

	listReq := httptest.NewRequest(http.MethodGet, "/memories?type=feedback", nil)
	listRec := httptest.NewRecorder()
	a.Routes().ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), `"a"`) {
		t.Fatalf("expected list to contain memory a, got %s", listRec.Body.String())
	}
	if strings.Contains(listRec.Body.String(), `"b"`) {
		t.Fatalf("expected list to exclude memory b, got %s", listRec.Body.String())
	}
}

func TestCreateMemoryInvalidType(t *testing.T) {
	a := newTestAPI(t)

	rec := createMemory(t, a, `{"name":"a","type":"bogus","description":"d","content":"c"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteMemory(t *testing.T) {
	a := newTestAPI(t)

	createRec := createMemory(t, a, `{"name":"a","type":"reference","description":"d","content":"c"}`)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/memories/1", nil)
	deleteRec := httptest.NewRecorder()
	a.Routes().ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/memories/1", nil)
	getRec := httptest.NewRecorder()
	a.Routes().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", getRec.Code, getRec.Body.String())
	}
}

func TestSearchMemories(t *testing.T) {
	a := newTestAPI(t)

	createMemory(t, a, `{"name":"go-tests","type":"feedback","description":"table-driven Go tests","content":"the user prefers table-driven Go tests"}`)
	createMemory(t, a, `{"name":"deploy-freeze","type":"project","description":"merge freeze","content":"mobile team is cutting a release branch"}`)

	searchReq := httptest.NewRequest(http.MethodGet, "/memories/search?q=table-driven", nil)
	searchRec := httptest.NewRecorder()
	a.Routes().ServeHTTP(searchRec, searchReq)

	if searchRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", searchRec.Code, searchRec.Body.String())
	}
	if !strings.Contains(searchRec.Body.String(), "go-tests") {
		t.Fatalf("expected search results to contain go-tests, got %s", searchRec.Body.String())
	}
	if strings.Contains(searchRec.Body.String(), "deploy-freeze") {
		t.Fatalf("expected search results to exclude deploy-freeze, got %s", searchRec.Body.String())
	}
}
