// In-process bot seat. Subscribes to room broadcasts and calls the same
// GameTurn the WS handler calls. No sockets, no subprocess.
package fly

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"fair_backgammon/game"
	"fair_backgammon/lobby"
)

type discarder struct{}

func (discarder) WriteMessage(int, []byte) error { return nil }

// turnMsg mirrors lobby's anonymous GameTurn message exactly (tags included).
type turnMsg struct {
	T      string  `json:"t"`
	From   *int    `json:"from"`
	To     *int    `json:"to"`
	Die    *int    `json:"die"`
	Action *string `json:"action"`
}

type stateMsg struct {
	Board       [24]int     `json:"board"`
	Bar         [2]int      `json:"bar"`
	Off         [2]int      `json:"off"`
	Turn        int         `json:"turn"`
	MovesLeft   []int       `json:"movesLeft"`
	HasRolled   bool        `json:"hasRolled"`
	Players     [2]string   `json:"players"`
	DoubleOffer any         `json:"doubleOffer"`
	LegalMoves  []game.Move `json:"legalMoves"`
}

// Play occupies the bot seat until the human leaves. Color follows Players
// (rematch swaps); rematch auto-accepted; room released on exit.
func Play(hub *lobby.Hub, code, botname, variant string, heads map[string]*Head) {
	h := heads[variant]
	if h == nil {
		h = heads["retarded"] // default arena head
		if h == nil {
			for _, hh := range heads {
				h = hh
				break
			}
		}
	}
	w := h.W[variant]
	if w == nil { // never expected: each variant head carries its own key
		for _, ww := range h.W {
			w = ww
			break
		}
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	room := hub.Get(code)
	if room == nil {
		return
	}
	ch := make(chan []byte, 64)
	room.AddSub(ch, botname)
	defer room.RemoveSub(ch)
	defer hub.Leave(code, botname)
	var dc discarder
	last, rematched, justRolled := "", false, false
	pace := func() { time.Sleep(150 * time.Millisecond) }
	// liveness: forward broadcasts; every 20s idle re-request server truth
	// (nil = nudge) so a missed broadcast can never stall the seat.
	live := make(chan []byte, 64)
	done := make(chan struct{})
	defer close(done)
	go func() {
		t := time.NewTimer(20 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case raw, ok := <-ch:
				if !ok {
					return
				}
				select {
				case live <- raw:
				case <-done:
					return
				}
				if !t.Stop() {
					select {
					case <-t.C:
					default:
					}
				}
				t.Reset(20 * time.Second)
			case <-t.C:
				room.BroadcastState()
				select {
				case live <- nil:
				case <-done:
					return
				}
				t.Reset(20 * time.Second)
			}
		}
	}()
	for raw := range live {
		if raw == nil {
			last = "" // re-decide from fresh truth
			continue
		}
		var base struct {
			T string `json:"t"`
		}
		if err := json.Unmarshal(raw, &base); err != nil {
			continue
		}
		switch base.T {
		case "opponent_left":
			return
		case "win":
			if !rematched {
				room.GameTurn(dc, nil, "", room.PlayerIndex(botname), turnMsg{T: "rematch"})
				rematched = true
			}
			last = ""
			continue
		case "state":
		default:
			continue
		}
		var st stateMsg
		if err := json.Unmarshal(raw, &st); err != nil {
			continue
		}
		idx := -1
		for i, p := range st.Players {
			if p == botname {
				idx = i
			}
		}
		if idx == -1 || st.Off[0] >= 15 || st.Off[1] >= 15 {
			continue // not seated / game over
		}
		if st.Turn != idx || st.DoubleOffer != nil {
			continue
		}
		if last == "" && st.Off == [2]int{} {
			rematched = false // fresh game
		}
		key := fmt.Sprintf("%v%v%v%v%v", st.Board, st.Bar, st.MovesLeft, st.HasRolled, st.Turn)
		if key == last {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		last = key
		pace() // human-like pacing, lets animations play
		switch {
		case !st.HasRolled:
			room.GameTurn(dc, nil, "", idx, turnMsg{T: "roll"})
			justRolled = true
		case len(st.LegalMoves) == 0:
			room.GameTurn(dc, nil, "", idx, turnMsg{T: "pass"})
			justRolled = false
		default:
			firstMove := justRolled
			if justRolled {
				// let the dice roll animation (600ms) finish before the first move
				time.Sleep(900 * time.Millisecond)
				justRolled = false
			}
			// Pure fly: every variant decides 1-ply. Lookahead search lives only
			// as offline training scaffolding in musca; never called live.
			m := Choose(h, w, st.Board, st.Bar, st.Off, st.Turn, st.LegalMoves, rng)
			LogActivity(h, w, variant, st.Board, st.Bar, st.Off, st.Turn, m)
			if firstMove {
				BroadcastActivity(h, w, variant, room, st.Board, st.Bar, st.Off, st.Turn, m)
				// Think-first: the panel wave (~2s) plays before the move lands.
				time.Sleep(2000 * time.Millisecond)
			}
			room.GameTurn(dc, nil, "", idx, turnMsg{T: "move", From: &m.From, To: &m.To, Die: &m.Die})
		}
	}
}
