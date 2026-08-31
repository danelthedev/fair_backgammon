package lobby

import (
	"encoding/json"
	"sync"

	"fair_backgammon/game"
)

type Hub struct {
	mu    sync.Mutex
	games map[string]*Room
}

type Room struct {
	Code    string
	Game    *game.Game
	Players [2]string // username, empty = slot free
	LastMoves []game.Move

	mu   sync.Mutex
	subs map[chan []byte]struct{}
}
func NewHub() *Hub { return &Hub{games: make(map[string]*Room)} }

func (h *Hub) Create(username string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	var code string
	for {
		code = genCode()
		if h.games[code] == nil {
			break
		}
	}
	r := &Room{Code: code, Game: game.NewGame(), subs: make(map[chan []byte]struct{})}
	r.Players[0] = username
	h.games[code] = r
	return r
}

func (h *Hub) Join(code, username string) (*Room, int, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r := h.games[code]
	if r == nil {
		return nil, -1, false
	}
	// already in?
	for i, p := range r.Players {
		if p == username {
			return r, i, true
		}
	}
	for i, p := range r.Players {
		if p == "" {
			r.Players[i] = username
			return r, i, true
		}
	}
	return nil, -1, false // full
}

func (h *Hub) Get(code string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.games[code]
}

func (r *Room) PlayerIndex(username string) int {
	for i, p := range r.Players {
		if p == username {
			return i
		}
	}
	return -1
}

func (r *Room) BroadcastState() {
	r.mu.Lock()
	defer r.mu.Unlock()
	msg, _ := json.Marshal(map[string]any{
		"t":         "state",
		"code":      r.Code,
		"board":     r.Game.Board,
		"bar":       r.Game.Bar,
		"off":       r.Game.Off,
		"turn":      r.Game.Turn,
		"dice":      r.Game.Dice,
		"movesLeft": r.Game.MovesLeft,
		"hasRolled": r.Game.HasRolled,
		"players":   r.Players,
		"lastMoves": r.LastMoves,
	})
	for ch := range r.subs {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (r *Room) AddSub(ch chan []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subs[ch] = struct{}{}
}
func (r *Room) RemoveSub(ch chan []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.subs, ch)
}
func (r *Room) BroadcastRaw(b []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for ch := range r.subs {
		select {
		case ch <- b:
		default:
		}
	}
}
