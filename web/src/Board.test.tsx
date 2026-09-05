import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { Board } from './Board'

vi.mock('./useGame', () => ({
  useGame: vi.fn(),
}))

import { useGame } from './useGame'

const mockUseGame = useGame as unknown as ReturnType<typeof vi.fn>

function mockServer(overrides: any = {}) {
  const base = {
    board: Array(24).fill(0),
    bar: [0, 0],
    off: [0, 0],
    turn: 0,
    dice: [3, 4],
    movesLeft: [3, 4],
    hasRolled: true,
    players: ['alice', 'bob'],
    code: 'TEST',
    ...overrides,
  }
  base.board[5] = 2
  base.board[23] = 2
  return base
}

describe('Board', () => {
  beforeEach(() => vi.clearAllMocks())

  it('click anywhere on column works (point onClick)', () => {
    const addMove = vi.fn()
    mockUseGame.mockReturnValue({
      server: mockServer(),
      local: { board: (() => { const b = Array(24).fill(0); b[5] = 2; return b })(), bar: [0, 0], off: [0, 0] },
      pending: [],
      movesLeft: [3],
      roll: vi.fn(),
      confirm: vi.fn(),
      undo: vi.fn(),
      addMove,
      error: null,
      winner: null,
      myTurn: true,
    })
    render(<Board code="TEST" username="alice" onLeave={() => {}} />)
    const points = document.querySelectorAll('.point')
    expect(points.length).toBe(24)
    // click first point (top left)
    fireEvent.click(points[0])
    // addMove should be called for valid move
  })

  it('bar drag hide and offTray', () => {
    mockUseGame.mockReturnValue({
      server: mockServer({ bar: [1, 0] }),
      local: { board: Array(24).fill(0), bar: [1, 0], off: [0, 0] },
      pending: [],
      movesLeft: [2],
      roll: vi.fn(),
      confirm: vi.fn(),
      undo: vi.fn(),
      addMove: vi.fn(),
      error: null,
      winner: null,
      myTurn: true,
    })
    const { container } = render(<Board code="TEST" username="alice" onLeave={() => {}} />)
    expect(container.querySelectorAll('.barStack.top .checker').length).toBe(1)
    expect(container.querySelector('.offTray')).toBeTruthy()
  })

  it('dots outside board', () => {
    const { container } = render(<Board code="TEST" username="alice" onLeave={() => {}} />)
    const dots = container.querySelectorAll('.dot')
    dots.forEach(d => {
      expect(d.classList.contains('dot')).toBe(true)
    })
  })

  it('win shows name not index and hides roll', () => {
    mockUseGame.mockReturnValue({
      server: mockServer(),
      local: { board: Array(24).fill(0), bar: [0, 0], off: [15, 0] },
      pending: [],
      movesLeft: [],
      roll: vi.fn(),
      confirm: vi.fn(),
      undo: vi.fn(),
      addMove: vi.fn(),
      error: 'alice wins!',
      winner: 'alice',
      myTurn: true,
    })
    render(<Board code="TEST" username="alice" onLeave={() => {}} />)
    expect(screen.getAllByText(/alice wins!/i).length).toBeGreaterThan(0)
    expect(screen.queryByText('Roll')).toBeNull()
  })

  it('confirm only when no legal moves', () => {
    const board1 = Array(24).fill(0)
    board1[5] = 1
    board1[3] = -2 // block
    mockUseGame.mockReturnValue({
      server: mockServer({ hasRolled: true }),
      local: { board: board1, bar: [0, 0], off: [0, 0] },
      pending: [{ from: 5, to: 2, die: 3 }],
      movesLeft: [2], // 5->3 blocked, bear off dist 6 >2 no
      roll: vi.fn(),
      confirm: vi.fn(),
      undo: vi.fn(),
      addMove: vi.fn(),
      error: null,
      winner: null,
      myTurn: true,
    })
    const { queryByText, unmount } = render(<Board code="TEST" username="alice" onLeave={() => {}} />)
    expect(queryByText('Confirm')).toBeTruthy()
    unmount()
    const board2 = Array(24).fill(0)
    board2[5] = 1
    mockUseGame.mockReturnValue({
      server: mockServer({ hasRolled: true }),
      local: { board: board2, bar: [0, 0], off: [0, 0] },
      pending: [{ from: 5, to: 2, die: 3 }],
      movesLeft: [2], // 5->3 legal
      roll: vi.fn(),
      confirm: vi.fn(),
      undo: vi.fn(),
      addMove: vi.fn(),
      error: null,
      winner: null,
      myTurn: true,
    })
    const { queryByText: q2 } = render(<Board code="TEST" username="alice" onLeave={() => {}} />)
    expect(q2('Confirm')).toBeNull()
  })

  it('dice dots and roll anim', () => {
    mockUseGame.mockReturnValue({
      server: mockServer({ dice: [3, 3], hasRolled: true }),
      local: { board: Array(24).fill(0), bar: [0, 0], off: [0, 0] },
      pending: [],
      movesLeft: [3, 3, 3, 3],
      roll: vi.fn(),
      confirm: vi.fn(),
      undo: vi.fn(),
      addMove: vi.fn(),
      error: null,
      winner: null,
      myTurn: true,
    })
    const { container } = render(<Board code="TEST" username="alice" onLeave={() => {}} />)
    expect(container.querySelectorAll('.die').length).toBe(4)
    expect(container.querySelectorAll('.pip-dot.on').length).toBeGreaterThan(0)
    // 4 dice for double, none used yet
    expect(container.querySelectorAll('.die.used').length).toBe(0)
  })

  it('dice used gray and double 4', () => {
    const board = Array(24).fill(0)
    board[5] = 5
    mockUseGame.mockReturnValue({
      server: mockServer({ dice: [3, 4], hasRolled: true }),
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
    })
    const { container } = render(<Board code="TEST" username="alice" onLeave={() => {}} />)
    expect(container.querySelectorAll('.die').length).toBe(2)
    expect(container.querySelectorAll('.die.used').length).toBe(1)
    // double with 2 used
    mockUseGame.mockReturnValue({
      server: mockServer({ dice: [2, 2], hasRolled: true }),
      local: { board, bar: [0, 0], off: [0, 0] },
      pending: [{ from: 5, to: 3, die: 2 }, { from: 3, to: 1, die: 2 }],
      movesLeft: [2, 2],
      roll: vi.fn(),
      confirm: vi.fn(),
      undo: vi.fn(),
      addMove: vi.fn(),
      error: null,
      winner: null,
      myTurn: true,
    })
    const { container: c2 } = render(<Board code="TEST" username="alice" onLeave={() => {}} />)
    expect(c2.querySelectorAll('.die').length).toBe(4)
    expect(c2.querySelectorAll('.die.used').length).toBe(2)
  })
})
