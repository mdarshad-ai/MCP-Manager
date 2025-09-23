package httpapi

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"mcp/manager/internal/registry"
)

func TestServersGET(t *testing.T) {
	reg := &registry.Registry{Version: "1.0", Servers: []registry.Server{{Name: "fs", Slug: "filesystem"}}}
	s := NewServer(reg)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/servers", nil)
	s.Router().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status %d", rr.Code)
	}
	var got []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 server, got %d", len(got))
	}

	name, ok := got[0]["Name"].(string)
	if !ok {
		t.Fatalf("name is not a string: %v", got[0]["Name"])
	}

	if name != "fs" {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}
