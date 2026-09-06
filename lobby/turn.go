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
	T      string  `json:"t"`
	From   *int    `json:"from"`
	To     *int    `json:"to"`
	Die    *int    `json:"die"`
	Action *string `json:"action"`
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
	if win, _ := g.CheckWin(); win {
		if msg.T != "rematch" {
			sendErr("game over, request rematch")
			return
		}
	} else if msg.T != "resign" && msg.T != "rematch" && msg.T != "double_response" && g.Turn != game.Player(idx) {
		sendErr("not your turn")
		return
	}
	switch msg.T {
	case "roll":
		if r.Players[0] == "" || r.Players[1] == "" {
			sendErr("waiting for opponent")
			return
		}
		if r.DoubleOffer != nil {
			sendErr("double offer pending")
			return
		}
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
			r.DoubledThisTurn = false
		}
		r.broadcastStateLocked()
		if win, w := g.CheckWin(); win {
			r.Scores[w] += r.Stake()
			r.Rematch = [2]bool{false, false}
			r.DoubleOffer = nil
			b, _ := json.Marshal(map[string]any{"t": "win", "winner": w, "winnerName": r.Players[w], "scores": r.Scores})
			for ch := range r.subs {
				select {
				case ch <- b:
				default:
				}
			}
			r.broadcastStateLocked()
		}
	case "move":
		if r.DoubleOffer != nil {
			sendErr("double offer pending")
			return
		}
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
			r.DoubledThisTurn = false
		}
		r.broadcastStateLocked()
		if win, w := g.CheckWin(); win {
			r.Scores[w] += r.Stake()
			r.Rematch = [2]bool{false, false}
			r.DoubleOffer = nil
			b, _ := json.Marshal(map[string]any{"t": "win", "winner": w, "winnerName": r.Players[w], "scores": r.Scores})
			for ch := range r.subs {
				select {
				case ch <- b:
				default:
				}
			}
			r.broadcastStateLocked()
		}
	case "pass":
		if r.DoubleOffer != nil {
			sendErr("double offer pending")
			return
		}
		if g.HasAnyLegal() {
			sendErr("you have legal moves")
			return
		}
		r.LastMoves = nil
		g.MovesLeft = nil
		g.HasRolled = false
		g.Turn = 1 - g.Turn
		r.DoubledThisTurn = false
		r.broadcastStateLocked()
	case "rematch":
		if win, _ := g.CheckWin(); !win {
			sendErr("game not over")
			return
		}
		r.Rematch[idx] = true
		if r.Rematch[0] && r.Rematch[1] {
			r.Game = game.NewGame()
			r.LastMoves = nil
			r.Rematch = [2]bool{false, false}
			r.Cube = 1
			r.DoubleOffer = nil
			r.DoubledThisTurn = false
			// swap colors on rematch
			r.Players[0], r.Players[1] = r.Players[1], r.Players[0]
			r.Scores[0], r.Scores[1] = r.Scores[1], r.Scores[0]
			r.broadcastStateLocked()
		} else {
			r.broadcastStateLocked()
			b, _ := json.Marshal(map[string]any{"t": "rematch", "rematch": r.Rematch, "scores": r.Scores})
			for ch := range r.subs {
				select {
				case ch <- b:
				default:
				}
			}
		}
	case "resign":
		if win, _ := g.CheckWin(); win {
			sendErr("game already over")
			return
		}
		if r.Players[0] == "" || r.Players[1] == "" {
			sendErr("waiting for opponent")
			return
		}
		winnerIdx := 1 - idx
		r.Scores[winnerIdx] += r.Stake()
		r.Rematch = [2]bool{false, false}
		r.DoubleOffer = nil
		r.Game.Off[winnerIdx] = 15
		b, _ := json.Marshal(map[string]any{"t": "win", "winner": winnerIdx, "winnerName": r.Players[winnerIdx], "scores": r.Scores, "reason": "resign"})
		for ch := range r.subs {
			select {
			case ch <- b:
			default:
			}
		}
		r.broadcastStateLocked()
	case "double":
		if win, _ := g.CheckWin(); win {
			sendErr("game over, request rematch")
			return
		}
		if g.Turn != game.Player(idx) {
			sendErr("not your turn")
			return
		}
		if g.HasRolled {
			sendErr("already rolled, cannot double")
			return
		}
		if r.DoubleOffer != nil {
			sendErr("double already offered")
			return
		}
		if r.DoubledThisTurn {
			sendErr("already doubled this turn")
			return
		}
		if r.Stake() >= 64 {
			sendErr("already at 64")
			return
		}
		r.DoubleOffer = &DoubleOffer{By: idx, Stake: r.Stake() * 2}
		r.DoubledThisTurn = true
		r.broadcastStateLocked()
	case "double_response":
		if r.DoubleOffer == nil {
			sendErr("no double offer")
			return
		}
		offer := r.DoubleOffer
		if idx == offer.By {
			sendErr("wait for opponent response")
			return
		}
		if msg.Action == nil {
			sendErr("action required")
			return
		}
		switch *msg.Action {
		case "accept":
			r.Cube = offer.Stake
			r.DoubleOffer = nil
			r.broadcastStateLocked()
		case "reject":
			w := offer.By
			r.Scores[w] += r.Stake()
			r.Rematch = [2]bool{false, false}
			r.DoubleOffer = nil
			r.Game.Off[w] = 15
			b, _ := json.Marshal(map[string]any{"t": "win", "winner": w, "winnerName": r.Players[w], "scores": r.Scores, "reason": "double_refused"})
			for ch := range r.subs {
				select {
				case ch <- b:
				default:
				}
			}
			r.broadcastStateLocked()
		case "redouble":
			if offer.Stake >= 64 {
				sendErr("already at 64")
				return
			}
			r.Cube = offer.Stake * 2
			r.DoubleOffer = nil
			r.DoubledThisTurn = true
			r.broadcastStateLocked()
		default:
			sendErr("unknown action")
		}
	default:
		sendErr("unknown t")
	}
}

func (r *Room) broadcastStateLocked() {
	msg, _ := json.Marshal(map[string]any{
		"t": "state", "code": r.Code, "board": r.Game.Board, "bar": r.Game.Bar, "off": r.Game.Off,
		"turn": r.Game.Turn, "dice": r.Game.Dice, "movesLeft": r.Game.MovesLeft, "hasRolled": r.Game.HasRolled, "players": r.Players, "lastMoves": r.LastMoves,
		"scores": r.Scores, "rematch": r.Rematch, "cube": r.Cube, "doubleOffer": r.DoubleOffer, "doubledThisTurn": r.DoubledThisTurn,
	})
	for ch := range r.subs {
		select {
		case ch <- msg:
		default:
		}
	}
}
