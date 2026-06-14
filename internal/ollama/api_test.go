package ollama

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPITags(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"models":[{"name":"gemma4:e4b","size":9600000000},{"name":"embeddinggemma","size":300000000}]}`))
	}))
	defer srv.Close()
	a := &api{host: hostFromURL(srv.URL), httpc: srv.Client()}
	models, err := a.tags(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].Name != "gemma4:e4b" || models[0].Size != 9600000000 {
		t.Fatalf("tags = %+v", models)
	}
}

func TestAPIPullStreamsProgress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"pulling","completed":10,"total":100}` + "\n"))
		w.Write([]byte(`{"status":"pulling","completed":100,"total":100}` + "\n"))
		w.Write([]byte(`{"status":"success"}` + "\n"))
	}))
	defer srv.Close()
	a := &api{host: hostFromURL(srv.URL), httpc: srv.Client()}
	var last PullProgress
	err := a.pull(context.Background(), "gemma4:e4b", func(p PullProgress) { last = p })
	if err != nil {
		t.Fatal(err)
	}
	if last.Status != "success" {
		t.Fatalf("last status = %q", last.Status)
	}
}

func TestAPIUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"models":[]}`))
	}))
	defer srv.Close()
	a := &api{host: hostFromURL(srv.URL), httpc: srv.Client()}
	if !a.up(context.Background()) {
		t.Fatal("up() = false, want true")
	}
}

func hostFromURL(u string) string {
	return u[len("http://"):]
}
