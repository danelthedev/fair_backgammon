import { useEffect, useRef, useState } from 'react'
import { useGame } from './useGame'
import { useSettings } from './useSettings'

function checkerColor(_v: number, idx: number, _top: boolean) {
  return idx % 2 === 0 ? 'tri dark' : 'tri light'
}

function Dice({ v, rolling, used }: { v: number; rolling: boolean; used?: boolean }) {
  const [display, setDisplay] = useState(v)
  useEffect(() => {
    if (!rolling) {
      setDisplay(v)
      return
    }
    setDisplay(Math.floor(Math.random() * 6) + 1)
    const id = setInterval(() => setDisplay(Math.floor(Math.random() * 6) + 1), 65)
    return () => clearInterval(id)
  }, [rolling, v])
  const cur = rolling ? display : v
  const map: Record<number, number[]> = {
    1: [4],
    2: [0, 8],
    3: [0, 4, 8],
    4: [0, 2, 6, 8],
    5: [0, 2, 4, 6, 8],
    6: [0, 2, 3, 5, 6, 8],
  }
  const dots = map[cur] || []
  return (
    <div className={`die ${rolling ? 'rolling' : ''} ${used ? 'used' : ''}`}>
      {Array.from({ length: 9 }).map((_, i) => (
        <span key={i} className={`pip-dot ${dots.includes(i) ? 'on' : ''}`} />
      ))}
    </div>
  )
}

export function Board({ code, username, onLeave }: { code: string; username: string; onLeave: () => void }) {
  const { server, local, pending, movesLeft, roll, confirm, undo, addMove, error, winner, myTurn, scores, rematch, requestRematch, requestResign, connectionError } = useGame(code, username)
  const { settings } = useSettings()
  const [selected, setSelected] = useState<number | null>(null)
  const [hover, setHover] = useState<number | null>(null)
  const [rolling, setRolling] = useState(false)
  const prevHasRolled = useRef(false)
  const isFirstRollRender = useRef(true)
  const boardRef = useRef<HTMLDivElement>(null)
  const prevServer = useRef<typeof server>(null)
  const [animBoard, setAnimBoard] = useState<{ board: number[]; bar: number[]; off: number[] } | null>(null)
  const [animMoves, setAnimMoves] = useState<{ moves: { from: number; to: number; die: number }[]; mover: number } | null>(null)
  const [fly, setFly] = useState<{ x: number; y: number; color: string; visible: boolean; from: number } | null>(null)
  const animating = !!animBoard
  useEffect(() => {
    if (isFirstRollRender.current) {
      isFirstRollRender.current = false
      prevHasRolled.current = !!server?.hasRolled
      return
    }
    if (server?.hasRolled && !prevHasRolled.current) {
      setRolling(true)
      const t = setTimeout(() => setRolling(false), 600)
      prevHasRolled.current = true
      return () => clearTimeout(t)
    }
    prevHasRolled.current = !!server?.hasRolled
  }, [server?.hasRolled])

  const myIdx = server ? server.players.indexOf(username) : -1

  useEffect(() => {
    setSelected(null)
  }, [server?.turn])

  // ponytail: opponent anim only, mover sees instant
  useEffect(() => {
    if (!server) return
    const prev = prevServer.current
    if (!prev) { prevServer.current = server; return }
    // turn switched -> possible anim
    if (server.turn !== prev.turn) {
      if (server.turn === myIdx && prev.turn !== myIdx && server.lastMoves?.length && !animBoard) {
        setAnimBoard({ board: [...prev.board], bar: [...prev.bar], off: [...prev.off] })
        setAnimMoves({ moves: server.lastMoves, mover: prev.turn })
      }
      prevServer.current = server
      return
    }
    // same turn: intermediate move from opponent, keep original prev for anim
    if (server.turn !== myIdx) return // keep prev as turn-start snapshot
    prevServer.current = server
  }, [server, myIdx, animBoard])

  useEffect(() => {
    if (!animMoves || !animBoard || !boardRef.current) return
    let cancelled = false
    const boardEl = boardRef.current
    ;(async () => {
      let curBoard = [...animBoard.board]
      let curBar = [...animBoard.bar]
      let curOff = [...animBoard.off]
      const mover = animMoves.mover
      const color = mover === 0 ? 'white' : 'black'
      for (const m of animMoves.moves) {
        if (cancelled) break
        const fromEl = m.from === -1 ? boardEl.querySelector(`[data-bar="${mover}"]`) as HTMLElement : boardEl.querySelector(`[data-idx="${m.from}"]`) as HTMLElement
        const toEl = m.to === -2 ? document.querySelector(`[data-off="${mover}"]`) as HTMLElement : boardEl.querySelector(`[data-idx="${m.to}"]`) as HTMLElement
        if (!fromEl || !toEl) {
          // apply logically and continue
          if (m.from === -1) curBar[mover]--
          else { if (mover === 0) curBoard[m.from]--; else curBoard[m.from]++ }
          if (m.to !== -2) {
            const v = curBoard[m.to]
            if (mover === 0 && v === -1) { curBoard[m.to] = 0; curBar[1]++ }
            else if (mover === 1 && v === 1) { curBoard[m.to] = 0; curBar[0]++ }
            if (mover === 0) curBoard[m.to]++; else curBoard[m.to]--
          } else curOff[mover]++
          setAnimBoard({ board: [...curBoard], bar: [...curBar], off: [...curOff] })
          continue
        }
        const br = boardEl.getBoundingClientRect()
        const narrow = typeof window !== 'undefined' && window.innerWidth <= 900
        const flyHalf = narrow ? 18 : 25
        const flyStep = narrow ? 39 : 53
        const flySize = narrow ? 36 : 50
        const fromChecker = fromEl.querySelector('.checker:last-child') as HTMLElement | null
        const fr = (fromChecker || fromEl).getBoundingClientRect()
        const fx = fr.left - br.left + fr.width / 2 - flyHalf
        const fy = fr.top - br.top + fr.height / 2 - flyHalf
        // dest landing: compute exact stack position where new checker will sit
        let tx: number, ty: number
        if (m.to === -2) {
          const tr = toEl.getBoundingClientRect()
          tx = tr.left - br.left + tr.width / 2 - flyHalf
          ty = tr.top - br.top + tr.height / 2 - flyHalf
        } else {
          const pointRect = toEl.getBoundingClientRect()
          const isTop = !!toEl.closest('.row.top')
          const v = curBoard[m.to]
          const hit = (mover === 0 && v === -1) || (mover === 1 && v === 1)
          const hasChecker = !!toEl.querySelector('.checker')
          if (!hasChecker || hit) {
            // empty or hit (cleared): base of stack
            tx = pointRect.left - br.left + pointRect.width / 2 - flyHalf
            ty = isTop ? pointRect.top - br.top + 12 : pointRect.bottom - br.top - 12 - flySize
          } else {
            const toChecker = toEl.querySelector('.checker:last-child') as HTMLElement
            const tr = toChecker.getBoundingClientRect()
            tx = tr.left - br.left + tr.width / 2 - flyHalf
            ty = tr.top - br.top + (isTop ? flyStep : -flyStep)
          }
        }
        setFly({ x: fx, y: fy, color, visible: true, from: m.from })
        await new Promise(r => requestAnimationFrame(() => r(null)))
        await new Promise(r => setTimeout(r, 30))
        if (cancelled) break
        setFly({ x: tx, y: ty, color, visible: true, from: m.from })
        await new Promise(r => setTimeout(r, 380))
        if (cancelled) break
        // land: apply full move (source remove + dest add)
        if (m.from === -1) curBar[mover]--
        else { if (mover === 0) curBoard[m.from]--; else curBoard[m.from]++ }
        if (m.to !== -2) {
          const v = curBoard[m.to]
          if (mover === 0 && v === -1) { curBoard[m.to] = 0; curBar[1]++ }
          else if (mover === 1 && v === 1) { curBoard[m.to] = 0; curBar[0]++ }
          if (mover === 0) curBoard[m.to]++; else curBoard[m.to]--
        } else curOff[mover]++
        setAnimBoard({ board: [...curBoard], bar: [...curBar], off: [...curOff] })
        setFly(null)
        await new Promise(r => setTimeout(r, 120))
      }
      if (!cancelled) { setAnimBoard(null); setAnimMoves(null); setFly(null) }
    })()
    return () => { cancelled = true }
  }, [animMoves])

  if (connectionError) return <div className="boardWrap loading"><div className="error">{connectionError}</div><button className="btn small ghost" onClick={onLeave} style={{ marginTop: 12 }}>Back to lobby</button></div>
  if (!server || !local) return <div className="boardWrap loading">connecting...</div>

  const display = animBoard ?? local
  const opponentIdx = myIdx === 0 ? 1 : 0
  const opponentName = server.players[opponentIdx] || 'waiting…'
  const myName = username
  const allInHome = (board: number[], bar: number[], p: number) => {
    if (bar[p] > 0) return false
    if (p === 0) return board.slice(6).every(v => v <= 0)
    return board.slice(0, 18).every(v => v >= 0)
  }
  const isBlocked = (to: number, p: number, board: number[]) => {
    if (to < 0 || to >= 24) return false
    const v = board[to]
    return p === 0 ? v <= -2 : v >= 2
  }
  const isLegal = (from: number, to: number, die: number) => {
    if (!server.hasRolled) return false
    if (!movesLeft.includes(die)) return false
    if (local.bar[myIdx] > 0 && from !== -1) return false
    if (local.bar[myIdx] === 0 && from === -1) return false
    if (from !== -1) {
      if (from < 0 || from >= 24) return false
      const v = local.board[from]
      if (myIdx === 0 && v <= 0) return false
      if (myIdx === 1 && v >= 0) return false
    }
    if (to === -2) {
      if (!allInHome(local.board, local.bar, myIdx)) return false
      if (from === -1) return false
      const dist = myIdx === 0 ? from + 1 : 24 - from
      if (die < dist) return false
      if (die > dist) {
        if (myIdx === 0) {
          for (let i = from + 1; i < 6; i++) if (local.board[i] > 0) return false
        } else {
          for (let i = 18; i < from; i++) if (local.board[i] < 0) return false
        }
      }
      return true
    }
    if (to < 0 || to >= 24) return false
    if (from === -1) {
      const entry = myIdx === 0 ? 24 - die : die - 1
      if (to !== entry) return false
    } else {
      const expected = myIdx === 0 ? from - to : to - from
      if (expected !== die) return false
    }
    if (isBlocked(to, myIdx, local.board)) return false
    return true
  }

  const hasAnyLegal = () => {
    for (const d of movesLeft) {
      for (let from = -1; from < 24; from++) {
        if (from === -1 && local.bar[myIdx] === 0) continue
        for (let to = -2; to < 24; to++) {
          if (to === -1) continue
          if (isLegal(from, to, d)) return true
        }
      }
    }
    return false
  }

  const source = selected !== null ? selected : hover
  const combinedMap = (() => {
    if (winner || animating || rolling || source === null || !myTurn || !server.hasRolled) return new Map<number, { from: number; to: number; die: number }[]>()
    const map = new Map<number, { from: number; to: number; die: number }[]>()
    const dice = [...movesLeft]
    if (dice.length < 2) return map
    const permute = (arr: number[]): number[][] => {
      if (arr.length <= 1) return [arr]
      const res: number[][] = []
      const seen = new Set<string>()
      for (let i = 0; i < arr.length; i++) {
        const rest = [...arr.slice(0, i), ...arr.slice(i + 1)]
        for (const p of permute(rest)) {
          const perm = [arr[i], ...p]
          const key = perm.join(',')
          if (!seen.has(key)) {
            seen.add(key)
            res.push(perm)
          }
        }
      }
      return res
    }
    for (let len = 2; len <= dice.length; len++) {
      const perms = permute(dice).map(p => p.slice(0, len))
      const uniq = new Map<string, number[]>()
      perms.forEach(p => uniq.set(p.join(','), p))
      for (const seq of uniq.values()) {
        let curBoard = [...local.board]
        let curBar = [...local.bar]
        let curFrom: number | null = source
        const moves: { from: number; to: number; die: number }[] = []
        let ok = true
        for (let i = 0; i < seq.length; i++) {
          const die = seq[i]
          let found: number | null = null
          for (let to = -2; to < 24; to++) {
            if (to === -1) continue
            const isL = (() => {
              if (!server.hasRolled) return false
              if (curBar[myIdx] > 0 && curFrom !== -1) return false
              if (curBar[myIdx] === 0 && curFrom === -1) return false
              if (curFrom !== null && curFrom !== -1) {
                if (curFrom < 0 || curFrom >= 24) return false
                const v = curBoard[curFrom]
                if (myIdx === 0 && v <= 0) return false
                if (myIdx === 1 && v >= 0) return false
              }
              if (to === -2) {
                const allHome = (() => {
                  if (curBar[myIdx] > 0) return false
                  if (myIdx === 0) return curBoard.slice(6).every(v => v <= 0)
                  return curBoard.slice(0, 18).every(v => v >= 0)
                })()
                if (!allHome) return false
                if (curFrom === -1) return false
                const cur = curFrom as number
                const dist = myIdx === 0 ? cur + 1 : 24 - cur
                if (die < dist) return false
                if (die > dist) {
                  if (myIdx === 0) {
                    for (let k = cur + 1; k < 6; k++) if (curBoard[k] > 0) return false
                  } else {
                    for (let k = 18; k < cur; k++) if (curBoard[k] < 0) return false
                  }
                }
                return true
              }
              if (to < 0 || to >= 24) return false
              if (curFrom === -1) {
                const entry = myIdx === 0 ? 24 - die : die - 1
                if (to !== entry) return false
              } else {
                const expected = myIdx === 0 ? (curFrom as number) - to : to - (curFrom as number)
                if (expected !== die) return false
              }
              const blocked = (() => {
                if (to < 0 || to >= 24) return false
                const v = curBoard[to]
                return myIdx === 0 ? v <= -2 : v >= 2
              })()
              if (blocked) return false
              return true
            })()
            if (isL) { found = to; break }
          }
          if (found === null) { ok = false; break }
          moves.push({ from: curFrom as number, to: found, die })
          const m = moves[moves.length - 1]
          if (m.from === -1) curBar[myIdx]--
          else {
            if (myIdx === 0) curBoard[m.from]--
            else curBoard[m.from]++
          }
          if (m.to !== -2) {
            const v = curBoard[m.to]
            if (myIdx === 0 && v === -1) { curBoard[m.to] = 0; curBar[1]++ }
            else if (myIdx === 1 && v === 1) { curBoard[m.to] = 0; curBar[0]++ }
            if (myIdx === 0) curBoard[m.to]++
            else curBoard[m.to]--
          }
          curFrom = found
          if (found === -2) break
        }
        if (ok && moves.length === len) {
          const finalTo = moves[moves.length - 1].to
          if (!map.has(finalTo)) map.set(finalTo, moves)
        }
      }
    }
    return map
  })()

  const validDests = new Set<number>()
  if (!winner && source !== null && myTurn && server.hasRolled && !animating && !rolling) {
    movesLeft.forEach((d: number) => {
      for (let to = -2; to < 24; to++) {
        if (to === -1) continue
        if (isLegal(source, to, d)) validDests.add(to)
      }
    })
    combinedMap.forEach((_, to) => validDests.add(to))
  }

  const handleSelect = (from: number) => {
    if (winner || animating || rolling) return
    if (!myTurn || !server.hasRolled) return
    if (selected === from) {
      setSelected(null)
      return
    }
    // must have piece
    if (from === -1) {
      if (local.bar[myIdx] === 0) return
    } else {
      const v = local.board[from]
      if (myIdx === 0 && v <= 0) return
      if (myIdx === 1 && v >= 0) return
    }
    // must have at least one legal move from there
    let has = false
    for (const d of movesLeft) {
      for (let to = -2; to < 24; to++) {
        if (to === -1) continue
        if (isLegal(from, to, d)) { has = true; break }
      }
      if (has) break
    }
    if (!has) return
    setSelected(from)
  }

  const handleDest = (to: number) => {
    if (winner || animating || rolling) return
    if (selected === null) return
    if (combinedMap.has(to)) {
      const seq = combinedMap.get(to)!
      seq.forEach(m => addMove(m))
      setSelected(null)
      return
    }
    const sorted = [...movesLeft].sort((a: number, b: number) => b - a)
    for (const d of sorted) {
      if (isLegal(selected, to, d)) {
        addMove({ from: selected, to, die: d })
        setSelected(null)
        return
      }
    }
  }

  // ponytail: right-click = biggest single-die legal move from that column
  const handleRightClick = (from: number) => {
    if (winner || animating || rolling) return
    if (!myTurn || !server.hasRolled) return
    if (from === -1) {
      if (local.bar[myIdx] === 0) return
    } else {
      const v = local.board[from]
      if (myIdx === 0 && v <= 0) return
      if (myIdx === 1 && v >= 0) return
    }
    const sorted = [...movesLeft].sort((a: number, b: number) => b - a)
    for (const d of sorted as number[]) {
      for (let to = -2; to < 24; to++) {
        if (to === -1) continue
        if (isLegal(from, to, d)) {
          addMove({ from, to, die: d })
          setSelected(null)
          return
        }
      }
    }
  }

  const isWhiteView = myIdx !== 1
  const topLeft = isWhiteView ? [12, 13, 14, 15, 16, 17] : [11, 10, 9, 8, 7, 6]
  const topRight = isWhiteView ? [18, 19, 20, 21, 22, 23] : [5, 4, 3, 2, 1, 0]
  const botLeft = isWhiteView ? [11, 10, 9, 8, 7, 6] : [12, 13, 14, 15, 16, 17]
  const botRight = isWhiteView ? [5, 4, 3, 2, 1, 0] : [18, 19, 20, 21, 22, 23]
  const barTopIdx = isWhiteView ? 0 : 1
  const barBottomIdx = isWhiteView ? 1 : 0
  const offTopIdx = isWhiteView ? 0 : 1
  const offBottomIdx = isWhiteView ? 1 : 0

  const canConfirm = pending.length > 0 && !hasAnyLegal()
  const showDice = server.dice[0] !== 0
  const isDouble = showDice && server.dice[0] === server.dice[1]
  const diceValues = isDouble ? (Array(4).fill(server.dice[0]) as number[]) : ([...server.dice] as number[])
  const remainingForDice = [...movesLeft]
  const diceUsed = diceValues.map((v: number) => {
    const idx = remainingForDice.indexOf(v)
    if (idx !== -1) {
      remainingForDice.splice(idx, 1)
      return false
    }
    return true
  })
  const renderPoint = (idx: number, top: boolean) => {
    const count = display.board[idx]
    const abs = Math.abs(count)
    const isWhite = count > 0
    const showDot = validDests.has(idx)
    const isSelected = selected === idx
    const isHover = hover === idx
    const hideOne = animating && fly?.from === idx && abs > 0
    // ponytail: overlap only when stack exceeds point height, never past triangle tip
    const cs = typeof window !== 'undefined' && window.innerWidth <= 900 ? 36 : 50
    const AVAIL = typeof window !== 'undefined' && window.innerWidth <= 900 ? 213 : 303
    const FIT = Math.floor((AVAIL + 3) / (cs + 3))
    const isOverflow = abs > FIT
    const MAX = AVAIL
    const gap = isOverflow ? (MAX - abs * cs) / (abs - 1) : 3
    return (
      <div
        key={idx}
        data-idx={idx}
        className={`point ${checkerColor(0, idx, top)} ${isSelected ? 'selected' : ''} ${isHover ? 'hover' : ''}`}
        onMouseEnter={() => setHover(idx)}
        onMouseLeave={() => setHover(null)}
        onClick={() => {
          if (settings.swapClicks) handleRightClick(idx)
          else if (selected !== null && validDests.has(idx)) handleDest(idx)
          else handleSelect(idx)
        }}
        onContextMenu={e => {
          e.preventDefault()
          if (settings.swapClicks) {
            if (selected !== null && validDests.has(idx)) handleDest(idx)
            else handleSelect(idx)
          } else handleRightClick(idx)
        }}
      >
        <div className={`dot ${showDot ? 'show' : ''}`} />
        <div className="stack" style={isOverflow ? { gap: 0 } : undefined}>
          {Array.from({ length: abs }).map((_, i) => {
            const isHidden = hideOne && i === abs - 1
            const s: React.CSSProperties & { zIndex: number } = { zIndex: i, opacity: isHidden ? 0 : 1 }
            if (isOverflow && i !== 0) {
              if (top) (s as any).marginTop = `${gap}px`
              else (s as any).marginBottom = `${gap}px`
            }
            return <div key={i} className={`checker ${isWhite ? 'white' : 'black'}`} style={s} />
          })}
        </div>
      </div>
    )
  }

  return (
    <div className="boardWrap" onContextMenu={e => e.preventDefault()}>
      <div className="topBar">
        {winner ? <span className="turn big">{winner} wins! {scores?.[0] ?? 0}-{scores?.[1] ?? 0}</span> : !(server.players[0] && server.players[1]) ? <span className="codePill">{code}</span> : <span />}
        <div style={{ display: 'flex', gap: '8px' }}>
          {!winner && server.players[0] && server.players[1] && (
            <button className="btn small ghost" onClick={requestResign} style={{ color: '#f87171', borderColor: '#7f1d1d' }}>
              Resign
            </button>
          )}
          <button className="btn small ghost" onClick={onLeave}>leave</button>
        </div>
      </div>

      {error && <div className="error">{error}</div>}
      <div className="playerHeader">
        <span className={`playerPill ${!myTurn ? 'active' : ''}`}>{opponentName} · {scores?.[opponentIdx] ?? 0}</span>
        <span className={`playerPill you ${myTurn ? 'active' : ''}`}>{myName} · {scores?.[myIdx] ?? 0}</span>
      </div>
      <div className="boardRow">
        <div className="board" ref={boardRef} onContextMenu={e => e.preventDefault()} style={animating ? { pointerEvents: 'none' } : undefined}>
          <div className="half left">
            <div className="row top">{topLeft.map(i => renderPoint(i, true))}</div>
            <div className="diceMid">
              {showDice ? (
                <div className={`dicePair ${isDouble ? 'double' : ''}`}>
                  {diceValues.map((v, i) => (
                    <Dice key={i} v={v} rolling={rolling} used={diceUsed[i]} />
                  ))}
                </div>
              ) : (
                <div className="dicePlaceholder">—</div>
              )}
            </div>
            <div className="row bottom">{botLeft.map(i => renderPoint(i, false))}</div>
          </div>

          <div className="barMid">
            <div
              data-bar={barTopIdx}
              className={`barStack top ${selected === -1 && myIdx === barTopIdx ? 'selected' : ''}`}
              onMouseEnter={() => setHover(-1)}
              onMouseLeave={() => setHover(null)}
              onClick={() => {
                if (settings.swapClicks) handleRightClick(-1)
                else if (selected === -1) setSelected(null)
                else if (validDests.has(-2) && selected !== null) handleDest(-2)
                else handleSelect(-1)
              }}
              onContextMenu={e => {
                e.preventDefault()
                if (settings.swapClicks) {
                  if (selected === -1) setSelected(null)
                  else if (validDests.has(-2) && selected !== null) handleDest(-2)
                  else handleSelect(-1)
                } else handleRightClick(-1)
              }}
            >
              {Array.from({ length: display.bar[barTopIdx] }).map((_, i) => {
                const hide = animating && fly?.from === -1 && animMoves?.mover === barTopIdx && i === display.bar[barTopIdx] - 1
                return <div key={`w${i}`} className={`checker ${barTopIdx === 0 ? 'white' : 'black'} ${selected === -1 && myIdx === barTopIdx ? 'selected' : ''}`} style={{ opacity: hide ? 0 : 1 }} />
              })}
              {selected === -1 && <div className={`dot ${validDests.has(-2) ? 'show' : ''}`} style={{ position: 'relative', top: 6 }} />}
            </div>
            <div data-bar={barBottomIdx} className={`barStack bottom ${selected === -1 && myIdx === barBottomIdx ? 'selected' : ''}`} onMouseEnter={() => setHover(-1)} onMouseLeave={() => setHover(null)} onClick={() => { if (settings.swapClicks) handleRightClick(-1); else handleSelect(-1) }} onContextMenu={e => { e.preventDefault(); if (settings.swapClicks) handleSelect(-1); else handleRightClick(-1) }}>
              {Array.from({ length: display.bar[barBottomIdx] }).map((_, i) => {
                const hide = animating && fly?.from === -1 && animMoves?.mover === barBottomIdx && i === display.bar[barBottomIdx] - 1
                return <div key={`b${i}`} className={`checker ${barBottomIdx === 0 ? 'white' : 'black'} ${selected === -1 && myIdx === barBottomIdx ? 'selected' : ''}`} style={{ opacity: hide ? 0 : 1 }} />
              })}
            </div>
          </div>

          <div className="half right">
            <div className="row top">{topRight.map(i => renderPoint(i, true))}</div>
            <div className="diceMid" aria-hidden style={{ visibility: 'hidden' }} />
            <div className="row bottom">{botRight.map(i => renderPoint(i, false))}</div>
          </div>
          {fly?.visible && <div className={`checker ${fly.color} fly`} style={{ left: fly.x, top: fly.y }} />}
        </div>
        <div className={`offTray trough ${selected !== null && validDests.has(-2) ? 'canBearOff' : ''}`} onClick={() => { if (selected !== null && validDests.has(-2)) handleDest(-2) }}>
          <div className="troughInner">
            <div className={`offStack ${offTopIdx === 0 ? 'white-trough' : 'black-trough'}`} data-off={offTopIdx}>
              {Array.from({ length: display.off[offTopIdx] }).map((_, i) => (
                <div key={`o${offTopIdx}${i}`} className={`checker ${offTopIdx === 0 ? 'white' : 'black'} small`} />
              ))}
            </div>
            <div className="troughCenter" aria-hidden />
            <div className={`offStack ${offBottomIdx === 0 ? 'white-trough' : 'black-trough'}`} data-off={offBottomIdx}>
              {Array.from({ length: display.off[offBottomIdx] }).map((_, i) => (
                <div key={`o${offBottomIdx}${i}`} className={`checker ${offBottomIdx === 0 ? 'white' : 'black'} small`} />
              ))}
            </div>
          </div>
        </div>
        <div className="sideBtn">
          {!winner && myTurn && !server.hasRolled && !animating && !rolling && server.players[0] && server.players[1] ? (
            <button className="btn primary large" onClick={roll}>Roll</button>
          ) : !winner && pending.length > 0 ? (
            <button className="btn ghost large" onClick={undo}>Undo</button>
          ) : null}
          {canConfirm && !winner && (
            <button className="btn primary large" onClick={confirm} style={{ marginTop: 10 }}>Confirm</button>
          )}
        </div>
      </div>
      {winner && (
        <div className="rematchBox" style={{ textAlign: 'center', margin: '16px 0' }}>
          <button className="btn primary large" onClick={requestRematch} disabled={rematch?.[myIdx]} style={{ marginTop: 10 }}>
            {rematch?.[myIdx] ? 'Waiting for opponent...' : 'Rematch'}
          </button>
          {rematch?.[opponentIdx] && !rematch?.[myIdx] && <div className="hint" style={{ marginTop: 6 }}>Opponent wants rematch</div>}
        </div>
      )}
    </div>
  )
}
