import { useEffect, useRef, useState } from 'react'

export type Move = { from: number; to: number; die: number } // from -1 bar, to -2 off
export type ServerState = {
  board: number[]
  bar: number[]
  off: number[]
  turn: number
  dice: number[]
  movesLeft: number[]
  hasRolled: boolean
  players: string[]
  code: string
  lastMoves?: Move[]
  scores?: [number, number]
  rematch?: [boolean, boolean]
}

export function useGame(code: string, username: string) {
  const wsRef = useRef<WebSocket | null>(null)
  const serverRef = useRef<ServerState | null>(null)
  const [server, setServer] = useState<ServerState | null>(null)
  const [pending, setPending] = useState<Move[]>([])
  const [error, setError] = useState<string | null>(null)
  const [winner, setWinner] = useState<string | null>(null)
  useEffect(() => { serverRef.current = server }, [server])

  // local board derived from server + pending
  const local = (() => {
    if (!server) return null
    const b = [...server.board] as number[]
    const bar = [...server.bar] as number[]
    const off = [...server.off] as number[]
    // apply pending optimistically for display
    pending.forEach((m) => {
      const p = server.turn
      // remove from
      if (m.from === -1) bar[p]--
      else {
        if (p === 0) b[m.from]--
        else b[m.from]++
      }
      // hit ?
      if (m.to !== -2) {
        const v = b[m.to]
        if (p === 0 && v === -1) {
          b[m.to] = 0
          bar[1]++
        } else if (p === 1 && v === 1) {
          b[m.to] = 0
          bar[0]++
        }
        if (p === 0) b[m.to]++
        else b[m.to]--
      } else {
        off[p]++
      }
    })
    return { board: b, bar, off }
  })()

  // remaining dice after pending
  const movesLeft = (() => {
    if (!server) return []
    const ml = [...server.movesLeft]
    pending.forEach((m) => {
      const idx = ml.indexOf(m.die)
      if (idx !== -1) ml.splice(idx, 1)
    })
    return ml
  })()

  useEffect(() => {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${proto}//${location.host}/ws?code=${code}`)
    wsRef.current = ws
    ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data)
        if (msg.t === 'state') {
          setServer({
            board: msg.board,
            bar: msg.bar,
            off: msg.off,
            turn: msg.turn,
            dice: msg.dice,
            movesLeft: msg.movesLeft || [],
            hasRolled: msg.hasRolled,
            players: msg.players,
            code: msg.code,
            lastMoves: msg.lastMoves || [],
            scores: msg.scores || [0, 0],
            rematch: msg.rematch || [false, false],
          })
          // clear winner when new game starts (rematch)
          if (msg.scores && msg.rematch && !msg.rematch[0] && !msg.rematch[1] && msg.board) {
            // if board is fresh and no one has won yet, clear winner
            const off0 = msg.off?.[0] ?? 0
            const off1 = msg.off?.[1] ?? 0
            if (off0 < 15 && off1 < 15) setWinner(null)
          }
          setError(null)
        } else if (msg.t === 'error') {
          setError(msg.msg)
          setTimeout(() => setError(null), 2000)
        } else if (msg.t === 'win') {
          const name = msg.winnerName ?? serverRef.current?.players[msg.winner] ?? `Player ${msg.winner}`
          setWinner(name)
          setError(`${name} wins!`)
        } else if (msg.t === 'rematch') {
          // rematch vote update comes via state, but also handle direct
          if (msg.scores) {
            setServer(s => s ? { ...s, scores: msg.scores, rematch: msg.rematch } : s)
          }
        }
      } catch {}
    }
    ws.onopen = () => {}
    return () => ws.close()
  }, [code])

  useEffect(() => {
    if (server) setPending([])
  }, [server?.turn])
  const send = (o: any) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) wsRef.current.send(JSON.stringify(o))
    else setError('ws not ready')
  }

  const roll = () => {
    if (!myTurn) { setError('not your turn'); return }
    send({ t: 'roll' })
  }
  const confirm = () => {
    if (pending.length === 0) return
    // send pending sequentially
    pending.forEach((m) => send({ t: 'move', from: m.from, to: m.to, die: m.die }))
    setPending([])
  }
  const undo = () => setPending((p) => p.slice(0, -1))
  const addMove = (m: Move) => setPending((p) => [...p, m])
  const requestRematch = () => send({ t: 'rematch' })

  const myIdx = server ? server.players.indexOf(username) : -1
  const myTurn = server ? server.turn === myIdx : false
  const scores = server?.scores ?? [0, 0] as [number, number]
  const rematch = server?.rematch ?? [false, false] as [boolean, boolean]

  return { server, local, pending, movesLeft, roll, confirm, undo, addMove, error, winner, myTurn, myIdx, send, scores, rematch, requestRematch }
}
