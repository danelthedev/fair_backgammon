package lobby

import (
	"encoding/json"
	"sync"
	"time"

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
	Scores    [2]int
	Rematch   [2]bool

	mu        sync.Mutex
	subs      map[chan []byte]struct{}
	userSubs  map[chan []byte]string
	connCount map[string]int
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
	r := &Room{Code: code, Game: game.NewGame(), subs: make(map[chan []byte]struct{}), userSubs: make(map[chan []byte]string), connCount: make(map[string]int)}
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

func (h *Hub) Leave(code, username string) {
	h.mu.Lock()
	r := h.games[code]
	if r == nil {
		h.mu.Unlock()
		return
	}
	for i, p := range r.Players {
		if p == username {
			r.Players[i] = ""
			r.Scores[i] = 0
			r.Rematch[i] = false
		}
	}
	shouldDelete := r.Players[0] == "" && r.Players[1] == ""
	if shouldDelete {
		delete(h.games, code)
		h.mu.Unlock()
		return
	}
	h.mu.Unlock()
	r.mu.Lock()
	msg, _ := json.Marshal(map[string]any{"t": "opponent_left", "scores": r.Scores})
	for ch := range r.subs {
		select { case ch <- msg: default: }
	}
	r.mu.Unlock()
	r.BroadcastState()
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
		"scores":    r.Scores,
		"rematch":   r.Rematch,
	})
	for ch := range r.subs {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (r *Room) AddSub(ch chan []byte, username string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subs[ch] = struct{}{}
	r.userSubs[ch] = username
	r.connCount[username]++
}

func (r *Room) RemoveSub(ch chan []byte) {
	r.mu.Lock()
	username := r.userSubs[ch]
	delete(r.subs, ch)
	delete(r.userSubs, ch)
	if username != "" {
		if r.connCount[username] > 0 {
			r.connCount[username]--
		}
		// if user has no more connections, schedule leave check
		if r.connCount[username] == 0 {
			// check if user is still in Players but has no conn - they may have closed tab
			// don't immediately remove, let them reconnect within a short window
			// we will handle via explicit Leave or via timeout in wsHandler
		}
	}
	r.mu.Unlock()
}

func (h *Hub) HandleDisconnect(code, username string) {
	go func() {
		select {
		case <-time.After(1500 * time.Millisecond):
		}
		h.mu.Lock()
		r := h.games[code]
		if r == nil {
			h.mu.Unlock()
			return
		}
		r.mu.Lock()
		if r.connCount[username] > 0 {
			r.mu.Unlock()
			h.mu.Unlock()
			return
		}
		idx := -1
		for i, p := range r.Players {
			if p == username {
				idx = i
				break
			}
		}
		if idx == -1 {
			r.mu.Unlock()
			h.mu.Unlock()
			return
		}
		// treat disconnect as leave - disband lobby
		r.Players[idx] = ""
		r.Scores[idx] = 0
		r.Rematch[idx] = false
		shouldDelete := r.Players[0] == "" && r.Players[1] == ""
		r.mu.Unlock()
		if shouldDelete {
			delete(h.games, code)
			h.mu.Unlock()
			return
		}
		h.mu.Unlock()
		r.mu.Lock()
		msg, _ := json.Marshal(map[string]any{"t": "opponent_left", "scores": r.Scores})
		for ch := range r.subs {
			select { case ch <- msg: default: }
		}
		r.mu.Unlock()
		r.BroadcastState()
	}()
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
