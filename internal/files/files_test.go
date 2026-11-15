package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveStaysInsideRoot(t *testing.T) {
	root := t.TempDir()
	m, err := New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	bad := []string{"../secret", "../../etc/passwd", "a/../../../etc/passwd", `..\..\windows\win.ini`}
	for _, rel := range bad {
		full, err := m.Resolve(rel)
		if err != nil {
			t.Fatalf("Resolve(%q) unexpected error: %v", rel, err)
		}
		if !within(root, full) {
			t.Errorf("Resolve(%q) escaped root: %s", rel, full)
		}
	}
}

func TestResolveDeniesMetadata(t *testing.T) {
	m, err := New(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{".braise", ".braise/events.jsonl", "sub/.braise/x"} {
		if _, err := m.Resolve(rel); err == nil {
			t.Errorf("Resolve(%q) should be denied", rel)
		}
	}
}

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"../evil.txt":    "evil.txt",
		`..\..\evil.txt`: "evil.txt",
		"a/b/c.txt":      "c.txt",
		"bad:name?.txt":  "bad_name_.txt",
		"CON":            "_CON",
		".braise":        "_braise",
		"  spaces.txt  ": "spaces.txt",
		"accents éà.txt": "accents éà.txt",
	}
	for in, want := range cases {
		if got := SanitizeName(in); got != want {
			t.Errorf("SanitizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSaveUploadUniqueAndList(t *testing.T) {
	root := t.TempDir()
	m, err := New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	first, err := m.SaveUpload("", "../hello.txt", strings.NewReader("one"))
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "hello.txt" || first.Path != "hello.txt" {
		t.Fatalf("unexpected entry: %+v", first)
	}
	second, err := m.SaveUpload("", "hello.txt", strings.NewReader("two"))
	if err != nil {
		t.Fatal(err)
	}
	if second.Name != "hello (1).txt" {
		t.Fatalf("expected collision suffix, got %q", second.Name)
	}
	entries, err := m.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestReadOnlyRefusesWrites(t *testing.T) {
	m, err := New(t.TempDir(), true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.SaveUpload("", "x.txt", strings.NewReader("x")); err != ErrReadOnly {
		t.Fatalf("expected ErrReadOnly, got %v", err)
	}
	if _, err := m.Mkdir("", "dir"); err != ErrReadOnly {
		t.Fatalf("expected ErrReadOnly, got %v", err)
	}
}

func TestDeleteMovesToTrash(t *testing.T) {
	root := t.TempDir()
	m, err := New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.SaveUpload("", "a.txt", strings.NewReader("a")); err != nil {
		t.Fatal(err)
	}
	if err := m.Delete("a.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "a.txt")); !os.IsNotExist(err) {
		t.Fatalf("file should be gone, err=%v", err)
	}
	trash, err := os.ReadDir(filepath.Join(root, ".braise", "trash"))
	if err != nil {
		t.Fatal(err)
	}
	if len(trash) != 1 {
		t.Fatalf("expected one trash entry, got %d", len(trash))
	}
}

func TestSymlinkEscapeDenied(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	m, err := New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := m.ResolveExisting("link/secret.txt"); err == nil {
		t.Fatal("symlink escape should be denied")
	}
}
