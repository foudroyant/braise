package store

import (
	"testing"
	"time"
)

func TestAppendMessagesAndReopen(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	ev, err := s.AppendFunc("message", func(id int64) any {
		return Message{ID: id, AuthorID: "peer-123456", Author: "Alice", Color: "#f97316", Text: "bonjour", At: time.Now().UTC()}
	})
	if err != nil {
		t.Fatal(err)
	}
	if ev.ID != 1 {
		t.Fatalf("expected event ID 1, got %d", ev.ID)
	}
	if _, err := s.Append("file", map[string]string{"path": "a.txt"}); err != nil {
		t.Fatal(err)
	}
	msgs := s.Messages(10)
	if len(msgs) != 1 || msgs[0].Text != "bonjour" || msgs[0].ID != ev.ID {
		t.Fatalf("unexpected messages: %+v", msgs)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s2, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if got := s2.Seq(); got != 2 {
		t.Fatalf("expected seq 2 after reopen, got %d", got)
	}
	if got := len(s2.Since(0, 0)); got != 2 {
		t.Fatalf("expected 2 replayed events, got %d", got)
	}
	if got := len(s2.Messages(10)); got != 1 {
		t.Fatalf("expected message history to survive reopen, got %d", got)
	}
}

func TestSinceLimit(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for i := 0; i < 3; i++ {
		if _, err := s.Append("test", i); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(s.Since(1, 0)); got != 2 {
		t.Fatalf("Since(1) = %d, want 2", got)
	}
	if got := len(s.Since(0, 2)); got != 2 {
		t.Fatalf("Since(0, limit 2) = %d, want 2", got)
	}
}
