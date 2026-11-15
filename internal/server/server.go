// Package server wires the HTTP API, the embedded web UI, SSE and WebDAV.
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"

	"braise/internal/dav"
	"braise/internal/files"
	"braise/internal/sse"
	"braise/internal/store"
)

const (
	maxUploadBytes = 20 << 30 // 20 GiB per request
	maxJSONBytes   = 1 << 20
	maxTextRunes   = 4000
	maxAuthorRunes = 48
)

// Config is everything the HTTP server needs.
type Config struct {
	Name      string
	Version   string
	Root      string
	Port      int
	ReadOnly  bool
	URLs      []string
	Web       http.FileSystem
	Store     *store.Store
	Files     *files.Manager
	Hub       *sse.Hub
	Logger    *slog.Logger
	StartedAt time.Time
}

// Server holds handlers.
type Server struct {
	cfg Config
	mux *http.ServeMux
}

// New builds the root HTTP handler.
func New(cfg Config) http.Handler {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	mime.AddExtensionType(".webmanifest", "application/manifest+json")
	s := &Server{cfg: cfg}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/info", s.info)
	mux.HandleFunc("GET /api/events", s.events)
	mux.HandleFunc("GET /api/messages", s.messages)
	mux.HandleFunc("POST /api/messages", s.postMessage)
	mux.HandleFunc("GET /api/files", s.listFiles)
	mux.HandleFunc("POST /api/upload", s.upload)
	mux.HandleFunc("GET /api/download", s.download)
	mux.HandleFunc("GET /api/raw", s.raw)
	mux.HandleFunc("POST /api/mkdir", s.mkdir)
	mux.HandleFunc("POST /api/delete", s.delete)
	mux.HandleFunc("GET /api/qr", s.qr)
	mux.HandleFunc("POST /api/presence", s.presence)
	mux.HandleFunc("POST /api/presence/leave", s.leave)
	mux.HandleFunc("POST /api/typing", s.typing)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok\n")
	})

	davHandler := dav.Handler(cfg.Files, "/dav", func(r *http.Request, err error) {
		if err != nil {
			cfg.Logger.Debug("webdav", "method", r.Method, "path", r.URL.Path, "err", err)
		}
	})
	mux.Handle("/dav/", davHandler)
	mux.HandleFunc("/dav", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/dav/"
		davHandler.ServeHTTP(w, r)
	})
	mux.Handle("/", http.FileServer(cfg.Web))

	s.mux = mux
	return s.recoverer(s.logger(s.headers(s.originCheck(mux))))
}

// ---- middleware ----

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				s.cfg.Logger.Error("panic", "err", v, "path", r.URL.Path)
				http.Error(w, "erreur interne", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status != 0 {
		return
	}
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += int64(n)
	return n, err
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (s *Server) logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/events" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}
		s.cfg.Logger.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"bytes", sw.bytes,
			"dur", time.Since(start).Round(time.Millisecond).String(),
		)
	})
}

func (s *Server) headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob:; media-src 'self' blob:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'self'")
		if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, ".html") {
			h.Set("Cache-Control", "no-cache")
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) originCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || !strings.EqualFold(u.Host, r.Host) {
					s.fail(w, http.StatusForbidden, "origine refusée")
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) fail(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "Content-Type application/json requis"})
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBytes)
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON invalide"})
		return false
	}
	return true
}

func normRel(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	return strings.Trim(path.Clean("/"+p), "/")
}

func parentOf(p string) string {
	d := path.Dir("/" + p)
	return strings.Trim(d, "/")
}

func (s *Server) fileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, files.ErrDenied), errors.Is(err, files.ErrReadOnly), errors.Is(err, os.ErrPermission):
		s.fail(w, http.StatusForbidden, "accès refusé")
	case errors.Is(err, os.ErrNotExist):
		s.fail(w, http.StatusNotFound, "introuvable")
	case errors.Is(err, os.ErrExist):
		s.fail(w, http.StatusConflict, "existe déjà")
	default:
		s.cfg.Logger.Error("fichier", "err", err)
		s.fail(w, http.StatusInternalServerError, "erreur fichier")
	}
}

var colorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var idRe = regexp.MustCompile(`^[A-Za-z0-9_-]{6,64}$`)

func cleanAuthor(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			b.WriteRune(' ')
			continue
		}
		if r < 32 || r == 127 {
			continue
		}
		b.WriteRune(r)
	}
	out := []rune(strings.TrimSpace(b.String()))
	if len(out) > maxAuthorRunes {
		out = out[:maxAuthorRunes]
	}
	return string(out)
}

// ---- API ----

type infoResponse struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Root        string    `json:"root"`
	RootName    string    `json:"rootName"`
	ReadOnly    bool      `json:"readOnly"`
	Port        int       `json:"port"`
	URLs        []string  `json:"urls"`
	DAVPath     string    `json:"davPath"`
	LastEventID int64     `json:"lastEventId"`
	StartedAt   time.Time `json:"startedAt"`
}

func (s *Server) info(w http.ResponseWriter, r *http.Request) {
	rootName := filepath.Base(s.cfg.Root)
	if rootName == "." || rootName == string(filepath.Separator) {
		rootName = "Braise"
	}
	writeJSON(w, http.StatusOK, infoResponse{
		Name:        s.cfg.Name,
		Version:     s.cfg.Version,
		Root:        s.cfg.Root,
		RootName:    rootName,
		ReadOnly:    s.cfg.ReadOnly,
		Port:        s.cfg.Port,
		URLs:        s.cfg.URLs,
		DAVPath:     "/dav",
		LastEventID: s.cfg.Store.Seq(),
		StartedAt:   s.cfg.StartedAt,
	})
}

func (s *Server) messages(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"messages":    s.cfg.Store.Messages(limit),
		"lastEventId": s.cfg.Store.Seq(),
	})
}

type messageRequest struct {
	AuthorID string            `json:"authorId"`
	Author   string            `json:"author"`
	Color    string            `json:"color"`
	Text     string            `json:"text"`
	File     *store.Attachment `json:"file,omitempty"`
}

func (s *Server) postMessage(w http.ResponseWriter, r *http.Request) {
	var req messageRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !idRe.MatchString(req.AuthorID) {
		s.fail(w, http.StatusBadRequest, "identité invalide")
		return
	}
	author := cleanAuthor(req.Author)
	if author == "" {
		s.fail(w, http.StatusBadRequest, "pseudo requis")
		return
	}
	color := req.Color
	if !colorRe.MatchString(color) {
		color = "#f97316"
	}
	text := strings.TrimSpace(req.Text)
	if len([]rune(text)) > maxTextRunes {
		s.fail(w, http.StatusBadRequest, "message trop long")
		return
	}

	var att *store.Attachment
	if req.File != nil && strings.TrimSpace(req.File.Path) != "" {
		rel := normRel(req.File.Path)
		_, fi, err := s.cfg.Files.ResolveExisting(rel)
		if err != nil || fi.IsDir() {
			s.fail(w, http.StatusBadRequest, "pièce jointe invalide")
			return
		}
		att = &store.Attachment{
			Name: path.Base(rel),
			Path: rel,
			Size: fi.Size(),
			MIME: files.MIMEFor(rel),
		}
	}
	if text == "" && att == nil {
		s.fail(w, http.StatusBadRequest, "message vide")
		return
	}

	m := store.Message{
		AuthorID: req.AuthorID,
		Author:   author,
		Color:    color,
		Text:     text,
		At:       time.Now().UTC(),
		File:     att,
	}
	ev, err := s.cfg.Store.AppendFunc("message", func(id int64) any {
		m.ID = id
		m.At = time.Now().UTC()
		return m
	})
	if err != nil {
		s.cfg.Logger.Error("append message", "err", err)
		s.fail(w, http.StatusInternalServerError, "impossible d'enregistrer le message")
		return
	}
	s.cfg.Hub.Broadcast(ev)
	writeJSON(w, http.StatusOK, map[string]any{"message": m, "eventId": ev.ID})
}

func (s *Server) listFiles(w http.ResponseWriter, r *http.Request) {
	rel := normRel(r.URL.Query().Get("path"))
	entries, err := s.cfg.Files.List(rel)
	if err != nil {
		s.fileError(w, err)
		return
	}
	if entries == nil {
		entries = []files.Entry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": rel, "entries": entries})
}

func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	if s.cfg.ReadOnly {
		s.fail(w, http.StatusForbidden, "espace en lecture seule")
		return
	}
	dir := normRel(r.URL.Query().Get("path"))
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	mr, err := r.MultipartReader()
	if err != nil {
		s.fail(w, http.StatusBadRequest, "formulaire multipart attendu")
		return
	}
	out := make([]files.Entry, 0, 4)
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			s.fail(w, http.StatusBadRequest, "envoi interrompu")
			return
		}
		if part.FormName() != "files" {
			part.Close()
			continue
		}
		name := part.FileName()
		if name == "" {
			name = "fichier"
		}
		entry, err := s.cfg.Files.SaveUpload(dir, name, part)
		part.Close()
		if err != nil {
			s.fileError(w, err)
			return
		}
		out = append(out, entry)
		s.broadcastFile("added", entry.Path, dir, &entry, peerID(r))
		if len(out) >= 200 {
			break
		}
	}
	if len(out) == 0 {
		s.fail(w, http.StatusBadRequest, "aucun fichier reçu")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"files": out})
}

func (s *Server) download(w http.ResponseWriter, r *http.Request) {
	s.serveFile(w, r, true)
}

func (s *Server) raw(w http.ResponseWriter, r *http.Request) {
	s.serveFile(w, r, false)
}

func (s *Server) serveFile(w http.ResponseWriter, r *http.Request, attachment bool) {
	rel := normRel(r.URL.Query().Get("path"))
	f, fi, err := s.cfg.Files.Open(rel)
	if err != nil {
		s.fileError(w, err)
		return
	}
	defer f.Close()
	name := filepath.Base(rel)
	ctype := files.MIMEFor(name)
	w.Header().Set("Content-Type", ctype)
	disposition := "inline"
	if attachment {
		disposition = "attachment"
	}
	if cd := mime.FormatMediaType(disposition, map[string]string{"filename": name}); cd != "" {
		w.Header().Set("Content-Disposition", cd)
	}
	if !attachment {
		w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; img-src 'self' data:; media-src 'self'; style-src 'unsafe-inline'")
	}
	http.ServeContent(w, r, name, fi.ModTime(), f)
}

func (s *Server) mkdir(w http.ResponseWriter, r *http.Request) {
	if s.cfg.ReadOnly {
		s.fail(w, http.StatusForbidden, "espace en lecture seule")
		return
	}
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	dir := normRel(req.Path)
	entry, err := s.cfg.Files.Mkdir(dir, req.Name)
	if err != nil {
		s.fileError(w, err)
		return
	}
	s.broadcastFile("mkdir", entry.Path, dir, &entry, peerID(r))
	writeJSON(w, http.StatusCreated, entry)
}

func (s *Server) delete(w http.ResponseWriter, r *http.Request) {
	if s.cfg.ReadOnly {
		s.fail(w, http.StatusForbidden, "espace en lecture seule")
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	rel := normRel(req.Path)
	if rel == "" {
		s.fail(w, http.StatusBadRequest, "chemin requis")
		return
	}
	parent := parentOf(rel)
	if err := s.cfg.Files.Delete(rel); err != nil {
		s.fileError(w, err)
		return
	}
	s.broadcastFile("deleted", rel, parent, nil, peerID(r))
	w.WriteHeader(http.StatusNoContent)
}

type fileEvent struct {
	Action string       `json:"action"`
	Path   string       `json:"path"`
	Parent string       `json:"parent"`
	Entry  *files.Entry `json:"entry,omitempty"`
	Actor  string       `json:"actor,omitempty"`
}

func (s *Server) broadcastFile(action, p, parent string, entry *files.Entry, actor string) {
	data := fileEvent{Action: action, Path: p, Parent: parent, Entry: entry, Actor: actor}
	ev, err := s.cfg.Store.Append("file", data)
	if err != nil {
		s.cfg.Logger.Error("append file event", "err", err)
		return
	}
	s.cfg.Hub.Broadcast(ev)
}

func peerID(r *http.Request) string {
	v := r.Header.Get("X-Braise-Peer")
	if idRe.MatchString(v) {
		return v
	}
	return ""
}

func (s *Server) qr(w http.ResponseWriter, r *http.Request) {
	target := strings.TrimSpace(r.URL.Query().Get("url"))
	if target == "" {
		if len(s.cfg.URLs) > 0 {
			target = s.cfg.URLs[0]
		} else {
			target = fmt.Sprintf("http://localhost:%d", s.cfg.Port)
		}
	}
	if len(target) > 512 {
		s.fail(w, http.StatusBadRequest, "URL trop longue")
		return
	}
	png, err := qrcode.Encode(target, qrcode.Medium, 512)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, "QR impossible")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(png)
}

func (s *Server) presence(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if !idRe.MatchString(req.ID) {
		s.fail(w, http.StatusBadRequest, "identité invalide")
		return
	}
	req.Name = cleanAuthor(req.Name)
	if req.Name == "" {
		req.Name = "Invité"
	}
	if !colorRe.MatchString(req.Color) {
		req.Color = "#f97316"
	}
	peers, changed := s.cfg.Hub.Touch(req.ID, req.Name, req.Color)
	if changed {
		s.cfg.Hub.BroadcastJSON("presence", peers)
	}
	writeJSON(w, http.StatusOK, map[string]any{"peers": peers})
}

func (s *Server) leave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.ID != "" {
		peers := s.cfg.Hub.Leave(req.ID)
		s.cfg.Hub.BroadcastJSON("presence", peers)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) typing(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if !idRe.MatchString(req.ID) {
		s.fail(w, http.StatusBadRequest, "identité invalide")
		return
	}
	req.Name = cleanAuthor(req.Name)
	if req.Name == "" {
		return
	}
	s.cfg.Hub.BroadcastJSON("typing", map[string]string{"id": req.ID, "name": req.Name})
	w.WriteHeader(http.StatusNoContent)
}

// ---- SSE ----

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-cache, no-transform")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	rc := http.NewResponseController(w)
	if err := writeSSE(w, rc, 0, "hello", []byte(`{"ok":true}`)); err != nil {
		return
	}
	if err := writeSSE(w, rc, 0, "presence", mustJSON(s.cfg.Hub.Peers())); err != nil {
		return
	}

	ch := s.cfg.Hub.Subscribe()
	defer s.cfg.Hub.Unsubscribe(ch)

	after := int64(0)
	if v := r.URL.Query().Get("after"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			after = n
		}
	}
	if v := r.Header.Get("Last-Event-ID"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > after {
			after = n
		}
	}
	// On a fresh connection, avoid replaying an unbounded history.
	if after <= 0 {
		if seq := s.cfg.Store.Seq(); seq > 200 {
			after = seq - 200
		}
	}
	last := after
	for _, e := range s.cfg.Store.Since(after, 0) {
		if e.ID <= last {
			continue
		}
		if err := writeSSE(w, rc, e.ID, e.Type, e.Data); err != nil {
			return
		}
		last = e.ID
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if err := writeSSE(w, rc, 0, "ping", []byte(`{}`)); err != nil {
				return
			}
		case e, ok := <-ch:
			if !ok {
				return
			}
			if e.ID > 0 && e.ID <= last {
				continue
			}
			if e.ID > 0 {
				last = e.ID
			}
			if err := writeSSE(w, rc, e.ID, e.Type, e.Data); err != nil {
				return
			}
		}
	}
}

func writeSSE(w io.Writer, rc *http.ResponseController, id int64, event string, data []byte) error {
	var b strings.Builder
	if id > 0 {
		fmt.Fprintf(&b, "id: %d\n", id)
	}
	fmt.Fprintf(&b, "event: %s\n", event)
	b.WriteString("data: ")
	b.Write(data)
	b.WriteString("\n\n")
	_ = rc.SetWriteDeadline(time.Now().Add(30 * time.Second))
	if _, err := io.WriteString(w, b.String()); err != nil {
		return err
	}
	return rc.Flush()
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("null")
	}
	return b
}
