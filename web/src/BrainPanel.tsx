// Live fly-brain panel: canvas dots at true soma positions, lighting up
// SEQUENTIALLY along pathways (BFS wave from the strongest unit), not all at
// once. Observer only — renders WS {t:"brain"} frames, never decides.
// Desktop only via .brainSide CSS (hidden <=1024px).
import { useEffect, useRef } from 'react'

export type BrainFrame = { units: [number, number][]; edges: [number, number][] } | null

let coordsP: Promise<([number, number] | null)[]> | null = null
let bgP: Promise<HTMLImageElement> | null = null

function assets() {
  if (!coordsP) {
    coordsP = fetch('brain_coords.json').then(r => r.json()).then(d => d.coords as ([number, number] | null)[])
    bgP = new Promise(res => {
      const img = new Image()
      img.onload = () => res(img)
      img.src = 'brain_bg.png'
    })
  }
  return Promise.all([coordsP, bgP!])
}

const RED_LIFE: [number, number, number] = [255, 70, 25]
const YEL_LIFE: [number, number, number] = [255, 220, 130]
const GRY_LIFE: [number, number, number] = [150, 150, 150]
function mix3(a: [number, number, number], b: [number, number, number], f: number): [number, number, number] {
  return [Math.round(a[0] + (b[0] - a[0]) * f), Math.round(a[1] + (b[1] - a[1]) * f), Math.round(a[2] + (b[2] - a[2]) * f)]
}

type Wave = {
  order: number[]
  at: Map<number, number>
  edgeAt: Map<string, number>
  vmax: number
  vals: [number, number][]
  t0: number
} | null

const STEP_MS = 20

function buildWave(f: NonNullable<BrainFrame>): Omit<Exclude<Wave, null>, 't0'> & { t0?: number } {
  const vOf = new Map<number, number>(f.units)
  const adj = new Map<number, number[]>()
  for (const [i, j] of f.edges) {
    if (!vOf.has(i) || !vOf.has(j)) continue
    if (!adj.has(i)) adj.set(i, [])
    if (!adj.has(j)) adj.set(j, [])
    adj.get(i)!.push(j)
    adj.get(j)!.push(i)
  }
  let start = f.units[0]?.[0] ?? -1
  let vmax = 0
  for (const [i, v] of f.units) if (v > vmax) { vmax = v; start = i }
  const order: number[] = []
  const seen = new Set<number>()
  const q: number[] = start >= 0 ? [start] : []
  if (start >= 0) seen.add(start)
  while (q.length) {
    const u = q.shift()!
    order.push(u)
    for (const w of adj.get(u) ?? []) {
      if (!seen.has(w)) { seen.add(w); q.push(w) }
    }
  }
  for (const [i] of [...f.units].sort((a, b) => b[1] - a[1])) {
    if (!seen.has(i)) { seen.add(i); order.push(i) }
  }
  const at = new Map<number, number>()
  order.forEach((u, k) => at.set(u, k * STEP_MS))
  const edgeAt = new Map<string, number>()
  for (const [i, j] of f.edges) {
    const a = at.get(i)
    const b = at.get(j)
    if (a === undefined || b === undefined) continue
    edgeAt.set(i < j ? `${i}-${j}` : `${j}-${i}`, Math.max(a, b))
  }
  return { order, at, edgeAt, vmax, vals: f.units }
}

export function BrainPanel({ frame }: { frame: BrainFrame }) {
  const ref = useRef<HTMLCanvasElement>(null)
  const wave = useRef<Wave>(null)

  useEffect(() => {
    let live = true
    let raf = 0
    assets().then(([coords, bg]) => {
      if (!live) return
      const draw = () => {
        const cv = ref.current
        if (cv) {
          const dpr = Math.min(2, window.devicePixelRatio || 1)
          const cw = Math.max(1, Math.round(cv.clientWidth * dpr))
          const ch = Math.max(1, Math.round(cv.clientHeight * dpr))
          if (cv.width !== cw || cv.height !== ch) { cv.width = cw; cv.height = ch }
          const ctx = cv.getContext('2d')
          if (ctx) {
            const W = cv.width
            const H = cv.height
            ctx.fillStyle = '#000'
            ctx.fillRect(0, 0, W, H)
            const s = Math.min(W / bg.width, H / bg.height)
            const dw = bg.width * s
            const dh = bg.height * s
            const dx = (W - dw) / 2
            const dy = (H - dh) / 2
            ctx.globalAlpha = 0.95
            ctx.drawImage(bg, dx, dy, dw, dh)
            ctx.globalAlpha = 1
            const wv = wave.current
            if (wv && coords) {
              const px = (nx: number) => dx + nx * dw
              const py = (ny: number) => dy + (1 - ny) * dh
              ctx.strokeStyle = '#ff2a1a'
              ctx.lineWidth = Math.max(2, W / 160)
              for (const [key, tEdge] of wv.edgeAt) {
                const dt = performance.now() - wv.t0 - tEdge
                if (dt < 0) continue
                const [a, b] = key.split('-').map(Number)
                const pa = coords[a]
                const pb = coords[b]
                if (!pa || !pb) continue
                const ta = wv.at.get(a) ?? tEdge
                const tb = wv.at.get(b) ?? tEdge
                const sx = ta <= tb ? px(pa[0]) : px(pb[0])
                const sy = ta <= tb ? py(pa[1]) : py(pb[1])
                const ex = ta <= tb ? px(pb[0]) : px(pa[0])
                const ey = ta <= tb ? py(pb[1]) : py(pa[1])
                const p = Math.min(1, dt / 150)
                const e = 1 - Math.pow(1 - p, 2)
                ctx.globalAlpha = 0.16 * Math.exp(-dt / 2200)
                ctx.beginPath()
                ctx.moveTo(sx, sy)
                ctx.lineTo(ex, ey)
                ctx.stroke()
                ctx.globalAlpha = 0.95 * Math.exp(-dt / 2200)
                ctx.beginPath()
                ctx.moveTo(sx, sy)
                ctx.lineTo(sx + (ex - sx) * e, sy + (ey - sy) * e)
                ctx.stroke()
              }
              ctx.globalAlpha = 1
              const vmap = new Map<number, number>()
              for (const [i, v] of wv.vals) vmap.set(i, v)
              for (const u of wv.order) {
                const dt = performance.now() - wv.t0 - (wv.at.get(u) ?? 0)
                if (dt < 0) continue
                const p = coords[u]
                if (!p) continue
                const v = vmap.get(u) ?? 0
                const vn = wv.vmax > 0 ? v / wv.vmax : 0
                if (vn <= 0.02) continue
                const udt = Math.max(0, dt)
                const ufade = Math.exp(-udt / 2200)
                const a1 = Math.min(1, udt / 500)
                const yellowed = mix3(RED_LIFE, YEL_LIFE, a1)
                const a2 = Math.min(1, Math.max(0, (udt - 500) / 900))
                const col = mix3(yellowed, GRY_LIFE, a2)
                const c = `rgb(${col[0]},${col[1]},${col[2]})`
                const t = vn * Math.min(1, dt / 150) * ufade
                ctx.fillStyle = c
                ctx.shadowColor = c
                ctx.shadowBlur = 10 * t
                ctx.beginPath()
                ctx.arc(px(p[0]), py(p[1]), (1.8 + 2.6 * t) * dpr, 0, Math.PI * 2)
                ctx.fill()
              }
              ctx.shadowBlur = 0
              ctx.globalAlpha = 1
            }
          }
        }
        raf = requestAnimationFrame(draw)
      }
      raf = requestAnimationFrame(draw)
    })
    return () => { live = false; cancelAnimationFrame(raf) }
  }, [])

  useEffect(() => {
    if (frame) {
      const wv = buildWave(frame) as Wave
      if (wv) {
        wv.t0 = performance.now()
        wave.current = wv
      }
    }
  }, [frame])

  return <canvas ref={ref} className="brainPanel" aria-label="fly brain activity" />
}
