// Activity logging for future brain visualization. Pure observer: records the
// frozen reservoir state but never influences decisions.
// Gated by FLY_ACTIVITY_LOG=path; no-op when unset.
package fly

import (
	"encoding/json"
	"os"

	"fair_backgammon/game"
)

// LogActivity appends one JSON line per decided move: the reservoir vector x
// (Kenyon-cell activations) for the chosen resulting state plus the MBON
// value v. Powers the "light up used pathways" display.
func LogActivity(h *Head, w []float64, variant string, board [24]int, bar, off [2]int, turn int, m game.Move) {
	path := os.Getenv("FLY_ACTIVITY_LOG")
	if path == "" {
		return
	}
	g := &game.Game{Board: board, Bar: bar, Off: off,
		Turn: game.Player(turn), MovesLeft: []int{m.Die}, HasRolled: true}
	if err := g.Apply(m); err != nil {
		return
	}
	phi := Phi(h, w, g.Board, g.Bar, g.Off, int(g.Turn))
	v := 0.0
	for i, c := range w {
		v += c * phi[i]
	}
	rec := map[string]any{
		"variant": variant,
		"board":   board, "bar": bar, "off": off, "turn": turn,
		"move": map[string]int{"from": m.From, "to": m.To, "die": m.Die},
		"x":    phi[:h.N],
		"v":    v,
	}
	buf, err := json.Marshal(rec)
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	buf = append(buf, '\n')
	_, _ = f.Write(buf)
}
