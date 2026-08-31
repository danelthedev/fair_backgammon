package game

import (
	"fmt"
	"math/rand"
)

// Player 0=white moves ->0, 1=black moves ->23
type Player int

const (
	White Player = 0
	Black Player = 1
	BarPos       = -1
	OffPos       = -2
)

type Move struct {
	From int `json:"from"` // -1=bar, 0-23 board
	To   int `json:"to"`   // -2=off, 0-23 board
	Die  int `json:"die"`
}

type Game struct {
	Board     [24]int `json:"board"` // +white -black
	Bar       [2]int  `json:"bar"`
	Off       [2]int  `json:"off"`
	Turn      Player  `json:"turn"`
	Dice      [2]int  `json:"dice"`
	MovesLeft []int   `json:"movesLeft"`
	HasRolled bool    `json:"hasRolled"`
}

func NewGame() *Game {
	g := &Game{Turn: White}
	// white
	g.Board[23] = 2
	g.Board[12] = 5
	g.Board[7] = 3
	g.Board[5] = 5
	// black (negative)
	g.Board[0] = -2
	g.Board[11] = -5
	g.Board[16] = -3
	g.Board[18] = -5
	return g
}

func (g *Game) Clone() *Game { c := *g; c.MovesLeft = append([]int(nil), g.MovesLeft...); return &c }

func (g *Game) Roll() {
	d1 := rand.Intn(6) + 1
	d2 := rand.Intn(6) + 1
	g.Dice = [2]int{d1, d2}
	if d1 == d2 {
		g.MovesLeft = []int{d1, d1, d1, d1}
	} else {
		g.MovesLeft = []int{d1, d2}
	}
	g.HasRolled = true
}

// direction
func dir(p Player) int {
	if p == White {
		return -1
	}
	return 1
}

func (g *Game) allInHome(p Player) bool {
	if g.Bar[p] > 0 {
		return false
	}
	if p == White {
		for i := 6; i < 24; i++ {
			if p == White && g.Board[i] > 0 {
				return false
			}
			if p == Black && g.Board[i] < 0 {
				// black home is 18-23, so check 0-17
				// actually for black we check different range below
			}
		}
		// white home 0-5
		return true
	}
	// black home 18-23
	for i := 0; i < 18; i++ {
		if g.Board[i] < 0 {
			return false
		}
	}
	return true
}

func (g *Game) isBlocked(to int, p Player) bool {
	if to < 0 || to >= 24 {
		return false
	}
	v := g.Board[to]
	if p == White {
		return v <= -2 // 2+ black block white
	}
	return v >= 2
}

func (g *Game) IsLegal(m Move) (bool, string) {
	if !g.HasRolled {
		return false, "must roll first"
	}
	// check die available
	found := false
	for _, d := range g.MovesLeft {
		if d == m.Die {
			found = true
			break
		}
	}
	if !found {
		return false, fmt.Sprintf("die %d not available %v", m.Die, g.MovesLeft)
	}
	p := g.Turn
	// bar rule
	if g.Bar[p] > 0 && m.From != BarPos {
		return false, "must enter from bar"
	}
	if g.Bar[p] == 0 && m.From == BarPos {
		return false, "bar empty"
	}
	// from ownership
	if m.From != BarPos {
		if m.From < 0 || m.From >= 24 {
			return false, "from out of range"
		}
		v := g.Board[m.From]
		if p == White && v <= 0 {
			return false, "no white checker there"
		}
		if p == Black && v >= 0 {
			return false, "no black checker there"
		}
	}
	// to validation
	if m.To == OffPos {
		if !g.allInHome(p) {
			return false, "not all in home"
		}
		if m.From == BarPos {
			return false, "cannot bear off from bar"
		}
		// exact or overshoot logic (simplified: allow overshoot if no checker beyond)
		dist := 0
		if p == White {
			dist = m.From + 1 // from 0 ->1 away, 5->6 away
		} else {
			dist = 24 - m.From
		}
		if m.Die < dist {
			return false, "die too small to bear off"
		}
		if m.Die > dist {
			// only allow if no checker further away (higher dist)
			if p == White {
				for i := m.From + 1; i < 6; i++ {
					if g.Board[i] > 0 {
						return false, "must bear off furthest"
					}
				}
				for i := 6; i < 24; i++ {
					if g.Board[i] > 0 {
						return false, "not all home" // redundant
					}
				}
			} else {
				for i := 18; i < m.From; i++ {
					if g.Board[i] < 0 {
						return false, "must bear off furthest"
					}
				}
			}
		}
		return true, ""
	}
	if m.To < 0 || m.To >= 24 {
		return false, "to out of range"
	}
	// distance must match die
	expected := 0
	if m.From == BarPos {
		if p == White {
			// white enters at 23..18 (die 1 ->23, 6->18)
			expected = 24 - m.To // not used, compute entry point
			// entry point for white: 24-die
			entry := 24 - m.Die
			if m.To != entry {
				return false, "bar entry mismatch"
			}
		} else {
			entry := m.Die - 1
			if m.To != entry {
				return false, "bar entry mismatch"
			}
		}
	} else {
		expected = m.From - m.To
		if p == Black {
			expected = m.To - m.From
		}
		if expected != m.Die {
			return false, "distance != die"
		}
	}
	if g.isBlocked(m.To, p) {
		return false, "point blocked"
	}
	return true, ""
}

func (g *Game) Apply(m Move) error {
	ok, msg := g.IsLegal(m)
	if !ok {
		return fmt.Errorf("illegal: %s", msg)
	}
	p := g.Turn
	// consume die
	idx := -1
	for i, d := range g.MovesLeft {
		if d == m.Die {
			idx = i
			break
		}
	}
	g.MovesLeft = append(g.MovesLeft[:idx], g.MovesLeft[idx+1:]...)

	// remove from
	if m.From == BarPos {
		g.Bar[p]--
	} else {
		if p == White {
			g.Board[m.From]--
		} else {
			g.Board[m.From]++
		}
	}
	// add to / hit / bear off
	if m.To == OffPos {
		g.Off[p]++
	} else {
		// hit?
		v := g.Board[m.To]
		if p == White && v == -1 {
			g.Board[m.To] = 0
			g.Bar[Black]++
		} else if p == Black && v == 1 {
			g.Board[m.To] = 0
			g.Bar[White]++
		}
		if p == White {
			g.Board[m.To]++
		} else {
			g.Board[m.To]--
		}
	}
	// if moves exhausted, switch turn (caller handles via CanMove check)
	return nil
}

func (g *Game) HasAnyLegal() bool {
	if !g.HasRolled || len(g.MovesLeft) == 0 {
		return false
	}
	for _, d := range g.MovesLeft {
		// try all from
		candidates := []int{BarPos}
		for i := 0; i < 24; i++ {
			candidates = append(candidates, i)
		}
		for _, from := range candidates {
			for to := -2; to < 24; to++ {
				// to -2 = off, skip invalid
				if to == -1 {
					continue
				}
				m := Move{From: from, To: to, Die: d}
				if ok, _ := g.IsLegal(m); ok {
					return true
				}
			}
		}
	}
	return false
}

func (g *Game) CheckWin() (bool, Player) {
	if g.Off[White] >= 15 {
		return true, White
	}
	if g.Off[Black] >= 15 {
		return true, Black
	}
	return false, White
}
