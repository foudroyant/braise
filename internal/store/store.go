// Package store persists the event log and chat messages of a Braise space.
//
// V0 uses an append-only JSONL file inside <root>/.braise. The whole log is
// kept in memory, which is plenty for a LAN space; V1 will move to SQLite.
package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Event is the unit broadcast over SSE and persisted to disk.
type Event struct {
	ID   int64           `json:"id"`
	Type string          `json:"type"`
	At   time.Time       `json:"at"`
	Data json.RawMessage `json:"data"`
}

// Attachment describes a file attached to a chat message.
type Attachment struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
	MIME string `json:"mime"`
}

// Message is the payload of a "message" event.
type Message struct {
	ID       int64       `json:"id"`
	AuthorID string      `json:"authorId"`
	Author   string      `json:"author"`
	Color    string      `json:"color"`
	Text     string      `json:"text"`
	At       time.Time   `json:"at"`
	File     *Attachment `json:"file,omitempty"`
}

// Store is a concurrency-safe append-only event store.
type Store struct {
	mu     sync.RWMutex
	meta   string
	path   string
	events []Event
	seq    int64
	f      *os.File
}

const (
	maxLogBytes  = 8 << 20
	keepOnRotate = 2000
)

// Open loads (and creates if needed) the event log under root/.braise.
func Open(root string) (*Store, error) {
	meta := filepath.Join(root, ".braise")
	if err := os.MkdirAll(meta, 0o755); err != nil {
		return nil, fmt.Errorf("create metadata dir: %w", err)
	}
	s := &Store{meta: meta, path: filepath.Join(meta, "events.jsonl")}
	if err := s.load(); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open event log: %w", err)
	}
	s.f = f
	return s, nil
}

func (s *Store) load() error {
	fi, err := os.Stat(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	f, err := os.Open(s.path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Event
		if err := json.Unmarshal(line, &e); err != nil {
			// Tolerance for a partially written last line after a crash.
			continue
		}
		if e.ID > s.seq {
			s.seq = e.ID
		}
		s.events = append(s.events, e)
	}
	if err := sc.Err(); err != nil {
		return err
	}

	// Rotate oversized logs, keeping the most recent events and IDs.
	if fi.Size() > maxLogBytes && len(s.events) > keepOnRotate {
		keep := append([]Event(nil), s.events[len(s.events)-keepOnRotate:]...)
		rotated := fmt.Sprintf("%s.%s", s.path, time.Now().UTC().Format("20060102-150405"))
		if err := os.Rename(s.path, rotated); err != nil {
			return err
		}
		nf, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		enc := json.NewEncoder(nf)
		for _, e := range keep {
			if err := enc.Encode(e); err != nil {
				nf.Close()
				return err
			}
		}
		if err := nf.Close(); err != nil {
			return err
		}
		s.events = keep
	}
	return nil
}

// Close flushes and closes the underlying log.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.f == nil {
		return nil
	}
	err := s.f.Close()
	s.f = nil
	return err
}

// Append persists an event and assigns it the next ID.
func (s *Store) Append(typ string, data any) (Event, error) {
	return s.AppendFunc(typ, func(int64) any { return data })
}

// AppendFunc is Append with access to the assigned ID, so payloads such as
// messages can carry their event ID.
func (s *Store) AppendFunc(typ string, build func(id int64) any) (Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.f == nil {
		return Event{}, errors.New("store is closed")
	}
	s.seq++
	id := s.seq
	raw, err := json.Marshal(build(id))
	if err != nil {
		return Event{}, err
	}
	e := Event{ID: id, Type: typ, At: time.Now().UTC(), Data: raw}
	line, err := json.Marshal(e)
	if err != nil {
		return Event{}, err
	}
	if _, err := s.f.Write(append(line, '\n')); err != nil {
		return Event{}, err
	}
	s.events = append(s.events, e)
	return e, nil
}

// Seq returns the current last event ID.
func (s *Store) Seq() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.seq
}

// Since returns events with ID greater than after, in order. limit <= 0 means
// no limit.
func (s *Store) Since(after int64, limit int) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Event, 0, 16)
	for _, e := range s.events {
		if e.ID <= after {
			continue
		}
		out = append(out, e)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

// Last returns the most recent limit events.
func (s *Store) Last(limit int) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.events) {
		limit = len(s.events)
	}
	out := make([]Event, limit)
	copy(out, s.events[len(s.events)-limit:])
	return out
}

// Messages returns the most recent limit chat messages, oldest first.
func (s *Store) Messages(limit int) []Message {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Message, 0, limit)
	for i := len(s.events) - 1; i >= 0 && len(out) < limit; i-- {
		if s.events[i].Type != "message" {
			continue
		}
		var m Message
		if err := json.Unmarshal(s.events[i].Data, &m); err != nil {
			continue
		}
		if m.At.IsZero() {
			m.At = s.events[i].At
		}
		out = append(out, m)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// MetaDir returns the path of the hidden metadata directory.
func (s *Store) MetaDir() string { return s.meta }
