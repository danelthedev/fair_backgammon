package lobby

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"

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
	Url    *string `json:"url"`
	V      *string `json:"v"`
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
	// ponytail: gif reactions bypass turn/game-over checks, ephemeral broadcast
	if msg.T == "gif" {
		r.handleGifLocked(idx, msg.Url, msg.V, sendErr)
		return
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
			g.SetTurn(1 - g.Turn)
			r.DoubledThisTurn = false
		}
		r.broadcastStateLocked()
		if win, w := g.CheckWin(); win {
			mult := g.WinMultiplier(w)
			r.Scores[w] += mult * r.Stake()
			r.Rematch = [2]bool{false, false}
			r.DoubleOffer = nil
			b, _ := json.Marshal(map[string]any{"t": "win", "winner": w, "winnerName": r.Players[w], "scores": r.Scores, "reason": g.WinReason(w), "mult": mult})
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
		// ponytail: tehnic pattern valid only at turn end, mid-turn dice may pass through it
		turnEnded := len(g.MovesLeft) == 0 || !g.HasAnyLegal()
		if turnEnded {
			g.MovesLeft = nil
			g.HasRolled = false
			g.SetTurn(1 - g.Turn)
			r.DoubledThisTurn = false
		}
		r.broadcastStateLocked()
		if win, w := g.CheckWin(); win {
			mult := g.WinMultiplier(w)
			r.Scores[w] += mult * r.Stake()
			r.Rematch = [2]bool{false, false}
			r.DoubleOffer = nil
			b, _ := json.Marshal(map[string]any{"t": "win", "winner": w, "winnerName": r.Players[w], "scores": r.Scores, "reason": g.WinReason(w), "mult": mult})
			for ch := range r.subs {
				select {
				case ch <- b:
				default:
				}
			}
			r.broadcastStateLocked()
		}
		if turnEnded {
			if ok, mult := g.CheckTechnicalWin(game.Player(idx)); ok {
				wp := game.Player(idx)
				r.Scores[wp] += mult * r.Stake()
				r.Rematch = [2]bool{false, false}
				r.DoubleOffer = nil
				r.Game.Off[wp] = 15
				b, _ := json.Marshal(map[string]any{"t": "win", "winner": wp, "winnerName": r.Players[wp], "scores": r.Scores, "reason": "tehnic"})
				for ch := range r.subs {
					select {
					case ch <- b:
					default:
					}
				}
				r.broadcastStateLocked()
			}
		}
	case "reroll":
		if r.DoubleOffer != nil {
			sendErr("double offer pending")
			return
		}
		if err := g.UseReroll(game.Player(idx)); err != nil {
			sendErr(err.Error())
			return
		}
		r.LastMoves = nil
		if !g.HasAnyLegal() {
			g.MovesLeft = nil
			g.HasRolled = false
			g.SetTurn(1 - g.Turn)
			r.DoubledThisTurn = false
		}
		r.broadcastStateLocked()
	case "skip":
		if r.DoubleOffer != nil {
			sendErr("double offer pending")
			return
		}
		if err := g.UseSkip(game.Player(idx)); err != nil {
			sendErr(err.Error())
			return
		}
		r.LastMoves = nil
		r.DoubledThisTurn = false
		r.broadcastStateLocked()
	case "protect":
		if err := g.UseProtect(game.Player(idx)); err != nil {
			sendErr(err.Error())
			return
		}
		r.broadcastStateLocked()
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
		g.SetTurn(1 - g.Turn)
		r.DoubledThisTurn = false
		r.broadcastStateLocked()
	case "rematch":
		if win, _ := g.CheckWin(); !win {
			sendErr("game not over")
			return
		}
		r.Rematch[idx] = true
		if r.Rematch[0] && r.Rematch[1] {
			r.Game = game.NewGameWithMods(r.Game.Mods)
			if r.Layout != nil {
				_ = r.Game.ApplyLayout(*r.Layout) // validated at create
			}
			r.LastMoves = nil
			r.Rematch = [2]bool{false, false}
			r.Cube = 1
			r.DoubleOffer = nil
			r.DoubledThisTurn = false
			r.LastDoubler = -1
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
		// ponytail: resign pays like the finished position — gammon/backgammon count
		mult := g.WinMultiplier(game.Player(winnerIdx))
		r.Scores[winnerIdx] += mult * r.Stake()
		r.Rematch = [2]bool{false, false}
		r.DoubleOffer = nil
		// ponytail: clear winner off the board so CheckWin/rematch see game over
		for i, v := range g.Board {
			if winnerIdx == 0 && v > 0 {
				g.Off[winnerIdx] += v
				g.Board[i] = 0
			} else if winnerIdx == 1 && v < 0 {
				g.Off[winnerIdx] -= v
				g.Board[i] = 0
			}
		}
		g.Off[winnerIdx] += g.Bar[winnerIdx]
		g.Bar[winnerIdx] = 0
		b, _ := json.Marshal(map[string]any{"t": "win", "winner": winnerIdx, "winnerName": r.Players[winnerIdx], "scores": r.Scores, "reason": "resign"})
		for ch := range r.subs {
			select {
			case ch <- b:
			default:
			}
		}
		r.broadcastStateLocked()
	case "double":
		if r.VsFly {
			sendErr("doubling disabled vs fly")
			return
		}
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
		if r.LastDoubler == idx {
			sendErr("wait for opponent to double")
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
		r.LastDoubler = idx
		r.broadcastStateLocked()
	case "double_response":
		if r.VsFly {
			sendErr("doubling disabled vs fly")
			return
		}
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

// ponytail: ephemeral gif relay — no state stored, no turn requirement.
func (r *Room) handleGifLocked(idx int, raw, vid *string, sendErr func(string)) {
	if raw == nil || *raw == "" {
		sendErr("url required")
		return
	}
	u := strings.TrimSpace(*raw)
	if len(u) > 500 || !strings.HasPrefix(u, "https://") {
		sendErr("bad gif url")
		return
	}
	if _, err := url.ParseRequestURI(u); err != nil {
		sendErr("bad gif url")
		return
	}
	now := time.Now().UnixNano()
	if now-r.gifLast[idx] < 3*int64(time.Second) {
		sendErr("gif too fast")
		return
	}
	r.gifLast[idx] = now
	// ponytail: duration fetched async (network) so the room lock isn't held
	go func() {
		dur := gifDuration(u)
		// video-rip gifs lie about frame delays — the mp4 sibling is the ground truth
		if vid != nil && *vid != "" {
			if d := mp4Duration(*vid); d > 0 {
				dur = d
			}
		}
		b, _ := json.Marshal(map[string]any{"t": "gif", "from": idx, "url": u, "dur": dur})
		r.BroadcastRaw(b)
	}()
}

func (r *Room) broadcastStateLocked() {
	msg, _ := json.Marshal(map[string]any{
		"t": "state", "code": r.Code, "board": r.Game.Board, "bar": r.Game.Bar, "off": r.Game.Off,
		"turn": r.Game.Turn, "dice": r.Game.Dice, "movesLeft": r.Game.MovesLeft, "hasRolled": r.Game.HasRolled, "players": r.Players, "lastMoves": r.LastMoves,
		"scores": r.Scores, "rematch": r.Rematch, "cube": r.Cube, "doubleOffer": r.DoubleOffer, "doubledThisTurn": r.DoubledThisTurn, "lastDoubler": r.LastDoubler, "mods": r.Game.Mods, "powerLeft": r.Game.PowerLeft, "shield": r.Game.Shield, "dealt": r.Game.Dealt, "hotseat": r.Hotseat, "rerolled": r.Game.Rerolled, "legalMoves": r.Game.LegalMoves(), "vsFly": r.VsFly,
	})
	for ch := range r.subs {
		select {
		case ch <- msg:
		default:
		}
	}
}
