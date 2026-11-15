// Package files owns the shared folder: safe path resolution, listing,
// uploads, folders and a soft-delete trash.
package files

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

var (
	ErrDenied   = errors.New("chemin interdit")
	ErrReadOnly = errors.New("espace en lecture seule")
)

// Entry is one file or folder exposed by the API.
type Entry struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Dir     bool      `json:"dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	MIME    string    `json:"mime,omitempty"`
}

// Manager confines all operations to a single root directory.
type Manager struct {
	root     string
	readOnly bool
}

// New validates and creates the shared root.
func New(root string, readOnly bool) (*Manager, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(real)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%s n'est pas un dossier", real)
	}
	return &Manager{root: real, readOnly: readOnly}, nil
}

// Root returns the absolute shared directory.
func (m *Manager) Root() string { return m.root }

// ReadOnly reports whether mutations are refused.
func (m *Manager) ReadOnly() bool { return m.readOnly }

// Resolve maps a slash-separated relative path to an absolute path, refusing
// the metadata directory. It does not require the target to exist.
func (m *Manager) Resolve(rel string) (string, error) {
	rel = strings.ReplaceAll(rel, "\\", "/")
	if strings.ContainsRune(rel, 0) {
		return "", ErrDenied
	}
	clean := path.Clean("/" + rel)
	trimmed := strings.TrimPrefix(clean, "/")
	if trimmed == "" {
		return m.root, nil
	}
	for _, part := range strings.Split(trimmed, "/") {
		if part == ".braise" {
			return "", ErrDenied
		}
	}
	full := filepath.Join(m.root, filepath.FromSlash(trimmed))
	if !within(m.root, full) {
		return "", ErrDenied
	}
	return full, nil
}

// ResolveExisting resolves a path and rejects symlinks and escapes.
func (m *Manager) ResolveExisting(rel string) (string, os.FileInfo, error) {
	full, err := m.Resolve(rel)
	if err != nil {
		return "", nil, err
	}
	fi, err := os.Lstat(full)
	if err != nil {
		return "", nil, err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return "", nil, ErrDenied
	}
	real, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", nil, err
	}
	if !within(m.root, real) {
		return "", nil, ErrDenied
	}
	return full, fi, nil
}

// List returns the entries of a directory, folders first.
func (m *Manager) List(rel string) ([]Entry, error) {
	full, fi, err := m.ResolveExisting(rel)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("pas un dossier")
	}
	des, err := os.ReadDir(full)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(des))
	for _, de := range des {
		name := de.Name()
		if name == ".braise" {
			continue
		}
		info, err := de.Info()
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		relPath := path.Join(strings.Trim(rel, "/"), name)
		e := Entry{
			Name:    name,
			Path:    relPath,
			Dir:     info.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}
		if !e.Dir {
			e.MIME = MIMEFor(name)
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dir != out[j].Dir {
			return out[i].Dir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

// Stat returns a single entry.
func (m *Manager) Stat(rel string) (Entry, error) {
	full, fi, err := m.ResolveExisting(rel)
	if err != nil {
		return Entry{}, err
	}
	_ = full
	e := Entry{
		Name:    filepath.Base(full),
		Path:    strings.Trim(path.Clean("/"+rel), "/"),
		Dir:     fi.IsDir(),
		Size:    fi.Size(),
		ModTime: fi.ModTime(),
	}
	if !e.Dir {
		e.MIME = MIMEFor(e.Name)
	}
	return e, nil
}

// Open opens an existing regular file for reading.
func (m *Manager) Open(rel string) (*os.File, os.FileInfo, error) {
	full, fi, err := m.ResolveExisting(rel)
	if err != nil {
		return nil, nil, err
	}
	if fi.IsDir() {
		return nil, nil, fmt.Errorf("pas un fichier")
	}
	f, err := os.Open(full)
	if err != nil {
		return nil, nil, err
	}
	return f, fi, nil
}

// SaveUpload streams r into relDir under a sanitized, non-colliding name.
func (m *Manager) SaveUpload(relDir, filename string, r io.Reader) (Entry, error) {
	if m.readOnly {
		return Entry{}, ErrReadOnly
	}
	dir, fi, err := m.ResolveExisting(relDir)
	if err != nil {
		return Entry{}, err
	}
	if !fi.IsDir() {
		return Entry{}, fmt.Errorf("destination invalide")
	}
	name := SanitizeName(filename)
	if name == "" {
		name = "fichier"
	}
	dest, err := uniquePath(dir, name)
	if err != nil {
		return Entry{}, err
	}

	tmpDir := filepath.Join(m.root, ".braise", "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return Entry{}, err
	}
	tmp, err := os.CreateTemp(tmpDir, "upload-*")
	if err != nil {
		return Entry{}, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	n, err := io.Copy(tmp, r)
	if err != nil {
		tmp.Close()
		return Entry{}, err
	}
	if err := tmp.Close(); err != nil {
		return Entry{}, err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return Entry{}, err
	}
	if err := os.Rename(tmpName, dest); err != nil {
		return Entry{}, err
	}
	info, err := os.Stat(dest)
	if err != nil {
		return Entry{}, err
	}
	return Entry{
		Name:    filepath.Base(dest),
		Path:    path.Join(strings.Trim(relDir, "/"), filepath.Base(dest)),
		Size:    n,
		ModTime: info.ModTime(),
		MIME:    MIMEFor(dest),
	}, nil
}

// Mkdir creates a subdirectory.
func (m *Manager) Mkdir(relDir, name string) (Entry, error) {
	if m.readOnly {
		return Entry{}, ErrReadOnly
	}
	name = SanitizeName(name)
	if name == "" || name == "." || name == ".." {
		return Entry{}, fmt.Errorf("nom invalide")
	}
	dir, fi, err := m.ResolveExisting(relDir)
	if err != nil {
		return Entry{}, err
	}
	if !fi.IsDir() {
		return Entry{}, fmt.Errorf("destination invalide")
	}
	full := filepath.Join(dir, name)
	if !within(m.root, full) {
		return Entry{}, ErrDenied
	}
	if err := os.Mkdir(full, 0o755); err != nil {
		return Entry{}, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return Entry{}, err
	}
	return Entry{
		Name:    name,
		Path:    path.Join(strings.Trim(relDir, "/"), name),
		Dir:     true,
		ModTime: info.ModTime(),
	}, nil
}

// Delete moves a file or folder to .braise/trash. The root cannot be deleted.
func (m *Manager) Delete(rel string) error {
	if m.readOnly {
		return ErrReadOnly
	}
	full, fi, err := m.ResolveExisting(rel)
	if err != nil {
		return err
	}
	if full == m.root {
		return fmt.Errorf("impossible de supprimer la racine")
	}
	_ = fi
	trash := filepath.Join(m.root, ".braise", "trash")
	if err := os.MkdirAll(trash, 0o755); err != nil {
		return err
	}
	base := time.Now().UTC().Format("20060102-150405.000") + "-" + filepath.Base(full)
	dest := filepath.Join(trash, base)
	if _, err := os.Lstat(dest); err == nil {
		dest = fmt.Sprintf("%s-%d", dest, time.Now().UnixNano())
	}
	if err := os.Rename(full, dest); err != nil {
		// Fall back to a hard delete only if the move itself is impossible.
		return os.RemoveAll(full)
	}
	return nil
}

func within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func uniquePath(dir, name string) (string, error) {
	full := filepath.Join(dir, name)
	if _, err := os.Lstat(full); errors.Is(err, os.ErrNotExist) {
		return full, nil
	} else if err != nil {
		return "", err
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 1; i < 10000; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", stem, i, ext))
		if _, err := os.Lstat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("trop de collisions pour %s", name)
}

var windowsReserved = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true,
	"COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
	"LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// SanitizeName strips path separators, control characters and Windows-hostile
// characters while preserving unicode filenames.
func SanitizeName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(name)
	if name == "." || name == "/" {
		return ""
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r == unicode.ReplacementChar:
			b.WriteRune('_')
		case unicode.IsControl(r):
			continue
		case strings.ContainsRune(`<>:"/\|?*`, r):
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	name = strings.TrimSpace(b.String())
	name = strings.TrimRight(name, ". ")
	if name == "" {
		return ""
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	if windowsReserved[strings.ToUpper(stem)] {
		name = "_" + name
	}
	// Keep names comfortably below common filesystem limits.
	runes := []rune(name)
	if len(runes) > 180 {
		name = string(runes[:180])
	}
	if name == ".braise" {
		name = "_braise"
	}
	return name
}

var extraMIME = map[string]string{
	".md":   "text/markdown; charset=utf-8",
	".go":   "text/plain; charset=utf-8",
	".rs":   "text/plain; charset=utf-8",
	".ts":   "text/plain; charset=utf-8",
	".tsx":  "text/plain; charset=utf-8",
	".jsx":  "text/plain; charset=utf-8",
	".toml": "text/plain; charset=utf-8",
	".yaml": "text/yaml; charset=utf-8",
	".yml":  "text/yaml; charset=utf-8",
	".json": "application/json",
	".webp": "image/webp",
	".heic": "image/heic",
	".mkv":  "video/x-matroska",
	".mov":  "video/quicktime",
	".m4a":  "audio/mp4",
	".flac": "audio/flac",
	".opus": "audio/ogg",
	".webm": "video/webm",
	".wasm": "application/wasm",
}

// MIMEFor returns the best-effort MIME type of a filename.
func MIMEFor(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if v, ok := extraMIME[ext]; ok {
		return v
	}
	if v := mime.TypeByExtension(ext); v != "" {
		return v
	}
	return "application/octet-stream"
}

// Previewable reports whether the browser can display the file inline.
func Previewable(name, mimeType string) bool {
	if strings.HasPrefix(mimeType, "image/") ||
		strings.HasPrefix(mimeType, "video/") ||
		strings.HasPrefix(mimeType, "audio/") ||
		strings.HasPrefix(mimeType, "text/") {
		return true
	}
	switch mimeType {
	case "application/pdf", "application/json", "application/xml", "application/javascript":
		return true
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".txt", ".md", ".log", ".csv", ".go", ".js", ".css", ".html", ".json", ".yaml", ".yml", ".toml", ".sh", ".py", ".rs", ".c", ".h", ".java", ".sql":
		return true
	}
	return false
}
