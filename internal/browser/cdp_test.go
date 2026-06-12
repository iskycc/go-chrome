package browser

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSelectPageTarget(t *testing.T) {
	targets := []devtoolsTarget{
		{ID: "browser", Type: "browser"},
		{ID: "worker", Type: "service_worker"},
		{ID: "page-1", Type: "page", URL: "about:blank"},
	}
	id, ok := selectPageTarget(targets)
	if !ok {
		t.Fatal("expected page target")
	}
	if id != "page-1" {
		t.Fatalf("unexpected target id: %s", id)
	}
}

func TestEnsurePageTargetUsesExistingPage(t *testing.T) {
	var newCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json/list":
			_, _ = w.Write([]byte(`[{"id":"page-1","type":"page","url":"about:blank"}]`))
		case "/json/new":
			newCalled = true
			http.Error(w, "should not create", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	id, err := ensurePageTarget(server.Client(), server.URL)
	if err != nil {
		t.Fatalf("ensure page target: %v", err)
	}
	if id != "page-1" {
		t.Fatalf("unexpected target id: %s", id)
	}
	if newCalled {
		t.Fatal("did not expect /json/new to be called")
	}
}

func TestEnsurePageTargetCreatesPageWhenMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json/list":
			_, _ = w.Write([]byte(`[{"id":"browser","type":"browser"}]`))
		case "/json/new":
			if r.Method != http.MethodPut {
				t.Errorf("expected PUT /json/new, got %s", r.Method)
			}
			if r.URL.RawQuery != "about:blank" {
				t.Errorf("expected about:blank query, got %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"id":"page-2","type":"page","url":"about:blank"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	id, err := ensurePageTarget(server.Client(), server.URL)
	if err != nil {
		t.Fatalf("ensure page target: %v", err)
	}
	if id != "page-2" {
		t.Fatalf("unexpected target id: %s", id)
	}
}

func TestCreatePageTargetRequiresID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"type":"page"}`))
	}))
	defer server.Close()

	if _, err := createPageTarget(server.Client(), server.URL); err == nil {
		t.Fatal("expected empty target id error")
	}
}
