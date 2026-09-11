import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, act } from '@testing-library/react'
import { StrictMode } from 'react'
import { Board } from './Board'
import { SettingsProvider } from './useSettings'

class MockWS {
  static instances: MockWS[] = []
  onmessage: ((e: any) => void) | null = null
  onerror: ((e: any) => void) | null = null
  onclose: (() => void) | null = null
  onopen: (() => void) | null = null
  readyState = 1
  sent: string[] = []
  url: string
  constructor(url: string) { this.url = url; MockWS.instances.push(this) }
  send(d: string) { this.sent.push(d) }
  close() {}
}
;(MockWS as any).OPEN = 1
vi.stubGlobal('WebSocket', MockWS)

function stateMsg(over: any = {}) {
  return JSON.stringify({
    t: 'state',
    board: Array(24).fill(0),
    bar: [0, 0],
    off: [0, 0],
    turn: 0,
    dice: [0, 0],
    movesLeft: [],
    hasRolled: false,
    players: ['alice', 'bob'],
    code: 'TEST',
    lastMoves: [],
    scores: [0, 0],
    rematch: [false, false],
    ...over,
  })
}

describe('repro blank screen', () => {
  beforeEach(() => { MockWS.instances = []; vi.clearAllMocks() })

  it('mounts, receives lobby state, no crash', async () => {
    const { container } = render(
      <StrictMode>
        <SettingsProvider>
          <Board code="TEST" username="alice" onLeave={() => {}} />
        </SettingsProvider>
      </StrictMode>
    )
    expect(MockWS.instances.length).toBeGreaterThan(0)
    const ws = MockWS.instances[MockWS.instances.length - 1]
    await act(async () => { ws.onmessage?.({ data: stateMsg() }) })
    expect(container.textContent).toContain('alice')
  })

  it('survives turn change with opponent moves (anim path)', async () => {
    const { container } = render(
      <StrictMode>
        <SettingsProvider>
          <Board code="TEST" username="alice" onLeave={() => {}} />
        </SettingsProvider>
      </StrictMode>
    )
    const ws = MockWS.instances[MockWS.instances.length - 1]
    const board = Array(24).fill(0)
    board[12] = -2
    await act(async () => { ws.onmessage?.({ data: stateMsg({ board, turn: 1, dice: [3, 2], movesLeft: [3, 2], hasRolled: true }) }) })
    const moved = Array(24).fill(0)
    moved[12] = -1
    moved[9] = -1
    await act(async () => {
      ws.onmessage?.({ data: stateMsg({ board: moved, turn: 0, dice: [3, 2], movesLeft: [], hasRolled: false, lastMoves: [{ from: 12, to: 9, die: 3 }] }) })
    })
    await act(async () => { await new Promise(r => setTimeout(r, 2500)) })
    expect(container.querySelector('.board')).toBeTruthy()
  }, 15000)
})
