import { useEffect, useRef, useState } from 'react'

export type GifItem = { id: string; url: string; preview: string; v?: string }

// ponytail: picker hits /api/gifs/* — Klipy key never touches the browser.
export function GifPicker({ onPick, onClose }: { onPick: (g: GifItem) => void; onClose: () => void }) {
  const [items, setItems] = useState<GifItem[]>([])
  const [loading, setLoading] = useState(false)
  const [q, setQ] = useState('')
  const fetchQ = async (query: string) => {
    setLoading(true)
    try {
      const ep = query ? `search?q=${encodeURIComponent(query)}&limit=12` : 'trending?limit=12'
      const r = await fetch(`/api/gifs/${ep}`, { credentials: 'include' })
      const j = await r.json()
      setItems(Array.isArray(j.items) ? j.items : [])
    } catch {
      setItems([])
    }
    setLoading(false)
  }
  const first = useRef(true)
  useEffect(() => {
    if (first.current) { first.current = false; fetchQ(''); return }
    const t = setTimeout(() => fetchQ(q), 300)
    return () => clearTimeout(t)
  }, [q])
  return (
    <div className="gifPicker">
      <input
        className="input gifSearch"
        autoFocus
        placeholder="Search KLIPY"
        value={q}
        onChange={e => setQ(e.target.value)}
        onKeyDown={e => {
          if (e.key === 'Escape') onClose()
          if (e.key === 'Enter' && items[0]) onPick(items[0])
        }}
      />
      {loading && items.length === 0 ? (
        <div className="hint gifHint">loading…</div>
      ) : (
        <div className="gifGrid">
          {items.map((g, i) => (
            <button key={g.id || i} className="gifThumb" title="Send this GIF" onClick={() => onPick(g)}>
              <img src={g.preview || g.url} alt="gif reaction" loading="lazy" />
            </button>
          ))}
        </div>
      )}
    </div>
  )
}