// Activity logging for future brain visualization. Pure observer: records the
// frozen reservoir state but never influences decisions.
// Gated by FLY_ACTIVITY_LOG=path; no-op when unset.
package fly

import (
	"encoding/json"
	"os"
	"sort"

	"fair_backgammon/game"
	"fair_backgammon/lobby"
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

// BroadcastActivity pushes the live brain frame to room subscribers.
// Observer only: same phi as the decision; top-40 firing units + top-150
// pathway edges. Powers the arena brain panel.
func BroadcastActivity(h *Head, w []float64, variant string, room *lobby.Room, board [24]int, bar, off [2]int, turn int, m game.Move) {
	g := &game.Game{Board: board, Bar: bar, Off: off,
		Turn: game.Player(turn), MovesLeft: []int{m.Die}, HasRolled: true}
	if err := g.Apply(m); err != nil {
		return
	}
	phi := Phi(h, w, g.Board, g.Bar, g.Off, int(g.Turn))
	fire := make([]float64, h.N)
	for i := 0; i < h.N; i++ {
		f := phi[i]
		if f < 0 {
			f = -f
		}
		fire[i] = f
	}
	top := topIdx(fire, 40)
	inTop := make([]bool, h.N)
	for _, i := range top {
		inTop[i] = true
	}
	type edge struct {
		i, j int
		w    float64
	}
	seen := make(map[[2]int]bool)
	edges := []edge{}
	add := func(i, j int, w float64) {
		a, b := i, j
		if a > b {
			a, b = b, a
		}
		if seen[[2]int{a, b}] {
			return
		}
		seen[[2]int{a, b}] = true
		edges = append(edges, edge{i, j, w})
	}
	// partner-first frame: every lit unit brings its strongest incoming
	// partner, so no firing neuron is left without lines.
	pedge := make(map[int]edge)
	for _, i := range top {
		bj, bv := -1, -1.0
		for k := h.Indptr[i]; k < h.Indptr[i+1]; k++ {
			j := h.Indices[k]
			if j < 0 || j >= h.N {
				continue
			}
			c := h.Data[k] * phi[j]
			if c < 0 {
				c = -c
			}
			if c > bv {
				bv, bj = c, j
			}
		}
		if bj >= 0 {
			pedge[i] = edge{i, bj, bv}
		}
	}
	frUnits := append([]int{}, top...)
	po := make([]int, 0)
	seenP := make(map[int]bool)
	for _, i := range top {
		if e, ok := pedge[i]; ok && !inTop[e.j] && !seenP[e.j] {
			seenP[e.j] = true
			po = append(po, e.j)
		}
	}
	sort.Slice(po, func(a, b int) bool { return fire[po[a]] > fire[po[b]] })
	for _, j := range po {
		if len(frUnits) >= 80 {
			break
		}
		frUnits = append(frUnits, j)
	}
	inFrame := make([]bool, h.N)
	for _, i := range frUnits {
		inFrame[i] = true
	}
	for _, i := range top {
		if e, ok := pedge[i]; ok && inFrame[e.j] {
			add(e.i, e.j, e.w)
		}
	}
	for _, i := range top {
		best := []edge{}
		for k := h.Indptr[i]; k < h.Indptr[i+1]; k++ {
			j := h.Indices[k]
			if j < 0 || j >= h.N || !inFrame[j] {
				continue
			}
			c := h.Data[k] * phi[j]
			if c < 0 {
				c = -c
			}
			best = append(best, edge{i, j, c})
		}
		sort.Slice(best, func(a, b int) bool { return best[a].w > best[b].w })
		for k := 0; k < 2 && k < len(best); k++ {
			add(best[k].i, best[k].j, best[k].w)
		}
	}
	var rest []edge
	for _, i := range top {
		for k := h.Indptr[i]; k < h.Indptr[i+1]; k++ {
			j := h.Indices[k]
			if j < 0 || j >= h.N || !inFrame[j] {
				continue
			}
			c := h.Data[k] * phi[j]
			if c < 0 {
				c = -c
			}
			rest = append(rest, edge{i, j, c})
		}
	}
	sort.Slice(rest, func(a, b int) bool { return rest[a].w > rest[b].w })
	for _, e := range rest {
		if len(edges) >= 200 {
			break
		}
		add(e.i, e.j, e.w)
	}
	units := make([][2]float64, len(frUnits))
	for k, i := range frUnits {
		units[k] = [2]float64{float64(i), fire[i]}
	}
	pairs := make([][2]int, len(edges))
	for k, e := range edges {
		pairs[k] = [2]int{e.i, e.j}
	}
	buf, err := json.Marshal(map[string]any{
		"t": "brain", "variant": variant, "units": units, "edges": pairs,
	})
	if err != nil {
		return
	}
	room.BroadcastRaw(buf)
}

func topIdx(fire []float64, k int) []int {
	idx := make([]int, len(fire))
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool { return fire[idx[a]] > fire[idx[b]] })
	if len(idx) > k {
		idx = idx[:k]
	}
	return idx
}
