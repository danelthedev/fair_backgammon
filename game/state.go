package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
)

// Player 0=white moves ->0, 1=black moves ->23
type Player int

const (
	White  Player = 0
	Black  Player = 1
	BarPos        = -1
	OffPos        = -2
)

type Move struct {
	From int `json:"from"` // -1=bar, 0-23 board
	To   int `json:"to"`   // -2=off, 0-23 board
	Die  int `json:"die"`
}

type Mods struct {
	Negative bool        `json:"negative"` // dice may roll negative, moving pieces backwards
	MaxDie   int         `json:"maxDie"`   // >6 raises max dice value, 0/<=6 = standard d6
	Powers   PowerConfig `json:"powers"`   // uses per game per power-up, 0 = disabled
}

// PowerConfig enables power-ups: reroll a roll, skip a turn, shield blots
// from capture during the opponent's next turn.
type PowerConfig struct {
	Reroll  int `json:"reroll"`
	Skip    int `json:"skip"`
	Protect int `json:"protect"`
}

type Game struct {
	Board     [24]int `json:"board"` // +white -black
	Bar       [2]int  `json:"bar"`
	Off       [2]int  `json:"off"`
	Turn      Player  `json:"turn"`
	Dice      [2]int  `json:"dice"`
	MovesLeft []int   `json:"movesLeft"`
	HasRolled bool    `json:"hasRolled"`
	Mods      Mods    `json:"mods"`
	PowerLeft [2]PowerConfig `json:"powerLeft"`
	Shield    [2]bool        `json:"shield"`
	Dealt     int            `json:"dealt"`    // dice dealt by current roll
	Rerolled  bool           `json:"rerolled"` // roll already rerolled once
}

func NewGame() *Game { return NewGameWithMods(Mods{}) }

// ponytail: power counts clamp 0..10, lobby UI defaults to 3 when enabled
func clampPower(n int) int {
	if n < 0 {
		return 0
	}
	if n > 10 {
		return 10
	}
	return n
}

// MaxDie reports the highest die face: standard 6 unless mods raise it.
func (g *Game) MaxDie() int {
	if g.Mods.MaxDie > 6 {
		if g.Mods.MaxDie > 20 {
			return 20
		}
		return g.Mods.MaxDie
	}
	return 6
}

func NewGameWithMods(m Mods) *Game {
	if m.MaxDie < 0 {
		m.MaxDie = 0
	}
	if m.MaxDie > 20 {
		m.MaxDie = 20
	}
	m.Powers.Reroll = clampPower(m.Powers.Reroll)
	m.Powers.Skip = clampPower(m.Powers.Skip)
	m.Powers.Protect = clampPower(m.Powers.Protect)
	// ponytail: try board config file, fallback to standard
	if g := tryLoadBoard(); g != nil {
		g.Mods = m
		g.PowerLeft = [2]PowerConfig{m.Powers, m.Powers}
		return g
	}
	g := &Game{Turn: White, Mods: m}
	g.PowerLeft = [2]PowerConfig{m.Powers, m.Powers}
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

// tryLoadBoard tries BOARD_CONFIG env, ./board.json, ./game/board.json — ponytail: file is simplest config
func tryLoadBoard() *Game {
	paths := []string{}
	if p := os.Getenv("BOARD_CONFIG"); p != "" {
		paths = append(paths, p)
	}
	paths = append(paths, "board.json", "game/board.json", "config/board.json")
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		// try object {board,bar,off,turn}
		var cfg struct {
			Board *[24]int `json:"board"`
			Bar   *[2]int  `json:"bar"`
			Off   *[2]int  `json:"off"`
			Turn  *Player  `json:"turn"`
		}
		if err := json.Unmarshal(b, &cfg); err == nil && cfg.Board != nil {
			g := &Game{Turn: White}
			g.Board = *cfg.Board
			if cfg.Bar != nil {
				g.Bar = *cfg.Bar
			}
			if cfg.Off != nil {
				g.Off = *cfg.Off
			}
			if cfg.Turn != nil {
				g.Turn = *cfg.Turn
			}
			return g
		}
		// try bare [24]int
		var arr [24]int
		if err := json.Unmarshal(b, &arr); err == nil {
			g := &Game{Turn: White, Board: arr}
			return g
		}
	}
	return nil
}

func (g *Game) Clone() *Game { c := *g; c.MovesLeft = append([]int(nil), g.MovesLeft...); return &c }

func (g *Game) Roll() {
	max := g.MaxDie()
	roll := func() int {
		d := rand.Intn(max) + 1
		// ponytail: negative mod flips sign 50/50, distance check handles backwards
		if g.Mods.Negative && rand.Intn(2) == 0 {
			d = -d
		}
		return d
	}
	d1, d2 := roll(), roll()
	g.Dice = [2]int{d1, d2}
	if d1 == d2 {
		g.MovesLeft = []int{d1, d1, d1, d1}
	} else {
		g.MovesLeft = []int{d1, d2}
	}
	g.HasRolled = true
	g.Dealt = len(g.MovesLeft)
	g.Rerolled = false
}

// SetTurn passes the turn, expiring the incoming player's shield
// (it protected them through the turn that just ended).
func (g *Game) SetTurn(p Player) {
	g.Turn = p
	g.Shield[p] = false
}

// UseReroll deals fresh dice: once per roll, before any die is spent.
func (g *Game) UseReroll(p Player) error {
	if g.Turn != p {
		return fmt.Errorf("not your turn")
	}
	if !g.HasRolled {
		return fmt.Errorf("roll first")
	}
	if g.Rerolled {
		return fmt.Errorf("already rerolled this roll")
	}
	if len(g.MovesLeft) != g.Dealt {
		return fmt.Errorf("too late, die already spent")
	}
	if g.PowerLeft[p].Reroll <= 0 {
		return fmt.Errorf("no rerolls left")
	}
	g.PowerLeft[p].Reroll--
	g.Roll()
	g.Rerolled = true
	return nil
}

// UseSkip forfeits the turn (rolled or not).
func (g *Game) UseSkip(p Player) error {
	if g.Turn != p {
		return fmt.Errorf("not your turn")
	}
	if g.PowerLeft[p].Skip <= 0 {
		return fmt.Errorf("no skips left")
	}
	g.PowerLeft[p].Skip--
	g.MovesLeft = nil
	g.HasRolled = false
	g.SetTurn(1 - g.Turn)
	return nil
}

// UseProtect shields p's blots: they count as made points next turn.
func (g *Game) UseProtect(p Player) error {
	if g.Turn != p {
		return fmt.Errorf("not your turn")
	}
	if g.Shield[p] {
		return fmt.Errorf("already protected")
	}
	if g.PowerLeft[p].Protect <= 0 {
		return fmt.Errorf("no protection left")
	}
	g.PowerLeft[p].Protect--
	g.Shield[p] = true
	return nil
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
		if v == -1 && g.Shield[Black] {
			return true
		}
		return v <= -2 // 2+ black block white
	}
	if v == 1 && g.Shield[White] {
		return true
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
	// ponytail: negative dice move backwards, never bear off
	if m.To == OffPos {
		if m.Die <= 0 {
			return false, "negative die cannot bear off"
		}
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
		if p == White && v == -1 && !g.Shield[Black] {
			g.Board[m.To] = 0
			g.Bar[Black]++
		} else if p == Black && v == 1 && !g.Shield[White] {
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

// LegalMoves lists all legal (from,to,die) triples. Additive read-only helper
// for bots; unused by existing flows. ponytail: same loops as HasAnyLegal.
func (g *Game) LegalMoves() []Move {
	out := []Move{}
	if !g.HasRolled || len(g.MovesLeft) == 0 {
		return out
	}
	seen := map[Move]bool{}
	for _, d := range g.MovesLeft {
		for from := BarPos; from < 24; from++ {
			if from == -1 && g.Bar[g.Turn] == 0 {
				continue
			}
			for to := -2; to < 24; to++ {
				if to == -1 {
					continue
				}
				m := Move{From: from, To: to, Die: d}
				if ok, _ := g.IsLegal(m); ok && !seen[m] {
					seen[m] = true
					out = append(out, m)
				}
			}
		}
	}
	return out
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

// WinMultiplier reports standard backgammon payout for winner w:
// 3 = backgammon (loser bore off none and has man on bar or in winner home),
// 2 = gammon (loser bore off none), else 1.
func (g *Game) WinMultiplier(w Player) int {
	l := 1 - w
	if g.Off[l] > 0 {
		return 1
	}
	if g.Bar[l] > 0 {
		return 3
	}
	if w == White {
		for i := 0; i < 6; i++ {
			if g.Board[i] < 0 {
				return 3
			}
		}
	} else {
		for i := 18; i < 24; i++ {
			if g.Board[i] > 0 {
				return 3
			}
		}
	}
	return 2
}

// WinReason labels WinMultiplier: "" single, "gammon", "backgammon".
func (g *Game) WinReason(w Player) string {
	switch g.WinMultiplier(w) {
	case 3:
		return "backgammon"
	case 2:
		return "gammon"
	default:
		return ""
	}
}

// CheckTechnicalWin reports marț tehnic for p: home holding exactly six
// columns of 2 (3 borne off) or six columns of 1 (9 borne off), nothing
// elsewhere. Instant win worth double.
func (g *Game) CheckTechnicalWin(p Player) (bool, int) {
	if g.Bar[p] > 0 {
		return false, 0
	}
	if p == White {
		home := 0
		for i := 0; i < 24; i++ {
			if v := g.Board[i]; v > 0 {
				if i >= 6 {
					return false, 0
				}
				home += v
			}
		}
		if g.Off[White]+home != 15 || (home != 12 && home != 6) {
			return false, 0
		}
		want := home / 6
		for i := 0; i < 6; i++ {
			if g.Board[i] != want {
				return false, 0
			}
		}
		return true, 2
	}
	home := 0
	for i := 0; i < 24; i++ {
		if v := g.Board[i]; v < 0 {
			if i < 18 {
				return false, 0
			}
			home -= v
		}
	}
	if g.Off[Black]+home != 15 || (home != 12 && home != 6) {
		return false, 0
	}
	want := -(home / 6)
	for i := 18; i < 24; i++ {
		if g.Board[i] != want {
			return false, 0
		}
	}
	return true, 2
}
