package lobby

import (
	"encoding/json"

	"fair_backgammon/game"
)

// TurnHandler allows main to inject logic if wanted, but default impl inside lobby
var TurnHandler func(room *Room, idx int, t string, from, to, die *int, sendErr func(string))

func (r *Room) GameTurnLocked(_ interface{}, _ *Room, _ int, _ any) {} // stub for main reference, keep compile

func (r *Room) GameTurn(conn interface {
	WriteMessage(int, []byte) error
}, _ *Room, _ string, idx int, msg struct {
	T    string `json:"t"`
	From *int   `json:"from"`
	To   *int   `json:"to"`
	Die  *int   `json:"die"`
}, replyCh ...chan []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var ch chan []byte
	if len(replyCh) > 0 {
		ch = replyCh[0]
	}
	sendErr := func(s string) {
		b, _ := json.Marshal(map[string]string{"t": "error", "msg": s})
		if ch != nil {
			select {
			case ch <- b:
			default:
			}
			return
		}
		_ = conn.WriteMessage(1, b)
	}
	if TurnHandler != nil {
		TurnHandler(r, idx, msg.T, msg.From, msg.To, msg.Die, sendErr)
		return
	}
	// fallback (should not hit, TurnHandler set in main init)
	g := r.Game
	if g.Turn != game.Player(idx) {
		sendErr("not your turn")
		return
	}
	switch msg.T {
	case "roll":
		if g.HasRolled {
			sendErr("already rolled")
			return
		}
		r.LastMoves = nil // ponytail: clear previous turn anim
		g.Roll()
		if !g.HasAnyLegal() {
			g.HasRolled = false
			g.MovesLeft = nil
			g.Turn = 1 - g.Turn
		}
		r.broadcastStateLocked()
		if win, w := g.CheckWin(); win {
			b, _ := json.Marshal(map[string]any{"t": "win", "winner": w})
			for ch := range r.subs {
				select { case ch <- b: default: }
			}
		}
	case "move":
		if msg.From == nil || msg.To == nil || msg.Die == nil {
			sendErr("from/to/die required")
			return
		}
		m := game.Move{From: *msg.From, To: *msg.To, Die: *msg.Die}
		if err := g.Apply(m); err != nil {
			sendErr(err.Error())
			return
		}
		r.LastMoves = append(r.LastMoves, m)
		if len(g.MovesLeft) == 0 || !g.HasAnyLegal() {
			g.MovesLeft = nil
			g.HasRolled = false
			g.Turn = 1 - g.Turn
		}
		r.broadcastStateLocked()
		if win, w := g.CheckWin(); win {
			b, _ := json.Marshal(map[string]any{"t": "win", "winner": w})
			for ch := range r.subs {
				select { case ch <- b: default: }
			}
		}
	case "pass":
		if g.HasAnyLegal() {
			sendErr("you have legal moves")
			return
		}
		r.LastMoves = nil
		g.MovesLeft = nil
		g.HasRolled = false
		g.Turn = 1 - g.Turn
		r.broadcastStateLocked()
	default:
		sendErr("unknown t")
	}
}

func (r *Room) broadcastStateLocked() {
	msg, _ := json.Marshal(map[string]any{
		"t": "state", "code": r.Code, "board": r.Game.Board, "bar": r.Game.Bar, "off": r.Game.Off,
		"turn": r.Game.Turn, "dice": r.Game.Dice, "movesLeft": r.Game.MovesLeft, "hasRolled": r.Game.HasRolled, "players": r.Players, "lastMoves": r.LastMoves,
	})
	for ch := range r.subs {
		select {
		case ch <- msg:
		default:
		}
	}
}
