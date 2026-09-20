import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { Board } from './Board'

vi.mock('./useGame', () => ({
  useGame: vi.fn(),
}))

import { useGame } from './useGame'

const mockUseGame = useGame as unknown as ReturnType<typeof vi.fn>

// ponytail: right-click quick-move must keep hover dots for remaining dice
// (regression of 8c63c2f, broken again by mobile stale-hover clearing)
describe('right-click keeps hint dots', () => {
  beforeEach(() => vi.clearAllMocks())

  it('dots survive a right-click quick move', () => {
    const board = Array(24).fill(0)
    board[5] = 2 // white, alice moves 5->2 with a 3
    const addMove = vi.fn()
    mockUseGame.mockReturnValue({
      server: {
        board: [...board], bar: [0, 0], off: [0, 0], turn: 0, dice: [3, 4],
        movesLeft: [3, 3], hasRolled: true, players: ['alice', 'bob'], code: 'TEST',
      },
      local: { board: [...board], bar: [0, 0], off: [0, 0] },
      pending: [],
      movesLeft: [3, 3],
      roll: vi.fn(),
      confirm: vi.fn(),
      undo: vi.fn(),
      addMove,
      error: null,
      winner: null,
      myTurn: true,
    })
    const { container } = render(<Board code="TEST" username="alice" onLeave={() => {}} />)
    const from = container.querySelector('[data-idx="5"]')!
    const dest = container.querySelector('[data-idx="2"]')!
    fireEvent.mouseEnter(from)
    expect(dest.querySelector('.dot.show')).toBeTruthy()
    fireEvent.contextMenu(from)
    expect(addMove).toHaveBeenCalledWith({ from: 5, to: 2, die: 3 })
    // dots still hint the remaining die instead of vanishing
    expect(dest.querySelector('.dot.show')).toBeTruthy()
    expect(screen.queryByText('Roll')).toBeNull() // sanity: no crash, board alive
  })
})
