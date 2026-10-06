package engine

import (
	"fmt"
	"testing"
	"time"

	"fair_backgammon/game"
)

func openBoard() ([24]int, [2]int) {
	var b [24]int
	b[23], b[12], b[7], b[5] = 2, 5, 3, 5
	b[0], b[11], b[16], b[18] = -2, -5, -3, -5
	return b, [2]int{}
}

func TestProbeOpening31(t *testing.T) {
	b, bar := openBoard()
	start := time.Now()
	plays, err := Plays(b, bar, game.White, [2]int{3, 1}, Medium)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("took %v, %d plays", time.Since(start), len(plays))
	for i := 0; i < 3 && i < len(plays); i++ {
		t.Logf("play %d eq=%.3f diff=%.3f %v", i, plays[i].Eq, plays[i].Diff, plays[i].Steps)
	}
	best := plays[0].Steps
	want := fmt.Sprintf("%v", []Step{{"8", "5"}, {"6", "5"}})
	if fmt.Sprintf("%v", best) != want {
		t.Fatalf("orientation? got %v want %s", best, want)
	}
	// mapping onto game moves must apply cleanly
	g := &game.Game{Board: b, Bar: bar, Off: [2]int{}, Turn: game.White,
		MovesLeft: []int{3, 1}, HasRolled: true}
	mv, err := ToGameMoves(g, best)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range mv {
		if err := g.Apply(m); err != nil {
			t.Fatalf("apply %v: %v", m, err)
		}
	}
}

func TestProbeLatency(t *testing.T) {
	b, bar := openBoard()
	for _, d := range []Difficulty{
		{Name: "x", Plies: 0},
		{Name: "x", Plies: 1},
		{Name: "x", Plies: 2},
	} {
		start := time.Now()
		plays, err := Plays(b, bar, game.White, [2]int{3, 1}, d)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("plies=%d took %v top=%v", d.Plies, time.Since(start), plays[0].Steps)
	}
}
