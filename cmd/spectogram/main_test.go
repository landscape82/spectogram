package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerListsAndServesGeneratedData(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(`{"ok":true}`), 0644); err != nil {
		t.Fatal(err)
	}
	server := newServer(t.TempDir(), dir)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/spectrograms", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "/data/one.json") {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
	}
}

func TestRunRejectsUnsupportedInput(t *testing.T) {
	if err := run(filepath.Join(t.TempDir(), "audio.txt"), "out.png", "out.json"); err == nil {
		t.Fatal("expected unsupported input error")
	}
}
