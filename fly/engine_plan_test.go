package fly

import (
	"math/rand"
	"testing"

	"fair_backgammon/engine"
	"fair_backgammon/game"
)

// planEngineTurn must return moves that apply legally on the same state,
// for both colors and edge phases (doubles, bar, bearoff).
func TestPlanEngineTurn(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	cases := map[string]func() *game.Game{
		"white opening 3-1": func() *game.Game {
			g := game.NewGame()
			g.Dice, g.MovesLeft, g.HasRolled = [2]int{3, 1}, []int{3, 1}, true
			return g
		},
		"black doubles": func() *game.Game {
			g := game.NewGame()
			g.Turn = game.Black
			g.Dice, g.MovesLeft, g.HasRolled = [2]int{6, 6}, []int{6, 6, 6, 6}, true
			return g
		},
		"white on bar": func() *game.Game {
			g := game.NewGame()
			g.Board[23] = 0
			g.Bar[0] = 1
			g.Dice, g.MovesLeft, g.HasRolled = [2]int{4, 2}, []int{4, 2}, true
			return g
		},
		"white bearoff": func() *game.Game {
			g := &game.Game{Turn: game.White, HasRolled: true}
			g.Board[5], g.Board[0] = 8, 2
			g.Off[0] = 5
			g.Dice, g.MovesLeft = [2]int{6, 3}, []int{6, 3}
			return g
		},
	}
	for name, mk := range cases {
		g := mk()
		st := stateMsg{Board: g.Board, Bar: g.Bar, Off: g.Off, Turn: int(g.Turn),
			MovesLeft: append([]int(nil), g.MovesLeft...), HasRolled: true, LegalMoves: g.LegalMoves()}
		if len(st.LegalMoves) == 0 {
			t.Fatalf("%s: no legal moves, bad fixture", name)
		}
		for _, diff := range []engine.Difficulty{engine.Easy, engine.Medium, engine.Hard} {
			mv := planEngineTurn(st, diff, rng)
			if len(mv) == 0 {
				t.Fatalf("%s %s: empty plan", name, diff.Name)
			}
			cl := &game.Game{Board: g.Board, Bar: g.Bar, Off: g.Off, Turn: g.Turn,
				Dice: g.Dice, MovesLeft: append([]int(nil), g.MovesLeft...), HasRolled: true}
			for _, m := range mv {
				if err := cl.Apply(m); err != nil {
					t.Fatalf("%s %s: apply %v: %v", name, diff.Name, m, err)
				}
			}
		}
	}
}
