package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/johnnyc20/myproject/internal/store"
)

type API struct {
	store *store.Store
}

func New(s *store.Store) *API {
	return &API{store: s}
}

func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.handleHealth)
	mux.HandleFunc("GET /items", a.handleListItems)
	mux.HandleFunc("POST /items", a.handleCreateItem)
	mux.HandleFunc("GET /items/{id}", a.handleGetItem)
	mux.HandleFunc("DELETE /items/{id}", a.handleDeleteItem)
	mux.HandleFunc("GET /widgets", a.handleListWidgets)
	mux.HandleFunc("POST /widgets", a.handleCreateWidget)
	mux.HandleFunc("GET /widgets/{id}", a.handleGetWidget)
	mux.HandleFunc("PUT /widgets/{id}", a.handleUpdateWidget)
	mux.HandleFunc("DELETE /widgets/{id}", a.handleDeleteWidget)
	mux.HandleFunc("GET /notes", a.handleListNotes)
	mux.HandleFunc("POST /notes", a.handleCreateNote)
	mux.HandleFunc("GET /notes/{id}", a.handleGetNote)
	mux.HandleFunc("GET /memories/search", a.handleSearchMemories)
	mux.HandleFunc("GET /memories", a.handleListMemories)
	mux.HandleFunc("POST /memories", a.handleCreateMemory)
	mux.HandleFunc("GET /memories/{id}", a.handleGetMemory)
	mux.HandleFunc("DELETE /memories/{id}", a.handleDeleteMemory)
	return mux
}

var validMemoryTypes = map[string]bool{
	"user":      true,
	"feedback":  true,
	"project":   true,
	"reference": true,
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) handleListItems(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListItems()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	item, err := a.store.CreateItem(body.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) handleGetItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	item, err := a.store.GetItem(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if err := a.store.DeleteItem(id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleListWidgets(w http.ResponseWriter, r *http.Request) {
	widgets, err := a.store.ListWidgets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, widgets)
}

func (a *API) handleCreateWidget(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name  string `json:"name"`
		Price int64  `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	if body.Price == 0 {
		writeError(w, http.StatusBadRequest, errors.New("price is required"))
		return
	}
	widget, err := a.store.CreateWidget(body.Name, body.Price)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, widget)
}

func (a *API) handleGetWidget(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	widget, err := a.store.GetWidget(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, widget)
}

func (a *API) handleUpdateWidget(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var body struct {
		Name  string `json:"name"`
		Price int64  `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	if body.Price == 0 {
		writeError(w, http.StatusBadRequest, errors.New("price is required"))
		return
	}
	widget, err := a.store.UpdateWidget(id, body.Name, body.Price)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, widget)
}

func (a *API) handleDeleteWidget(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if err := a.store.DeleteWidget(id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleListNotes(w http.ResponseWriter, r *http.Request) {
	notes, err := a.store.ListNotes()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, notes)
}

func (a *API) handleCreateNote(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	note, err := a.store.CreateNote(body.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, note)
}

func (a *API) handleGetNote(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	note, err := a.store.GetNote(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, note)
}

func (a *API) handleListMemories(w http.ResponseWriter, r *http.Request) {
	memType := r.URL.Query().Get("type")
	if memType != "" && !validMemoryTypes[memType] {
		writeError(w, http.StatusBadRequest, errors.New("invalid type"))
		return
	}
	memories, err := a.store.ListMemories(memType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, memories)
}

func (a *API) handleCreateMemory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Description string `json:"description"`
		Content     string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	if !validMemoryTypes[body.Type] {
		writeError(w, http.StatusBadRequest, errors.New("invalid type"))
		return
	}
	if body.Description == "" {
		writeError(w, http.StatusBadRequest, errors.New("description is required"))
		return
	}
	if body.Content == "" {
		writeError(w, http.StatusBadRequest, errors.New("content is required"))
		return
	}
	memory, err := a.store.CreateMemory(body.Name, body.Type, body.Description, body.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, memory)
}

func (a *API) handleGetMemory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	memory, err := a.store.GetMemory(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, memory)
}

func (a *API) handleDeleteMemory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if err := a.store.DeleteMemory(id); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleSearchMemories(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, errors.New("q is required"))
		return
	}
	memories, err := a.store.SearchMemories(q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, memories)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
