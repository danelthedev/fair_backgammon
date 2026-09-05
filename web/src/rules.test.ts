import { describe, it, expect } from 'vitest'
import { isLegal, hasAnyLegal } from './rules'

describe('isLegal', () => {
  it('bar entry only when bar >0', () => {
    const board = Array(24).fill(0)
    board[23] = -2 // blocked
    expect(isLegal(board, [1, 0], -1, 23, 1, [1], true, 0)).toBe(false)
    expect(isLegal(board, [1, 0], 5, 2, 3, [3], true, 0)).toBe(false) // must from bar
    expect(isLegal(board, [1, 0], -1, 22, 2, [2], true, 0)).toBe(true)
  })
  it('bear off needs all home', () => {
    const board = Array(24).fill(0)
    board[0] = 1
    board[10] = 1
    expect(isLegal(board, [0, 0], 0, -2, 1, [1], true, 0)).toBe(false)
    board[10] = 0
    expect(isLegal(board, [0, 0], 0, -2, 1, [1], true, 0)).toBe(true)
  })
  it('bear off single die cannot remove 2 pieces', () => {
    const board = Array(24).fill(0)
    board[0] = 1
    board[1] = 1
    // dice [3] only one die, first bear off with 3 should be ok for from 2? dist 3, but second with same die should fail because die not in movesLeft
    expect(isLegal(board, [0, 0], 0, -2, 3, [3], true, 0)).toBe(false) // dist 1, die 3 > dist but need check no checker beyond? board[1]=1 at higher point than 0, so overshoot from 0 should fail
    expect(isLegal(board, [0, 0], 1, -2, 2, [2], true, 0)).toBe(true)
    // second bear off with same die should fail if die not in movesLeft
    expect(isLegal(board, [0, 0], 0, -2, 2, [3], true, 0)).toBe(false)
  })
  it('bar drag to board not bear off', () => {
    const board = Array(24).fill(0)
    // bar has 1, try to bear off from bar should fail
    expect(isLegal(board, [1, 0], -1, -2, 3, [3], true, 0)).toBe(false)
  })
  it('hasAnyLegal with 2+ bar and blocked', () => {
    const board = Array(24).fill(0)
    board[23] = -2
    board[22] = -2
    // white bar 2, dice [1,2] both entry blocked
    expect(hasAnyLegal(board, [2, 0], [1, 2], true, 0)).toBe(false)
  })
  it('bear off overshoot blocked when higher point occupied (bug #getting_pieces_out)', () => {
    const board = Array(24).fill(0)
    board[5] = 14 // 6th column
    board[2] = 1 // 3rd column
    // white to move, dice 6-4, bear off 3rd with 4 should be illegal (must bear off furthest)
    expect(isLegal(board, [0, 0], 2, -2, 4, [6, 4], true, 0)).toBe(false)
    expect(isLegal(board, [0, 0], 2, -2, 6, [6, 4], true, 0)).toBe(false)
    // exact 6 is legal
    expect(isLegal(board, [0, 0], 5, -2, 6, [6, 4], true, 0)).toBe(true)
    // after bearing off one from 6, 3rd still blocked (13 left on 6)
    const board2 = [...board]; board2[5]--
    expect(isLegal(board2, [0, 0], 2, -2, 4, [4], true, 0)).toBe(false)
    // black mirror: 14 on 18, 1 on 21, dice 6-4
    const b2 = Array(24).fill(0)
    b2[18] = -14
    b2[21] = -1
    expect(isLegal(b2, [0, 0], 21, -2, 4, [6, 4], true, 1)).toBe(false)
  })
})

describe('hasAnyLegal', () => {
  it('detects no moves', () => {
    const board = Array(24).fill(0)
    board[23] = 1
    board[20] = -2
    board[19] = -2
    expect(hasAnyLegal(board, [0, 0], [3, 4], true, 0)).toBe(false)
  })
})
