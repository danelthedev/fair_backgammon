import { useState } from 'react'

export type SavedLayout = { name: string; board: number[]; bar: [number, number]; off: [number, number]; turn: number }

const KEY = 'fair_backgammon_layouts'
// ponytail: matches game.NewGame standard position
export const STANDARD_BOARD = [-2, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, -5, 5, 0, 0, 0, -3, 0, -5, 0, 0, 0, 0, 2]

export function loadLayouts(): SavedLayout[] {
  try {
    const raw = JSON.parse(localStorage.getItem(KEY) ?? '[]')
    if (!Array.isArray(raw)) return []
    return raw.filter(l => l && typeof l.name === 'string' && Array.isArray(l.board) && l.board.length === 24)
  } catch { return [] }
}

function persist(list: SavedLayout[]) {
  try { localStorage.setItem(KEY, JSON.stringify(list)) } catch {}
}

// ponytail: white-view order, same as the game board
const TOP = [12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23]
const BOT = [11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0]
const CK = 26
const SHOW = 5

function Point({ idx, pos, v, paint }: { idx: number; pos: number; v: number; paint: (i: number, c: 1 | -1) => void }) {
  const top = pos < 12
  const n = Math.abs(v)
  const shown = Math.min(n, SHOW)
  return (
    <div
      onClick={() => paint(idx, 1)}
      onContextMenu={e => { e.preventDefault(); paint(idx, -1) }}
      title={`point ${idx}: left white, right black`}
      style={{ position: 'relative', height: 152, cursor: 'pointer', background: '#e19247', borderRadius: 4, overflow: 'hidden' }}
    >
      <div style={{
        position: 'absolute', inset: 0,
        background: pos % 2 === 0 ? '#9f452d' : '#f4b862',
        clipPath: top ? 'polygon(0 0, 100% 0, 50% 100%)' : 'polygon(50% 0, 0 100%, 100% 100%)',
      }} />
      <div style={{ position: 'absolute', top: top ? 2 : undefined, bottom: top ? undefined : 2, width: '100%', textAlign: 'center', fontSize: 9, opacity: .6, zIndex: 3 }}>{idx}</div>
      {Array.from({ length: shown }).map((_, k) => (
        <div
          key={k}
          style={{
            position: 'absolute', left: '50%', transform: 'translateX(-50%)',
            top: top ? 6 + k * (CK + 1) : undefined,
            bottom: top ? undefined : 6 + k * (CK + 1),
            width: CK, height: CK, borderRadius: '50%', pointerEvents: 'none', zIndex: 2,
            background: v > 0 ? '#fffaf0' : '#1e1e1e',
            border: '1px solid #a0a0a0',
            boxShadow: '0 2px 5px rgba(0,0,0,.4)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontSize: 11, fontWeight: 800, color: v > 0 ? '#333' : '#fff',
          }}
        >
          {k === shown - 1 && n > SHOW ? `+${n - SHOW + 1}` : ''}
        </div>
      ))}
    </div>
  )
}

export function LayoutEditor({ onSave, onClose }: { onSave: (l: SavedLayout) => void; onClose: () => void }) {
  const [saved, setSaved] = useState<SavedLayout[]>(loadLayouts)
  const [name, setName] = useState('')
  const [board, setBoard] = useState<number[]>([...STANDARD_BOARD])

  // ponytail: one rule — left adds white, right adds black; opposite clicks
  // shrink the stack through zero, then build the other color
  const paint = (i: number, color: 1 | -1) => setBoard(b => {
    const n = [...b]; n[i] += color; return n
  })

  let w = 0, b = 0
  for (const v of board) { if (v > 0) w += v; else b -= v }
  const valid = w >= 1 && b >= 1 && name.trim() !== ''

  const del = (n: string) => {
    const next = loadLayouts().filter(l => l.name !== n)
    persist(next); setSaved(next)
  }

  const half = (idxs: number[], top: boolean) => (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(6, 1fr)', gap: 3, flex: 1 }}>
      {idxs.map((idx, k) => <Point key={idx} idx={idx} pos={top ? k : k + 12} v={board[idx]} paint={paint} />)}
    </div>
  )

  return (
    <>
      <div className="pickerBackdrop" onClick={onClose} />
      <div style={{ position: 'fixed', inset: 0, zIndex: 50, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 16, pointerEvents: 'none' }}>
        <div className="card" onContextMenu={e => e.preventDefault()} style={{ pointerEvents: 'auto', width: 600, maxWidth: '100%', maxHeight: 'calc(100dvh - 60px)', overflowY: 'auto', alignItems: 'stretch', textAlign: 'left', borderTop: '3px solid var(--accent)' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <strong>Board layouts</strong>
            <button className="btn small ghost" onClick={onClose}>✕</button>
          </div>

          {saved.length > 0 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
              {saved.map(l => (
                <div key={l.name} style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                  <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{l.name}</span>
                  <button className="btn small" onClick={() => { setName(l.name); setBoard([...l.board]) }}>Load</button>
                  <button className="btn small ghost" onClick={() => del(l.name)}>✕</button>
                </div>
              ))}
            </div>
          )}

          <div style={{ display: 'flex', gap: 6, background: '#7a3f2a', border: '6px solid #3e2723', borderRadius: 12, padding: 6 }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6, flex: 1 }}>
              {half(TOP.slice(0, 6), true)}
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6, flex: 1 }}>
              {half(TOP.slice(6), true)}
            </div>
          </div>
          <div style={{ display: 'flex', gap: 6, background: '#7a3f2a', border: '6px solid #3e2723', borderTop: 'none', borderRadius: '0 0 12px 12px', padding: 6, marginTop: -8 }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6, flex: 1 }}>
              {half(BOT.slice(0, 6), false)}
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6, flex: 1 }}>
              {half(BOT.slice(6), false)}
            </div>
          </div>

          <div style={{ fontSize: '.85rem', opacity: .75 }}>White {w} · Black {b}</div>

          <div style={{ display: 'flex', gap: 8 }}>
            <input className="input" placeholder="Layout name" value={name} onChange={e => setName(e.target.value)} maxLength={30} style={{ flex: 1 }} />
            <button className="btn" onClick={() => setBoard([...STANDARD_BOARD])}>Standard</button>
            <button className="btn" onClick={() => setBoard(Array(24).fill(0))}>Clear</button>
          </div>
          <button
            className="btn primary large"
            disabled={!valid}
            onClick={() => { const l = { name: name.trim(), board: [...board], bar: [0, 0] as [number, number], off: [0, 0] as [number, number], turn: 0 }; const next = [...loadLayouts().filter(x => x.name !== l.name), l]; persist(next); onSave(l) }}
          >
            Save layout
          </button>
        </div>
      </div>
    </>
  )
}
