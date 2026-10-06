// Package engine adapts the vendored gnubg evaluator (engine/gnubg,
// ported from foochu/bgweb-api, MIT) to fair_backgammon's game state and
// exposes Easy/Medium/Hard difficulty presets.
package engine

import (
	"embed"
	"io/fs"
	"fmt"
	"math/rand"
	"sync"

	"fair_backgammon/engine/gnubg"
	"fair_backgammon/game"
)

//go:embed gnubg/data/gnubg.weights gnubg/data/gnubg_os0.bd gnubg/data/gnubg_ts0.bd gnubg/data/met/*
var dataFS embed.FS


var initOnce sync.Once
var initErr error

// Ensure loads weights + bearoff DBs once. Thread-safe; eval itself is
// driven single-threaded (global NN cache is not goroutine-safe).
func Ensure() error {
	initOnce.Do(func() {
		sub, err := fs.Sub(dataFS, "gnubg/data")
		if err != nil {
			initErr = err
			return
		}
		initErr = gnubg.Init(sub)
	})
	return initErr
}

// Difficulty is one rung: search depth + eval noise + pick breadth.
// Plies 0..4, Noise is NN output stddev (0 = exact), TopN picks uniformly
// among the top N scored moves (1 = best).
type Difficulty struct {
	Name  string
	Plies int
	Noise float32
	TopN  int
}

var (
	Easy   = Difficulty{Name: "easy", Plies: 0, Noise: 0, TopN: 5}
	Medium = Difficulty{Name: "medium", Plies: 0, Noise: 0, TopN: 1}
	Hard   = Difficulty{Name: "hard", Plies: 2, Noise: 0, TopN: 1}
)

// Step is one checker move in gnubg 1-based points: "bar" enters, "off" bears off.
type Step struct {
	From string
	To   string
}

// ScoredPlay is a full-turn play with its equity + diff-to-best.
type ScoredPlay struct {
	Steps []Step
	Eq    float32
	Diff  float32
}

// toTan builds halves in mover orientation (mover runs high->low).
// Returns {self, opp}: FindMoves with player=1 seats public[0] as the mover.
func toTan(board [24]int, bar [2]int, turn game.Player) gnubg.TanBoard {
	var self, opp [25]int
	if turn == game.White {
		for i, v := range board {
			if v > 0 {
				self[i] += v
			} else if v < 0 {
				opp[23-i] += -v
			}
		}
		self[24] = bar[0]
		opp[24] = bar[1]
	} else {
		for i, v := range board {
			if v < 0 {
				self[23-i] += -v
			} else if v > 0 {
				opp[i] += v
			}
		}
		self[24] = bar[1]
		opp[24] = bar[0]
	}
	return gnubg.TanBoard{self, opp}
}

// Plays returns ranked full-turn plays for the position + dice.
func Plays(board [24]int, bar [2]int, turn game.Player, dice [2]int, d Difficulty) ([]ScoredPlay, error) {
	if err := Ensure(); err != nil {
		return nil, err
	}
	tan := toTan(board, bar, turn)
	ml, err := gnubg.FindMovesEx(tan, dice, 1, true, false, d.Plies, d.Noise)
	if err != nil {
		return nil, err
	}
	out := make([]ScoredPlay, 0, ml.GetMovesNum())
	for i := 0; i < ml.GetMovesNum(); i++ {
		m := ml.GetMove(i)
		steps := make([]Step, 0, m.GetPlaysNum())
		for j := 0; j < m.GetPlaysNum(); j++ {
			p := m.GetPlay(j)
			steps = append(steps, Step{From: itos(p[0]), To: itos(p[1])})
		}
		out = append(out, ScoredPlay{Steps: steps, Eq: m.GetEquity(), Diff: diffOf(ml, i)})
	}
	return out, nil
}

func diffOf(ml gnubg.MoveList, i int) float32 {
	if ml.GetMovesNum() == 0 {
		return 0
	}
	return ml.GetMove(i).GetEquity() - ml.GetMove(0).GetEquity()
}

func itos(internal int) string {
	if internal == 24 {
		return "bar"
	}
	if internal == -1 {
		return "off"
	}
	return fmt.Sprintf("%d", internal+1)
}

// ToGameMoves maps one play's steps onto game.Move with dice assigned by
// backtracking over the remaining dice (exact distance first).
// Returns moves in play order; each applies legally in sequence.
func ToGameMoves(g *game.Game, steps []Step) ([]game.Move, error) {
	turn := g.Turn
	conv := func(s Step) (from, to int, err error) {
		from, err = parseFrom(s.From, turn)
		if err != nil {
			return
		}
		to, err = parseTo(s.To, turn)
		return
	}
	type pair struct{ from, to int }
	pairs := make([]pair, len(steps))
	for i, s := range steps {
		f, t, err := conv(s)
		if err != nil {
			return nil, err
		}
		pairs[i] = pair{f, t}
	}
	dice := append([]int(nil), g.MovesLeft...)
	assign := make([]int, len(pairs))
	used := make([]bool, len(dice))
	// ponytail: <=4 dice, try exact distances before overshoot bear-offs
	order := func() []int {
		idx := make([]int, len(dice))
		for i := range idx {
			idx[i] = i
		}
		return idx
	}
	var rec func(step int, cl *game.Game) bool
	rec = func(step int, cl *game.Game) bool {
		if step == len(pairs) {
			return true
		}
		// exact-first: two passes
		for pass := 0; pass < 2; pass++ {
			for _, di := range order() {
				if used[di] {
					continue
				}
				m := game.Move{From: pairs[step].from, To: pairs[step].to, Die: dice[di]}
				exact := isExact(cl, m)
				if (pass == 0) != exact {
					continue
				}
				nx := cl.Clone()
				if nx.Apply(m) != nil {
					continue
				}
				used[di] = true
				assign[step] = di
				if rec(step+1, nx) {
					return true
				}
				used[di] = false
			}
		}
		return false
	}
	if !rec(0, g.Clone()) {
		return nil, fmt.Errorf("engine: cannot map play %v to dice %v", steps, dice)
	}
	out := make([]game.Move, len(pairs))
	for i, p := range pairs {
		out[i] = game.Move{From: p.from, To: p.to, Die: dice[assign[i]]}
	}
	return out, nil
}

func isExact(g *game.Game, m game.Move) bool {
	if m.To == game.OffPos {
		var dist int
		if g.Turn == game.White {
			dist = m.From + 1
		} else {
			dist = 24 - m.From
		}
		return m.Die == dist
	}
	if m.From == game.BarPos {
		return true // entry distance fixed by die
	}
	var dist int
	if g.Turn == game.White {
		dist = m.From - m.To
	} else {
		dist = m.To - m.From
	}
	return dist == m.Die
}

func parseFrom(s string, turn game.Player) (int, error) {
	if s == "bar" {
		return game.BarPos, nil
	}
	var pt int
	if _, err := fmt.Sscanf(s, "%d", &pt); err != nil || pt < 1 || pt > 24 {
		return 0, fmt.Errorf("bad from %q", s)
	}
	if turn == game.White {
		return pt - 1, nil
	}
	return 24 - pt, nil
}

func parseTo(s string, turn game.Player) (int, error) {
	if s == "off" {
		return game.OffPos, nil
	}
	var pt int
	if _, err := fmt.Sscanf(s, "%d", &pt); err != nil || pt < 1 || pt > 24 {
		return 0, fmt.Errorf("bad to %q", s)
	}
	if turn == game.White {
		return pt - 1, nil
	}
	return 24 - pt, nil
}

// PickPlay selects among ranked plays: uniform over topN (clamped).
func PickPlay(plays []ScoredPlay, topN int, rng *rand.Rand) ScoredPlay {
	n := topN
	if n < 1 {
		n = 1
	}
	if n > len(plays) {
		n = len(plays)
	}
	return plays[rng.Intn(n)]
}
