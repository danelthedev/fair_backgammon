package engine_test

// Headless h2h: standard games, no network. Fly picks die-by-die (like the
// live bot); engine sides plan the full turn once per roll.

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"fair_backgammon/engine"
	"fair_backgammon/fly"
	"fair_backgammon/game"
)

type botFn func(g *game.Game, rng *rand.Rand) error

func flyBot(h *fly.Head, w []float64) botFn {
	return func(g *game.Game, rng *rand.Rand) error {
		for len(g.MovesLeft) > 0 && g.HasAnyLegal() {
			m := fly.Choose(h, w, g.Board, g.Bar, g.Off, int(g.Turn), g.LegalMoves(), rng)
			if err := g.Apply(m); err != nil {
				return err
			}
		}
		return nil
	}
}

var engFallbacks int
func engBot(d engine.Difficulty) botFn {
	return func(g *game.Game, rng *rand.Rand) error {
		if len(g.MovesLeft) == 0 || !g.HasAnyLegal() {
			return nil
		}
		plays, err := engine.Plays(g.Board, g.Bar, g.Turn, g.Dice, d)
		if err != nil {
			return err
		}
		if len(plays) == 0 {
			return nil
		}
		pick := engine.PickPlay(plays, d.TopN, rng)
		mv, err := engine.ToGameMoves(g, pick.Steps)
		if err != nil {
			engFallbacks++
			return randomBot(g, rng) // mapping failed: salvage turn randomly
		}
		if len(mv) == 0 {
		}
		for _, m := range mv {
			if err := g.Apply(m); err != nil {
				return fmt.Errorf("%s apply %v: %w", d.Name, m, err)
			}
		}
		// ponytail: mapped play should exhaust dice; leftover = random
		for len(g.MovesLeft) > 0 && g.HasAnyLegal() {
			engFallbacks++
			if err := randomSingle(g, rng); err != nil {
				return err
			}
		}
		return nil
	}
}

func randomSingle(g *game.Game, rng *rand.Rand) error {
	lm := g.LegalMoves()
	if len(lm) == 0 {
		return nil
	}
	return g.Apply(lm[rng.Intn(len(lm))])
}

func randomBot(g *game.Game, rng *rand.Rand) error {
	for len(g.MovesLeft) > 0 && g.HasAnyLegal() {
		if err := randomSingle(g, rng); err != nil {
			return err
		}
	}
	return nil
}

// playGame returns 0/1 winner, -1 draw (cap).
func playGame(white, black botFn, seed int64) int {
	rng := rand.New(rand.NewSource(seed))
	g := game.NewGame()
	g.Turn = game.Player(rand.New(rand.NewSource(seed ^ 0x9e37)).Intn(2)) // random starter
	for half := 0; half < 2000; half++ {
		if traceT != nil && half%50 == 0 {
			traceT.Logf("half=%d turn=%d off=%v bar=%v", half, g.Turn, g.Off, g.Bar)
		}
		g.Roll()
		var err error
		if !g.HasAnyLegal() {
			// pass
		} else if g.Turn == game.White {
			err = white(g, rng)
		} else {
			err = black(g, rng)
		}
		if err != nil {
			panic(fmt.Sprintf("seed %d half %d: %v", seed, half, err))
		}
		if win, w := g.CheckWin(); win {
			return int(w)
		}
		g.MovesLeft = nil
		g.HasRolled = false
		g.SetTurn(1 - g.Turn)
	}
	return -1
}
var traceT *testing.T

func playGameCounted(white, black botFn, seed int64, t *testing.T) int {
	traceT = t
	defer func() { traceT = nil }()
	return playGame(white, black, seed)
}

func TestH2H(t *testing.T) {
	if testing.Short() {
		t.Skip("h2h needs full runs")
	}
	heads, err := fly.Load()
	if err != nil {
		t.Fatal(err)
	}
	h := heads["retarded"]
	fw := h.W["retarded"]
	bots := map[string]botFn{
		"fly":    flyBot(h, fw),
		"easy":   engBot(engine.Easy),
		"medium": engBot(engine.Medium),
		"hard":   engBot(engine.Hard),
	}
	allPairs := [][2]string{
		{"fly", "easy"}, {"fly", "medium"}, {"fly", "hard"},
		{"easy", "medium"}, {"easy", "hard"}, {"medium", "hard"},
	}
	// ponytail: one pair per process (global NN cache is not goroutine-safe).
	// H2H_PAIR=a:b selects, H2H_N sets games.
	pairs := allPairs
	if sel := os.Getenv("H2H_PAIR"); sel != "" {
		pairs = nil
		if ab := strings.SplitN(sel, ":", 2); len(ab) == 2 {
			pairs = [][2]string{{ab[0], ab[1]}}
		}
	}
	n := 30
	if v := os.Getenv("H2H_N"); v != "" {
		fmt.Sscanf(v, "%d", &n)
	}
	t.Logf("%-12s %4s %4s %5s %8s %8s", "pair(a-b)", "a", "b", "draw", "time", "fallback")
	for _, p := range pairs {
		a, b := p[0], p[1]
		aw, bw, draws := 0, 0, 0
		start := time.Now()
		fb0 := engFallbacks
		for i := 0; i < n; i++ {
			seed := int64(1000 + i)
			var winner string // bot name
			if i%2 == 0 {
				// a=white, b=black
				switch playGame(bots[a], bots[b], seed) {
				case 0:
					winner = a
				case 1:
					winner = b
				}
			} else {
				switch playGame(bots[b], bots[a], seed) {
				case 0:
					winner = b
				case 1:
					winner = a
				}
			}
			switch winner {
			case a:
				aw++
			case b:
				bw++
			default:
				draws++
			}
		}
		t.Logf("%-12s %4d %4d %5d %8v %8d", a+"-"+b, aw, bw, draws, time.Since(start).Round(time.Second), engFallbacks-fb0)
	}
}
