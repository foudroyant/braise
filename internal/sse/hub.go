// Package sse provides a small fan-out hub for Server-Sent Events and
// best-effort presence tracking.
package sse

import (
	"encoding/json"
	"sort"
	"sync"
	"time"

	"braise/internal/store"
)

// Peer is someone currently connected to the space.
type Peer struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Color    string    `json:"color"`
	LastSeen time.Time `json:"-"`
}

// Hub broadcasts events to every subscribed SSE client.
type Hub struct {
	mu    sync.Mutex
	subs  map[chan store.Event]struct{}
	peers map[string]*Peer
}

// New returns an empty hub.
func New() *Hub {
	return &Hub{
		subs:  make(map[chan store.Event]struct{}),
		peers: make(map[string]*Peer),
	}
}

// Subscribe registers a client. The returned channel is closed by Unsubscribe.
func (h *Hub) Subscribe() chan store.Event {
	ch := make(chan store.Event, 128)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes a client and closes its channel.
func (h *Hub) Unsubscribe(ch chan store.Event) {
	h.mu.Lock()
	if _, ok := h.subs[ch]; ok {
		delete(h.subs, ch)
		close(ch)
	}
	h.mu.Unlock()
}

// Broadcast sends e to every client without blocking. A slow client misses
// events and recovers them through replay on reconnect.
func (h *Hub) Broadcast(e store.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- e:
		default:
		}
	}
}

// BroadcastJSON sends an ephemeral (non-persisted) event to every client.
func (h *Hub) BroadcastJSON(typ string, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	h.Broadcast(store.Event{Type: typ, At: time.Now().UTC(), Data: raw})
}

const peerTTL = 45 * time.Second

// Touch records a heartbeat, adds the peer if new, and returns the live list
// plus whether the visible roster changed.
func (h *Hub) Touch(id, name, color string) ([]Peer, bool) {
	if id == "" {
		return h.Peers(), false
	}
	now := time.Now()
	h.mu.Lock()
	defer h.mu.Unlock()
	p, ok := h.peers[id]
	changed := !ok
	if !ok {
		h.peers[id] = &Peer{ID: id, Name: name, Color: color, LastSeen: now}
	} else {
		p.LastSeen = now
		if name != "" && p.Name != name {
			p.Name = name
			changed = true
		}
		if color != "" && p.Color != color {
			p.Color = color
			changed = true
		}
	}
	return h.peersLocked(now), changed
}

// Leave removes a peer immediately.
func (h *Hub) Leave(id string) []Peer {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.peers, id)
	return h.peersLocked(time.Now())
}

// Peers returns live peers and prunes stale ones.
func (h *Hub) Peers() []Peer {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.peersLocked(time.Now())
}

// Prune removes peers that stopped heartbeating and reports whether the list
// changed.
func (h *Hub) Prune() ([]Peer, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	changed := false
	for id, p := range h.peers {
		if now.Sub(p.LastSeen) > peerTTL {
			delete(h.peers, id)
			changed = true
		}
	}
	return h.peersLocked(now), changed
}

func (h *Hub) peersLocked(now time.Time) []Peer {
	out := make([]Peer, 0, len(h.peers))
	for _, p := range h.peers {
		if now.Sub(p.LastSeen) <= peerTTL {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}
