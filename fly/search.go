// 2-ply expectimax over the value head. Port of musca brain/search.py.
//
// For each own candidate: average V over all 21 opp dice rolls, the opponent
// replying 1-ply. Turn is treated as passing after the candidate.
package fly

import (
	"math"
	"math/rand"

	"fair_backgammon/game"
)

type diceCombo struct {
	d1, d2 int
	prob   float64
	moves  []int
}

var diceTable []diceCombo

func init() {
	for d1 := 1; d1 <= 6; d1++ {
		for d2 := d1; d2 <= 6; d2++ {
			if d1 == d2 {
				diceTable = append(diceTable, diceCombo{d1, d2, 1.0 / 36.0, []int{d1, d1, d1, d1}})
			} else {
				diceTable = append(diceTable, diceCombo{d1, d2, 2.0 / 36.0, []int{d1, d2}})
			}
		}
	}
}

// ExpValue is the expected V after the opponent's best reply.
func ExpValue(h *Head, w []float64, board [24]int, bar, off [2]int, mover int) float64 {
	opp := 1 - mover
	whiteOpp := opp == 0
	tot := 0.0
	for _, dc := range diceTable {
		g := &game.Game{Board: board, Bar: bar, Off: off,
			Turn: game.Player(opp), MovesLeft: append([]int{}, dc.moves...), HasRolled: true}
		replies := g.LegalMoves()
		if len(replies) == 0 {
			tot += dc.prob * Value(h, w, board, bar, off, opp)
			continue
		}
		best, first := 0.0, true
		for _, r := range replies {
			c := &game.Game{Board: board, Bar: bar, Off: off,
				Turn: game.Player(opp), MovesLeft: append([]int{}, dc.moves...), HasRolled: true}
			if err := c.Apply(r); err != nil {
				continue
			}
			v := Value(h, w, c.Board, c.Bar, c.Off, opp)
			if first || (whiteOpp && v > best+1e-9) || (!whiteOpp && v < best-1e-9) {
				best, first = v, false
			}
		}
		tot += dc.prob * best
	}
	return tot
}

// Expectimax is 2-ply pick. White max, black min, ties within 1e-9 random.
func Expectimax(h *Head, w []float64, board [24]int, bar, off [2]int, turn int, moves []game.Move, rng *rand.Rand) game.Move {
	var best []game.Move
	var bestV float64
	white := turn == 0
	for _, m := range moves {
		g := &game.Game{Board: board, Bar: bar, Off: off,
			Turn: game.Player(turn), MovesLeft: []int{m.Die}, HasRolled: true}
		if err := g.Apply(m); err != nil {
			continue
		}
		v := ExpValue(h, w, g.Board, g.Bar, g.Off, turn)
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
