package dav

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"braise/internal/files"
)

func TestPropfindHidesMetadata(t *testing.T) {
	root := t.TempDir()
	m, err := files.New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".braise"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := Handler(m, "/dav", nil)
	req := httptest.NewRequest("PROPFIND", "/dav/", nil)
	req.Header.Set("Depth", "1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 207 {
		t.Fatalf("status = %d, want 207; body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if strings.Contains(body, ".braise") {
		t.Fatalf("WebDAV listing leaks .braise: %s", body)
	}
	if !strings.Contains(body, "hello.txt") {
		t.Fatalf("WebDAV listing misses hello.txt: %s", body)
	}
	if strings.Contains(body, "Internal Server Error") {
		t.Fatalf("unexpected trailing error: %s", body)
	}
}

func TestMetadataPathIsHidden(t *testing.T) {
	m, err := files.New(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	h := Handler(m, "/dav", nil)
	req := httptest.NewRequest("PROPFIND", "/dav/.braise", nil)
	req.Header.Set("Depth", "0")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 404 {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestReadOnlyRefusesPut(t *testing.T) {
	m, err := files.New(t.TempDir(), true)
	if err != nil {
		t.Fatal(err)
	}
	h := Handler(m, "/dav", nil)
	req := httptest.NewRequest("PUT", "/dav/x.txt", strings.NewReader("x"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code < 400 {
		t.Fatalf("read-only PUT status = %d, want an error", rr.Code)
	}
}
