package api

import (
	"encoding/json"
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

func createItem(t *testing.T, a *API, name string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{"name":"`+name+`"}`))
	rec := httptest.NewRecorder()
	a.Routes().ServeHTTP(rec, req)
	return rec
}

func TestListItemsPagination(t *testing.T) {
	a := newTestAPI(t)

	for _, name := range []string{"a", "b", "c"} {
		rec := createItem(t, a, name)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	}

	listReq := httptest.NewRequest(http.MethodGet, "/items?limit=2&offset=1", nil)
	listRec := httptest.NewRecorder()
	a.Routes().ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}

	var page struct {
		Items  []store.Item `json:"items"`
		Total  int          `json:"total"`
		Limit  int          `json:"limit"`
		Offset int          `json:"offset"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if page.Total != 3 {
		t.Fatalf("expected total 3, got %d", page.Total)
	}
	if page.Limit != 2 {
		t.Fatalf("expected limit 2, got %d", page.Limit)
	}
	if page.Offset != 1 {
		t.Fatalf("expected offset 1, got %d", page.Offset)
	}
	if len(page.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(page.Items))
	}
	if page.Items[0].Name != "b" || page.Items[1].Name != "c" {
		t.Fatalf("expected items b, c, got %s, %s", page.Items[0].Name, page.Items[1].Name)
	}
}

func TestListItemsInvalidPaginationParams(t *testing.T) {
	a := newTestAPI(t)

	createItem(t, a, "a")

	cases := []struct {
		name  string
		query string
	}{
		{"non-numeric limit", "limit=abc"},
		{"negative offset", "offset=-1"},
		{"non-numeric offset", "offset=abc"},
		{"limit too large", "limit=500"},
		{"limit zero", "limit=0"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/items?"+tc.query, nil)
			rec := httptest.NewRecorder()
			a.Routes().ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestListItemsPagination_OffsetPastEnd(t *testing.T) {
	a := newTestAPI(t)

	for _, name := range []string{"a", "b", "c"} {
		rec := createItem(t, a, name)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	}

	listReq := httptest.NewRequest(http.MethodGet, "/items?limit=10&offset=50", nil)
	listRec := httptest.NewRecorder()
	a.Routes().ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}

	var page struct {
		Items  []store.Item `json:"items"`
		Total  int          `json:"total"`
		Limit  int          `json:"limit"`
		Offset int          `json:"offset"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if page.Total != 3 {
		t.Fatalf("expected total 3, got %d", page.Total)
	}
	if page.Offset != 50 {
		t.Fatalf("expected offset 50, got %d", page.Offset)
	}
	if len(page.Items) != 0 {
		t.Fatalf("expected 0 items for offset past end, got %d", len(page.Items))
	}
}

func TestListItemsPagination_DefaultsWhenOmitted(t *testing.T) {
	a := newTestAPI(t)

	createItem(t, a, "a")

	listReq := httptest.NewRequest(http.MethodGet, "/items", nil)
	listRec := httptest.NewRecorder()
	a.Routes().ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}

	var page struct {
		Items  []store.Item `json:"items"`
		Total  int          `json:"total"`
		Limit  int          `json:"limit"`
		Offset int          `json:"offset"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if page.Limit != 20 {
		t.Fatalf("expected default limit 20, got %d", page.Limit)
	}
	if page.Offset != 0 {
		t.Fatalf("expected default offset 0, got %d", page.Offset)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(page.Items))
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
