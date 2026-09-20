// Package fly: in-process backgammon bot. Data (fly.json) is exported from
// musca via scripts/export_go.py: shared reservoir + per-variant weight map.
// No subprocess, no Python at runtime.
package fly

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"

	"fair_backgammon/game"
)

//go:embed fly.json
var rawJSON []byte

// Head is one frozen reservoir plus a weight vector per variant.
type Head struct {
	N, Dim, Steps   int
	Indptr, Indices []int
	Data            []float64
	Win             [][]float64
	W               map[string][]float64
}

// Load parses the embedded export. Fails on any shape mismatch.
func Load() (*Head, error) {
	var d struct {
		N       int                  `json:"n"`
		Dim     int                  `json:"dim"`
		Steps   int                  `json:"steps"`
		Indptr  []int                `json:"indptr"`
		Indices []int                `json:"indices"`
		Data    []float64            `json:"data"`
		Win     [][]float64          `json:"win"`
		Weights map[string][]float64 `json:"weights"`
	}
	if err := json.Unmarshal(rawJSON, &d); err != nil {
		return nil, err
	}
	h := &Head{N: d.N, Dim: d.Dim, Steps: d.Steps, Indptr: d.Indptr,
		Indices: d.Indices, Data: d.Data, Win: d.Win, W: d.Weights}
	if len(h.Indptr) != h.N+1 || h.Indptr[h.N] != len(h.Data) || len(h.Indices) != len(h.Data) {
		return nil, fmt.Errorf("fly: bad CSR n=%d nnz=%d", h.N, len(h.Data))
	}
	if len(h.Win) != h.N {
		return nil, fmt.Errorf("fly: bad Win rows %d", len(h.Win))
	}
	for _, row := range h.Win {
		if len(row) != h.Dim {
			return nil, fmt.Errorf("fly: bad Win dim %d", len(row))
		}
	}
	for name, w := range h.W {
		if len(w) != h.N+h.Dim+1 {
			return nil, fmt.Errorf("fly: bad weights %s len %d", name, len(w))
		}
	}
	return h, nil
}

// Pip counts race pips. Port of musca brain/features.py.
func Pip(board [24]int, bar [2]int, p int) int {
	tot := bar[p] * 25
	if p == 0 {
		for i, v := range board {
			if v > 0 {
				tot += v * (i + 1)
			}
		}
	} else {
		for i, v := range board {
			if v < 0 {
				tot += -v * (24 - i)
			}
		}
	}
	return tot
}

// EncodeState builds the 38-dim state vector. Port of musca brain/features.py.
func EncodeState(board [24]int, bar, off [2]int, turn int) []float64 {
	f := make([]float64, 0, 38)
	for _, v := range board {
		f = append(f, float64(v)/5.0)
	}
	f = append(f, float64(bar[0])/2.0, float64(bar[1])/2.0,
		float64(off[0])/15.0, float64(off[1])/15.0)
	if turn == 0 {
		f = append(f, 1.0)
	} else {
		f = append(f, -1.0)
	}
	pw, pb := Pip(board, bar, 0), Pip(board, bar, 1)
	f = append(f, float64(pb-pw)/100.0, float64(pw)/200.0, float64(pb)/200.0)
	for p := 0; p < 2; p++ {
		s := 1
		if p == 1 {
			s = -1
		}
		blots, made, home := 0, 0, 0
		lo, hi := 0, 6
		if p == 1 {
			lo, hi = 18, 24
		}
		for i := lo; i < hi; i++ {
			if board[i]*s > 0 {
				home += board[i] * s
			}
		}
		for _, v := range board {
			if v == s {
				blots++
			} else if v*s >= 2 {
				made++
			}
		}
		f = append(f, float64(blots)/8.0, float64(made)/6.0, float64(home)/15.0)
	}
	return f
}

// Phi runs x=tanh(Wx+Win*u) and returns [x, u, 1]. Port of brain/value.py.
func Phi(h *Head, w []float64, board [24]int, bar, off [2]int, turn int) []float64 {
	u := EncodeState(board, bar, off, turn)
	x := make([]float64, h.N)
	useRes := false
	for _, v := range w[:h.N] {
		if v != 0 {
			useRes = true
			break
		}
	}
	if useRes {
		nxt := make([]float64, h.N)
		for s := 0; s < h.Steps; s++ {
			for i := 0; i < h.N; i++ {
				acc := 0.0
				row := h.Win[i]
				for j, c := range row {
					acc += c * u[j]
				}
				for k := h.Indptr[i]; k < h.Indptr[i+1]; k++ {
					acc += h.Data[k] * x[h.Indices[k]]
				}
				nxt[i] = math.Tanh(acc)
			}
			x, nxt = nxt, x
		}
	}
	return append(append(append([]float64{}, x...), u...), 1.0)
}

// Value is the white-win estimate V(s) = w.phi(s).
func Value(h *Head, w []float64, board [24]int, bar, off [2]int, turn int) float64 {
	phi := Phi(h, w, board, bar, off, turn)
	v := 0.0
	for i, c := range w {
		v += c * phi[i]
	}
	return v
}

// Choose is 1-ply: white max V(s'), black min, ties within 1e-9 random.
func Choose(h *Head, w []float64, board [24]int, bar, off [2]int, turn int, moves []game.Move, rng *rand.Rand) game.Move {
	var best []game.Move
	var bestV float64
	white := turn == 0
	for _, m := range moves {
		g := &game.Game{Board: board, Bar: bar, Off: off,
			Turn: game.Player(turn), MovesLeft: []int{m.Die}, HasRolled: true}
		if err := g.Apply(m); err != nil {
			continue
		}
		v := Value(h, w, g.Board, g.Bar, g.Off, int(g.Turn))
		switch {
		case best == nil,
			white && v > bestV+1e-9,
			!white && v < bestV-1e-9:
			best, bestV = []game.Move{m}, v
		default:
			if math.Abs(v-bestV) <= 1e-9 {
				best = append(best, m)
			}
		}
	}
	return best[rng.Intn(len(best))]
}

// Variants lists available weight names.
func Variants(h *Head) []string {
	out := make([]string, 0, len(h.W))
	for name := range h.W {
		out = append(out, name)
	}
	return out
}
