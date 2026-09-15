import { useEffect, useRef, useState } from 'react'
import { useGame } from './useGame'
import { useSettings } from './useSettings'
import { playCapture, playDice, playMove, playTurn } from './sound'

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
  const { server, local, pending, movesLeft, roll, confirm, undo, addMove, error, winner, winReason, myTurn, scores, rematch, requestRematch, requestResign, connectionError, cube, doubleOffer, requestDouble, respondDouble, doubledThisTurn, lastDoubler } = useGame(code, username)
  const { settings } = useSettings()
  const vol = (settings.volume ?? 100) / 100
  const [selected, setSelected] = useState<number | null>(null)
  const [hover, setHover] = useState<number | null>(null)
  const [rolling, setRolling] = useState(false)
  const [autoRoll, setAutoRoll] = useState(() => { try { return localStorage.getItem('fair_backgammon_autoroll') === '1' } catch { return false } })
  const toggleAutoRoll = () => setAutoRoll(v => { const n = !v; try { localStorage.setItem('fair_backgammon_autoroll', n ? '1' : '0') } catch {} return n })
  const prevHasRolled = useRef(false)
  const isFirstRollRender = useRef(true)
  const boardRef = useRef<HTMLDivElement>(null)
  const prevServer = useRef<typeof server>(null)
  const [animBoard, setAnimBoard] = useState<{ board: number[]; bar: number[]; off: number[] } | null>(null)
  const [animMoves, setAnimMoves] = useState<{ moves: { from: number; to: number; die: number }[]; mover: number } | null>(null)
  const [fly, setFly] = useState<{ x: number; y: number; color: string; visible: boolean; from: number } | null>(null)
  const animating = !!animBoard
  // ponytail: drag feeds the same select/dest path as click, no parallel rules
  const [drag, setDrag] = useState<{ from: number; x: number; y: number; touch: boolean } | null>(null)
  const suppressRef = useRef(false)
  const dragOnRef = useRef(false)
  const candRef = useRef<{ from: number; x0: number; y0: number; id: number } | null>(null)
  const apiRef = useRef<{ canStart: (f: number) => boolean; target: (x: number, y: number) => number | null; drop: (f: number, t: number) => void }>({ canStart: () => false, target: () => null, drop: () => {} })
  useEffect(() => {
    if (isFirstRollRender.current) {
      isFirstRollRender.current = false
      prevHasRolled.current = !!server?.hasRolled
      return
    }
    if (server?.hasRolled && !prevHasRolled.current) {
      sfx.dice()
      setRolling(true)
      const t = setTimeout(() => setRolling(false), 600)
      prevHasRolled.current = true
      return () => clearTimeout(t)
    }
    prevHasRolled.current = !!server?.hasRolled
  }, [server?.hasRolled])
  // ponytail: auto-roll fires once per turn start, delayed so double stays possible
  useEffect(() => {
    if (!autoRoll || !myTurn || winner || !server || server.hasRolled || rolling || animating || doubleOffer || !(server.players[0] && server.players[1])) return
    const t = setTimeout(() => roll(), 1000)
    return () => clearTimeout(t)
  }, [autoRoll, myTurn, winner, server?.hasRolled, server?.turn, rolling, animating, doubleOffer, server?.players])
  // ponytail: notify at turn start; deferred to anim end when opponent moves play
  const prevTurn = useRef(server?.turn)
  useEffect(() => {
    if (prevTurn.current !== undefined && server && server.turn !== prevTurn.current && server.players[server.turn] === username && server.players[0] && server.players[1] && !server.lastMoves?.length) sfx.turn()
    prevTurn.current = server?.turn
  }, [server?.turn])

  const myIdx = server ? server.players.indexOf(username) : -1

  // ponytail: capture knock instead of slide when a blot gets hit
  const sfx = {
    move: (hit = false) => (hit ? playCapture : playMove)(settings.sound, vol, settings.customSounds?.[hit ? 'capture' : 'move']),
    dice: () => playDice(settings.sound, vol, settings.customSounds?.dice),
    turn: () => playTurn(settings.sound, vol, settings.customSounds?.turn),
  }
  const isHit = (to: number) => !!local && to !== -2 && to >= 0 && to < 24 && (myIdx === 0 ? local.board[to] === -1 : local.board[to] === 1)
  useEffect(() => {
    setSelected(null)
    setHover(null) // ponytail: touch hover lingers, clear with turn
  }, [server?.turn])

  // ponytail: Safari freezes dvh on scroll-locked pages — track live height instead
  useEffect(() => {
    let lastH = 0 // ponytail: big jumps = keyboard/toolbar, yank page back to top
    const setH = () => {
      const h = window.visualViewport?.height ?? window.innerHeight
      document.documentElement.style.setProperty('--apph', `${h}px`)
      if (lastH && Math.abs(h - lastH) > 100) window.scrollTo(0, 0)
      lastH = h
    }
    setH()
    ;(document.activeElement as HTMLElement | null)?.blur?.() // ponytail: drop lobby keyboard, it pans page
    window.scrollTo(0, 0)
    window.visualViewport?.addEventListener('resize', setH)
    window.addEventListener('orientationchange', setH)
    return () => {
      window.visualViewport?.removeEventListener('resize', setH)
      window.removeEventListener('orientationchange', setH)
    }
  }, [])
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
        const landedHit = m.to !== -2 && m.to >= 0 && ((mover === 0 && curBoard[m.to] === -1) || (mover === 1 && curBoard[m.to] === 1))
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
          sfx.move(landedHit)
          continue
        }
        const br = boardEl.getBoundingClientRect()
        const narrow = typeof window !== 'undefined' && (window.innerWidth <= 900 || window.innerHeight <= 600)
        // ponytail: measure real checker so fly lands right on mobile sizes
        const flySize = narrow ? (boardEl.querySelector('.checker')?.getBoundingClientRect().width || 24) : 50
        const flyHalf = flySize / 2
        const flyStep = flySize + 3
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
        sfx.move(landedHit)
        setFly(null)
        await new Promise(r => setTimeout(r, 120))
      }
      if (!cancelled) { await new Promise(r => setTimeout(r, 700)); if (!cancelled) { setAnimBoard(null); setAnimMoves(null); setFly(null); sfx.turn() } }
    })()
    return () => { cancelled = true }
  }, [animMoves])

  // ponytail: window-level move/up so drops outside the board still resolve; hooks live above early returns
  useEffect(() => {
    const TH = 8
    const onMove = (e: PointerEvent) => {
      const c = candRef.current
      if (!c || e.pointerId !== c.id) return
      if (!dragOnRef.current) {
        if (Math.hypot(e.clientX - c.x0, e.clientY - c.y0) < TH) return
        if (!apiRef.current.canStart(c.from)) { candRef.current = null; return }
        dragOnRef.current = true
        suppressRef.current = true
        setSelected(c.from)
        setDrag({ from: c.from, x: e.clientX, y: e.clientY, touch: e.pointerType === 'touch' || e.pointerType === 'pen' })
        return
      }
      setDrag(d => (d ? { ...d, x: e.clientX, y: e.clientY } : d))
      const t = apiRef.current.target(e.clientX, e.clientY)
      setHover(t !== null && t >= 0 ? t : null)
    }
    const onUp = (e: PointerEvent) => {
      const c = candRef.current
      if (!c || e.pointerId !== c.id) return
      candRef.current = null
      if (!dragOnRef.current) return // plain tap, onClick handles it
      dragOnRef.current = false
      setDrag(null)
      setHover(null)
      setSelected(null)
      setTimeout(() => { suppressRef.current = false }, 0) // eaten tap may never come
      const t = apiRef.current.target(e.clientX, e.clientY)
      if (t !== null) apiRef.current.drop(c.from, t)
    }
    const onAbort = () => {
      if (!candRef.current && !dragOnRef.current) return
      candRef.current = null
      dragOnRef.current = false
      suppressRef.current = false
      setDrag(null)
      setHover(null)
      setSelected(null)
    }
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
    window.addEventListener('pointercancel', onAbort)
    window.addEventListener('blur', onAbort)
    return () => {
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
      window.removeEventListener('pointercancel', onAbort)
      window.removeEventListener('blur', onAbort)
    }
  }, [])

  if (connectionError) return <div className="boardWrap loading"><div className="error">{connectionError}</div><button className="btn small ghost" onClick={onLeave} style={{ marginTop: 12 }}>Back to lobby</button></div>
  if (!server || !local) return <div className="boardWrap loading">connecting...</div>

  const display = animBoard ?? local
  const opponentIdx = myIdx === 0 ? 1 : 0
  const opponentName = server.players[opponentIdx] || 'waiting…'
  const myName = username
  const pipCount = (board: number[], bar: number[], p: number) => {
    let n = bar[p] * 25
    for (let i = 0; i < 24; i++) {
      const v = board[i]
      if (p === 0 && v > 0) n += v * (i + 1)
      if (p === 1 && v < 0) n += -v * (24 - i)
    }
    return n
  }
  const myPips = myIdx < 0 ? 0 : pipCount(display.board, display.bar, myIdx)
  const oppPips = myIdx < 0 ? 0 : pipCount(display.board, display.bar, opponentIdx)
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
  // ponytail: single-die version for greying out unplayable dice
  const canPlayDie = (die: number) => {
    if (!local || !myTurn || !server.hasRolled) return false
    for (let from = -1; from < 24; from++) {
      if (from === -1 && local.bar[myIdx] === 0) continue
      for (let to = -2; to < 24; to++) {
        if (to === -1) continue
        if (isLegal(from, to, die)) return true
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
      setHover(null) // ponytail: re-tap untoggles dots too
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

  // ponytail: explicit-from core so drag drops don't depend on select timing
  const handleDestFrom = (from: number, to: number) => {
    if (winner || animating || rolling) return
    if (combinedMap.has(to)) {
      const seq = combinedMap.get(to)!
      if (seq.length > 0 && seq[0].from === from) {
        seq.forEach(m => addMove(m))
        sfx.move(seq.some(m => isHit(m.to)))
        setSelected(null)
        setHover(null) // ponytail: touch keeps stale hover, dots lingered
        return
      }
    }
    const sorted = [...movesLeft].sort((a: number, b: number) => b - a)
    for (const d of sorted) {
      if (isLegal(from, to, d)) {
        addMove({ from, to, die: d })
        sfx.move(isHit(to))
        setSelected(null)
        setHover(null) // ponytail: touch keeps stale hover, dots lingered
        return
      }
    }
  }
  const handleDest = (to: number) => {
    if (selected === null) return
    handleDestFrom(selected, to)
  }

  // ponytail: drag reuses click guards; legality still enforced inside handleDestFrom
  const canDragFrom = (from: number) => {
    if (!server || !local) return false
    if (winner || animating || rolling) return false
    if (!myTurn || !server.hasRolled) return false
    if (from === -1) return local.bar[myIdx] > 0
    if (from < 0 || from >= 24) return false
    const v = local.board[from]
    return myIdx === 0 ? v > 0 : v < 0
  }
  const dropTarget = (x: number, y: number): number | null => {
    if (typeof document.elementFromPoint !== 'function') return null
    const el = document.elementFromPoint(x, y)
    const t = el?.closest?.('[data-idx],[data-off]')
    if (!t) return null
    return t.hasAttribute('data-idx') ? Number(t.getAttribute('data-idx')) : -2
  }
  apiRef.current = { canStart: canDragFrom, target: dropTarget, drop: handleDestFrom }
  const beginCandidate = (e: React.PointerEvent, from: number) => {
    if (e.pointerType === 'mouse' && e.button !== 0) return
    if (candRef.current) return
    candRef.current = { from, x0: e.clientX, y0: e.clientY, id: e.pointerId }
  }
  const consumeTap = () => {
    if (suppressRef.current) { suppressRef.current = false; return true }
    return false
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
          sfx.move(isHit(to))
          setSelected(null)
          setHover(null) // ponytail: quick-move leaves stale hover dots
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
  const stake = cube && cube > 1 ? cube : 1
  const bothHere = !!(server.players[0] && server.players[1])
  const canDouble = !winner && myTurn && !server.hasRolled && !animating && !rolling && bothHere && !doubleOffer && !doubledThisTurn && (lastDoubler ?? -1) !== myIdx && stake < 64
  const showDice = server.dice[0] !== 0
  const isDouble = showDice && server.dice[0] === server.dice[1]
  const diceValues = isDouble ? (Array(4).fill(server.dice[0]) as number[]) : ([...server.dice] as number[])
  const remainingForDice = [...movesLeft]
  // ponytail: dice with no legal move from current spot grey out immediately
  const deadValues = myTurn && server.hasRolled ? new Set(movesLeft.filter(d => !canPlayDie(d))) : new Set<number>()
  const diceUsed = diceValues.map((v: number) => {
    const idx = remainingForDice.indexOf(v)
    if (idx !== -1) {
      remainingForDice.splice(idx, 1)
      return deadValues.has(v)
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
    // ponytail: measure real checker/point so overlap math matches CSS at any size
    const narrowPt = typeof window !== 'undefined' && (window.innerWidth <= 900 || window.innerHeight <= 600)
    const ptEl = narrowPt ? boardRef.current?.querySelector('.point') as HTMLElement | null : null
    const ckEl = narrowPt ? boardRef.current?.querySelector('.checker') as HTMLElement | null : null
    const cs = narrowPt ? Math.max(18, Math.round(ckEl?.getBoundingClientRect().width || 24)) : 50
    const AVAIL = narrowPt ? Math.max(80, (ptEl?.clientHeight || 200) - 20) : 303
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
        onPointerDown={e => beginCandidate(e, idx)}
        onClick={() => {
          if (consumeTap()) return
          if (settings.swapClicks) handleRightClick(idx)
          else if (selected !== null && validDests.has(idx)) handleDest(idx)
          else handleSelect(idx)
        }}
        onContextMenu={e => {
          e.preventDefault()
          if (drag) return
          if (settings.swapClicks) {
            if (selected !== null && validDests.has(idx)) handleDest(idx)
            else handleSelect(idx)
          } else handleRightClick(idx)
        }}
      >
        <div className={`dot ${showDot ? 'show' : ''}`} />
        <div className="stack" style={isOverflow ? { gap: 0 } : undefined}>
          {Array.from({ length: abs }).map((_, i) => {
            const isHidden = (hideOne || drag?.from === idx) && i === abs - 1
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
        {winner ? <span className="turn big">{winner} wins{winReason === 'gammon' || winReason === 'backgammon' ? ` by ${winReason}` : ''}! {scores?.[0] ?? 0}-{scores?.[1] ?? 0}</span> : !(server.players[0] && server.players[1]) ? <span className="codePill">{code}</span> : <span />}
        <div style={{ display: 'flex', gap: '8px' }}>
          {!winner && server.players[0] && server.players[1] && (
            <button className="btn small ghost" onClick={requestResign} style={{ color: '#f87171', borderColor: '#7f1d1d' }}>
              Resign
            </button>
          )}
          <button className="btn small ghost" onClick={onLeave}>Leave</button>
        </div>
      </div>

      {error && <div className="error">{error}</div>}
      <div className="playerHeader">
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span className={`playerPill ${!myTurn ? 'active' : ''}`}>{opponentName} · {scores?.[opponentIdx] ?? 0}</span>
          <span className="pipCount">{oppPips}</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span className="pipCount">{myPips}</span>
          <span className={`playerPill you ${myTurn ? 'active' : ''}`}>{myName} · {scores?.[myIdx] ?? 0}</span>
        </div>
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
                <div className="dicePlaceholder">{doubleOffer && doubleOffer.by === myIdx ? 'Waiting for double…' : '—'}</div>
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
              onPointerDown={e => beginCandidate(e, -1)}
              onClick={() => {
                if (consumeTap()) return
                if (settings.swapClicks) handleRightClick(-1)
                else if (selected === -1) { setSelected(null); setHover(null) }
                else if (validDests.has(-2) && selected !== null) handleDest(-2)
                else handleSelect(-1)
              }}
              onContextMenu={e => {
                e.preventDefault()
                if (drag) return
                if (settings.swapClicks) {
                  if (selected === -1) { setSelected(null); setHover(null) }
                  else if (validDests.has(-2) && selected !== null) handleDest(-2)
                  else handleSelect(-1)
                } else handleRightClick(-1)
              }}
            >
              {Array.from({ length: display.bar[barTopIdx] }).map((_, i) => {
                const hide = (animating && fly?.from === -1 && animMoves?.mover === barTopIdx && i === display.bar[barTopIdx] - 1) || (drag?.from === -1 && myIdx === barTopIdx && i === display.bar[barTopIdx] - 1)
                return <div key={`w${i}`} className={`checker ${barTopIdx === 0 ? 'white' : 'black'} ${selected === -1 && myIdx === barTopIdx ? 'selected' : ''}`} style={{ opacity: hide ? 0 : 1 }} />
              })}
              {selected === -1 && <div className={`dot ${validDests.has(-2) ? 'show' : ''}`} style={{ position: 'relative', top: 6 }} />}
            </div>
            <div data-bar={barBottomIdx} className={`barStack bottom ${selected === -1 && myIdx === barBottomIdx ? 'selected' : ''}`} onMouseEnter={() => setHover(-1)} onMouseLeave={() => setHover(null)} onPointerDown={e => beginCandidate(e, -1)} onClick={() => { if (consumeTap()) return; if (settings.swapClicks) handleRightClick(-1); else handleSelect(-1) }} onContextMenu={e => { e.preventDefault(); if (drag) return; if (settings.swapClicks) handleSelect(-1); else handleRightClick(-1) }}>
              {Array.from({ length: display.bar[barBottomIdx] }).map((_, i) => {
                const hide = (animating && fly?.from === -1 && animMoves?.mover === barBottomIdx && i === display.bar[barBottomIdx] - 1) || (drag?.from === -1 && myIdx === barBottomIdx && i === display.bar[barBottomIdx] - 1)
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
          {drag && <div className={`checker ${myIdx === 0 ? 'white' : 'black'} drag-ghost`} style={{ position: 'fixed', left: drag.x, top: drag.y, transform: drag.touch ? 'translate(-50%, -135%)' : 'translate(-50%, -50%)', zIndex: 60, pointerEvents: 'none' }} />}
          {!winner && doubleOffer && doubleOffer.by !== myIdx && (
            <div className="doubleOverlay">
              <div className="doubleBox">
                <div className="turn big">{server.players[doubleOffer.by] || 'Opponent'} doubles to {doubleOffer.stake}</div>
                <button className="btn primary large" onClick={() => respondDouble('accept')}>Accept ×{doubleOffer.stake}</button>
                {doubleOffer.stake < 64 && (
                  <button className="btn ghost large" onClick={() => respondDouble('redouble')}>Re-double to ×{doubleOffer.stake * 2}</button>
                )}
                <button className="btn small ghost" onClick={() => respondDouble('reject')} style={{ color: '#f87171', borderColor: '#7f1d1d' }}>Reject (lose)</button>
              </div>
            </div>
          )}
        </div>
        <div className={`offTray trough ${selected !== null && validDests.has(-2) ? 'canBearOff' : ''}`} onClick={() => { if (consumeTap()) return; if (selected !== null && validDests.has(-2)) handleDest(-2) }}>
          <div className="troughInner">
            <div className={`offStack ${offTopIdx === 0 ? 'white-trough' : 'black-trough'}`} data-off={offTopIdx}>
              {Array.from({ length: display.off[offTopIdx] }).map((_, i) => (
                <div key={`o${offTopIdx}${i}`} className={`checker ${offTopIdx === 0 ? 'white' : 'black'} small`} />
              ))}
            </div>
            <div className="troughCenter">
              {canDouble && (
                <button className="doubleBtn" onClick={requestDouble}>Double</button>
              )}
              <div className="cube" title={stake <= 1 ? 'No double yet' : `Stake ×${stake}`}>{stake <= 1 ? 64 : stake}</div>
            </div>
            <div className={`offStack ${offBottomIdx === 0 ? 'white-trough' : 'black-trough'}`} data-off={offBottomIdx}>
              {Array.from({ length: display.off[offBottomIdx] }).map((_, i) => (
                <div key={`o${offBottomIdx}${i}`} className={`checker ${offBottomIdx === 0 ? 'white' : 'black'} small`} />
              ))}
            </div>
          </div>
        </div>
        <div className="sideBtn">
          {!winner && myTurn && !server.hasRolled && !animating && !rolling && !doubleOffer && !autoRoll && server.players[0] && server.players[1] ? (
            <button className="btn primary large" onClick={roll}>Roll</button>
          ) : !winner && pending.length > 0 ? (
            <button className="btn ghost large" onClick={undo}>Undo</button>
          ) : null}
          {canConfirm && !winner && (
            <button className="btn primary large" onClick={confirm} style={{ marginTop: 10 }}>Confirm</button>
          )}
          {!winner && (
            <div className={`autoRoll ${autoRoll ? 'active' : ''}`} onClick={toggleAutoRoll} title="Roll automatically at turn start">
              Auto-roll
            </div>
          )}
        </div>
      </div>
      {winner && (
        <div className="rematchBox" style={{ textAlign: 'center', margin: '16px 0' }}>
          <div className="winLabelMobile turn big">{winner} wins{winReason === 'gammon' || winReason === 'backgammon' ? ` by ${winReason}` : ''}! {scores?.[0] ?? 0}-{scores?.[1] ?? 0}</div>
          <button className="btn primary large" onClick={requestRematch} disabled={rematch?.[myIdx]} style={{ marginTop: 10 }}>
            {rematch?.[myIdx] ? 'Waiting for opponent...' : 'Rematch'}
          </button>
          {rematch?.[opponentIdx] && !rematch?.[myIdx] && <div className="hint" style={{ marginTop: 6 }}>Opponent wants rematch</div>}
        </div>
      )}
    </div>
  )
}
