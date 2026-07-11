package panel

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The panel PUT /api/users/{name} is a full replace: it 422s without a username
// in the body, and it takes the service list verbatim.
func TestUpdateUserSendsUsername(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/users/alice" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		io.WriteString(w, `{"username":"alice","service_ids":[2]}`)
	}))
	defer srv.Close()

	if _, err := New(srv.URL).UpdateUser(context.Background(), "alice", "never", []int{2}); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if got["username"] != "alice" {
		t.Errorf("body username = %v, want alice", got["username"])
	}
	if got["expire_strategy"] != "never" {
		t.Errorf("body expire_strategy = %v, want never", got["expire_strategy"])
	}
}

func TestUpdateUserClearsWithEmptyList(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		io.WriteString(w, `{"username":"alice"}`)
	}))
	defer srv.Close()

	// nil must serialise as [] so removing the last location clears services.
	if _, err := New(srv.URL).UpdateUser(context.Background(), "alice", "never", nil); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	ids, ok := got["service_ids"]
	if !ok {
		t.Fatal("service_ids omitted; an empty list must be sent to clear services")
	}
	if arr, ok := ids.([]any); !ok || len(arr) != 0 {
		t.Errorf("service_ids = %v, want []", ids)
	}
}
