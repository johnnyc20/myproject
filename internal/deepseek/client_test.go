package deepseek

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testClient(t *testing.T, srv *httptest.Server, apiKey string) *Client {
	t.Helper()
	c := NewClient(Config{
		APIKey:       apiKey,
		BaseURL:      srv.URL,
		DefaultModel: "deepseek-chat",
		Timeout:      2 * time.Second,
		MaxBodyBytes: 1 << 20,
	})
	// Trust httptest's self-signed cert, same trick internal/secfetch uses.
	c.httpClient.Transport = srv.Client().Transport
	return c
}

func TestChat_Success(t *testing.T) {
	var gotAuth, gotBody string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"model": "deepseek-chat",
			"choices": []map[string]any{
				{
					"finish_reason": "stop",
					"message": map[string]any{
						"role":    "assistant",
						"content": "hello back",
					},
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     5,
				"completion_tokens": 2,
				"total_tokens":      7,
			},
		})
	}))
	defer srv.Close()

	c := testClient(t, srv, "test-key")
	result, err := c.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "hello back" {
		t.Errorf("Content = %q, want %q", result.Content, "hello back")
	}
	if result.TotalTokens != 7 {
		t.Errorf("TotalTokens = %d, want 7", result.TotalTokens)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer test-key")
	}
	if !strings.Contains(gotBody, `"model":"deepseek-chat"`) {
		t.Errorf("request body missing default model, got: %s", gotBody)
	}
}

func TestChat_RequiresAPIKey(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called when API key is empty")
	}))
	defer srv.Close()

	c := testClient(t, srv, "")
	if _, err := c.Chat(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "hi"}}}); err == nil {
		t.Fatal("expected error for missing API key, got nil")
	}
}

func TestChat_RequiresMessages(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called with no messages")
	}))
	defer srv.Close()

	c := testClient(t, srv, "test-key")
	if _, err := c.Chat(context.Background(), ChatRequest{}); err == nil {
		t.Fatal("expected error for empty messages, got nil")
	}
}

func TestChat_RejectsNonHTTPSBaseURL(t *testing.T) {
	c := NewClient(Config{
		APIKey:       "test-key",
		BaseURL:      "http://api.deepseek.com",
		DefaultModel: "deepseek-chat",
		Timeout:      2 * time.Second,
		MaxBodyBytes: 1 << 20,
	})
	_, err := c.Chat(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "hi"}}})
	if err == nil {
		t.Fatal("expected error for non-https base URL, got nil")
	}
}

func TestChat_SurfacesAPIError(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Authentication Fails",
				"type":    "authentication_error",
			},
		})
	}))
	defer srv.Close()

	c := testClient(t, srv, "bad-key")
	_, err := c.Chat(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "hi"}}})
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
	if !strings.Contains(err.Error(), "Authentication Fails") {
		t.Errorf("error = %q, want it to include the API's message", err.Error())
	}
}
