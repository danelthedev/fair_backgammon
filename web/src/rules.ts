// ponytail: pure funcs, same as game.IsLegal — testable
export function allInHome(board: number[], bar: number[], p: number) {
  if (bar[p] > 0) return false
  if (p === 0) return board.slice(6).every(v => v <= 0)
  return board.slice(0, 18).every(v => v >= 0)
}
export function isBlocked(to: number, p: number, board: number[], shield?: boolean[] | null) {
  if (to < 0 || to >= 24) return false
  const v = board[to]
  // ponytail: shielded blots count as made points
  if (p === 0) {
    if (v === -1 && shield?.[1]) return true
    return v <= -2
  }
  if (v === 1 && shield?.[0]) return true
  return v >= 2
}
export function isLegal(
  board: number[],
  bar: number[],
  from: number,
  to: number,
  die: number,
  movesLeft: number[],
  hasRolled: boolean,
  turn: number,
  shield?: boolean[] | null
) {
  if (!hasRolled) return false
  if (!movesLeft.includes(die)) return false
  if (from === to) return false // ponytail: zero dice are dead
  if (bar[turn] > 0 && from !== -1) return false
  if (bar[turn] === 0 && from === -1) return false
  if (from !== -1) {
    if (from < 0 || from >= 24) return false
    const v = board[from]
    if (turn === 0 && v <= 0) return false
    if (turn === 1 && v >= 0) return false
  }
  if (to === -2) {
    if (die < 1) return false // ponytail: only positive dice bear off
    if (!allInHome(board, bar, turn)) return false
    if (from === -1) return false
    const dist = turn === 0 ? from + 1 : 24 - from
    if (die < dist) return false
    if (die > dist) {
      if (turn === 0) {
        for (let i = from + 1; i < 6; i++) if (board[i] > 0) return false
      } else {
        for (let i = 18; i < from; i++) if (board[i] < 0) return false
      }
    }
    return true
  }
  if (to < 0 || to >= 24) return false
  if (from === -1) {
    if (die < 1 || die > 6) return false // ponytail: only dice 1-6 enter from bar
    const entry = turn === 0 ? 24 - die : die - 1
    if (to !== entry) return false
  } else {
    const expected = turn === 0 ? from - to : to - from
    if (expected !== die) return false
  }
  if (isBlocked(to, turn, board, shield)) return false
  return true
}
export function hasAnyLegal(board: number[], bar: number[], movesLeft: number[], hasRolled: boolean, turn: number, shield?: boolean[] | null) {
  for (const d of movesLeft) {
    for (let from = -1; from < 24; from++) {
      if (from === -1 && bar[turn] === 0) continue
      for (let to = -2; to < 24; to++) {
        if (to === -1) continue
        if (isLegal(board, bar, from, to, d, movesLeft, hasRolled, turn, shield)) return true
      }
    }
  }
  return false
}
