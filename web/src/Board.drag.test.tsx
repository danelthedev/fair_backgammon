import { describe, it, expect, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/react'
import { Board } from './Board'
import { useGame } from './useGame'

vi.mock('./useGame', () => ({ useGame: vi.fn() }))
const mockUseGame = useGame as unknown as ReturnType<typeof vi.fn>

describe('click select', () => {
  it('select bar then valid dest shows dots', () => {
    mockUseGame.mockReturnValue({
      server: { board: Array(24).fill(0), bar: [1, 0], off: [0, 0], turn: 0, dice: [2, 3], movesLeft: [2, 3], hasRolled: true, players: ['alice', 'bob'], code: 'TEST' },
      local: { board: Array(24).fill(0), bar: [1, 0], off: [0, 0] },
      pending: [],
      movesLeft: [2, 3],
      roll: vi.fn(),
      confirm: vi.fn(),
      undo: vi.fn(),
      addMove: vi.fn(),
      error: null,
      winner: null,
      myTurn: true,
    } as any)
    const { container } = render(<Board code="TEST" username="alice" onLeave={() => {}} />)
    const bar = container.querySelector('.barStack.top') as HTMLElement
    fireEvent.click(bar)
    expect(container.querySelectorAll('.dot.show').length).toBeGreaterThan(0)
  })

  it('click dest after select does move', () => {
    const addMove = vi.fn()
    const board = Array(24).fill(0)
    board[5] = 1
    mockUseGame.mockReturnValue({
      server: { board, bar: [0, 0], off: [0, 0], turn: 0, dice: [3, 4], movesLeft: [3, 4], hasRolled: true, players: ['alice', 'bob'], code: 'TEST' },
      local: { board, bar: [0, 0], off: [0, 0] },
      pending: [],
      movesLeft: [3, 4],
      roll: vi.fn(),
      confirm: vi.fn(),
      undo: vi.fn(),
      addMove,
      error: null,
      winner: null,
      myTurn: true,
    } as any)
    const { container } = render(<Board code="TEST" username="alice" onLeave={() => {}} />)
    const from = container.querySelector('[data-idx="5"]') as HTMLElement
    fireEvent.click(from)
    const dest = container.querySelector('[data-idx="2"]') as HTMLElement
    fireEvent.click(dest)
    expect(addMove).toHaveBeenCalled()
  })

  it('dots for selected column after move without leave', () => {
    const board = Array(24).fill(0)
    board[5] = 2
    mockUseGame.mockReturnValue({
      server: { board, bar: [0, 0], off: [0, 0], turn: 0, dice: [3, 4], movesLeft: [4], hasRolled: true, players: ['alice', 'bob'], code: 'TEST' },
      local: { board, bar: [0, 0], off: [0, 0] },
      pending: [{ from: 5, to: 2, die: 3 }],
      movesLeft: [4],
      roll: vi.fn(),
      confirm: vi.fn(),
      undo: vi.fn(),
      addMove: vi.fn(),
      error: null,
      winner: null,
      myTurn: true,
    } as any)
    const { container } = render(<Board code="TEST" username="alice" onLeave={() => {}} />)
    const point = container.querySelector('[data-idx="5"]') as HTMLElement
    fireEvent.click(point)
    expect(container.querySelectorAll('.dot.show').length).toBeGreaterThan(0)
  })
})
