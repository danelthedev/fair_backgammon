import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, waitFor } from '@testing-library/react'
import { Board } from './Board'

vi.mock('./useGame', () => ({ useGame: vi.fn() }))
vi.mock('./sound', () => ({
  playDice: vi.fn(),
  playMove: vi.fn(),
  playTurn: vi.fn(),
  playCapture: vi.fn(),
}))

import { useGame } from './useGame'
import { playDice } from './sound'

const mockUseGame = useGame as unknown as ReturnType<typeof vi.fn>
const mockPlayDice = playDice as unknown as ReturnType<typeof vi.fn>

function base(over: any = {}) {
  const board = Array(24).fill(0)
  const { hook, dice = [0, 0], movesLeft = [], board: _b, ...rest } = over
  return {
    server: {
      board: [...board], bar: [0, 0], off: [0, 0], turn: 0, hasRolled: false,
      players: ['alice', 'bob'], code: 'T',
      scores: [0, 0], rematch: [false, false], ...rest,
      // fresh arrays every render, like production WS messages
      dice: [...dice],
      movesLeft: [...movesLeft],
    },
    local: { board: [...board], bar: [0, 0], off: [0, 0] },
    pending: [],
    movesLeft: [...movesLeft],
    roll: vi.fn(), confirm: vi.fn(), undo: vi.fn(), addMove: vi.fn(),
    error: null, winner: null, winReason: null, myTurn: true, myIdx: 0,
    scores: [0, 0], rematch: [false, false],
    requestRematch: vi.fn(), requestResign: vi.fn(),
    useReroll: vi.fn(), useSkip: vi.fn(), useProtect: vi.fn(),
    connectionError: null, cube: 1, doubleOffer: null,
    requestDouble: vi.fn(), respondDouble: vi.fn(),
    doubledThisTurn: false, lastDoubler: -1,
    brain: null, gifs: [], sendGif: vi.fn(),
  }
}

const sleep = (ms: number) => new Promise(r => setTimeout(r, ms))

describe('roll anim phantom', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('real roll animates; game-end and fresh states stay silent', async () => {
    mockUseGame.mockReturnValue(base())
    const r = render(<Board code="T" username="alice" onLeave={() => {}} />)
    await sleep(50)
    expect(mockPlayDice).not.toHaveBeenCalled()

    // real roll lands
    mockUseGame.mockReturnValue(base({ hasRolled: true, dice: [3, 4], movesLeft: [3, 4] }))
    r.rerender(<Board code="T" username="alice" onLeave={() => {}} />)
    await waitFor(() => expect(document.querySelector('.die.rolling')).toBeTruthy())
    expect(mockPlayDice).toHaveBeenCalledTimes(1)
    await sleep(700)

    // game ends on same dice: silent
    mockUseGame.mockReturnValue(base({ hasRolled: false, movesLeft: [], dice: [3, 4], winner: 'alice', myTurn: false }))
    r.rerender(<Board code="T" username="alice" onLeave={() => {}} />)
    await sleep(100)
    expect(document.querySelector('.die.rolling')).toBeFalsy()
    expect(mockPlayDice).toHaveBeenCalledTimes(1)

    // rematch accepted: fresh game silent
    mockUseGame.mockReturnValue(base({ winner: null, myTurn: true }))
    r.rerender(<Board code="T" username="alice" onLeave={() => {}} />)
    await sleep(100)
    expect(document.querySelector('.die.rolling')).toBeFalsy()
    expect(mockPlayDice).toHaveBeenCalledTimes(1)
  })

  it('dead roll (auto-passed, turn switched) still animates', async () => {
    mockUseGame.mockReturnValue(base())
    const r = render(<Board code="T" username="alice" onLeave={() => {}} />)
    await sleep(50)
    mockUseGame.mockReturnValue(base({ hasRolled: false, movesLeft: [], dice: [5, 2], turn: 1, myTurn: false }))
    r.rerender(<Board code="T" username="alice" onLeave={() => {}} />)
    await waitFor(() => expect(document.querySelector('.die.rolling')).toBeTruthy())
    expect(mockPlayDice).toHaveBeenCalledTimes(1)
  })
})
