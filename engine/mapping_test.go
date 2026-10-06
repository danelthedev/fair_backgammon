package engine

import (
	"math/rand"
	"testing"

	"fair_backgammon/game"
)

// Top-5 plays map onto legal game moves from real (asymmetric) positions, both colors.
func TestTopNMapping(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	fails, total := 0, 0
	for i := 0; i < 200; i++ {
		g := game.NewGame()
		for h := 0; h < rng.Intn(40); h++ {
			g.Roll()
			for len(g.MovesLeft) > 0 && g.HasAnyLegal() {
				lm := g.LegalMoves()
				_ = g.Apply(lm[rng.Intn(len(lm))])
			}
			if win, _ := g.CheckWin(); win {
				break
			}
			g.MovesLeft = nil
			g.HasRolled = false
			g.SetTurn(1 - g.Turn)
		}
		if win, _ := g.CheckWin(); win {
			continue
		}
		g.Roll()
		if !g.HasAnyLegal() {
			continue
		}
		plays, err := Plays(g.Board, g.Bar, g.Turn, g.Dice, Easy)
		if err != nil || len(plays) == 0 {
			continue
		}
		top := 5
		if len(plays) < top {
			top = len(plays)
		}
		for k := 0; k < top; k++ {
			total++
			if _, err := ToGameMoves(g.Clone(), plays[k].Steps); err != nil {
				fails++
				if fails <= 4 {
					t.Logf("i=%d turn=%d dice=%v left=%v\n board=%v bar=%v\n play#%d eq=%.3f steps=%v",
						i, g.Turn, g.Dice, g.MovesLeft, g.Board, g.Bar, k, plays[k].Eq, plays[k].Steps)
				}
			}
		}
	}
	t.Logf("fails=%d/%d", fails, total)
}
