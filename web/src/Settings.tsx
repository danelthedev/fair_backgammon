import { useState } from 'react'
import { useSettings } from './useSettings'

const labels: Record<string, string> = {
  boardBase: 'Base board',
  boardField: 'Board field',
  triLight: 'Triangle light',
  triDark: 'Triangle dark',
  exterior: 'Exterior / border',
  whitePiece: 'White pieces',
  blackPiece: 'Black pieces',
  pieceBorder: 'Piece border',
  diceBase: 'Dice base',
  diceDots: 'Dice dots',
}
// ponytail: downscale to <=1024px jpeg so the dataURL fits localStorage
function fileToFieldImage(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file)
    const img = new Image()
    img.onload = () => {
      const s = Math.min(1, 1024 / Math.max(img.width, img.height))
      const c = document.createElement('canvas')
      c.width = Math.max(1, Math.round(img.width * s))
      c.height = Math.max(1, Math.round(img.height * s))
      c.getContext('2d')?.drawImage(img, 0, 0, c.width, c.height)
      URL.revokeObjectURL(url)
      resolve(c.toDataURL('image/jpeg', 0.85))
    }
    img.onerror = () => { URL.revokeObjectURL(url); reject(new Error('bad image')) }
    img.src = url
  })
}

export function SettingsButton() {
  const { settings, update, updateColor, reset, randomize } = useSettings()
  const [open, setOpen] = useState(false)

  return (
    <div style={{ position: 'fixed', top: 12, right: 12, zIndex: 100 }}>
      <button
        aria-label="settings"
        onClick={() => setOpen(v => !v)}
        className="btn small ghost"
        style={{ fontSize: '1.2rem', padding: '6px 10px', borderRadius: '10px', background: 'var(--card)', border: '1px solid var(--line)' }}
      >
        ⚙
      </button>
      {open && (
        <div
          style={{
            position: 'absolute',
            top: '42px',
            right: 0,
            width: 280,
            background: 'var(--card)',
            border: '1px solid var(--line)',
            borderRadius: 12,
            padding: 12,
            display: 'flex',
            flexDirection: 'column',
            gap: 10,
            boxShadow: '0 12px 32px rgba(0,0,0,0.5)',
          }}
          onClick={e => e.stopPropagation()}
        >
          <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: '0.9rem', cursor: 'pointer' }}>
            <input type="checkbox" checked={settings.swapClicks} onChange={e => update({ swapClicks: e.target.checked })} />
            Swap left / right click
          </label>
          <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: '0.9rem', cursor: 'pointer' }}>
            <input type="checkbox" checked={settings.sound} onChange={e => update({ sound: e.target.checked })} />
            <span style={{ whiteSpace: 'nowrap', fontSize: '0.9rem' }}>Sound effects</span>
            <input
              type="range"
              min={0}
              max={100}
              value={settings.volume ?? 100}
              onChange={e => update({ volume: Number(e.target.value) })}
              style={{ width: 80, cursor: 'pointer' }}
              aria-label="Sound volume"
            />
            <span style={{ fontSize: '0.75rem', opacity: 0.7, minWidth: 30 }}>{settings.volume ?? 100}%</span>
          </label>
          <div style={{ height: 1, background: 'var(--line)', opacity: 0.5 }} />
          {Object.entries(labels).map(([key, label]) => (
            <div key={key} style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-start', gap: 8 }}>
              <input
                type="color"
                value={(settings.colors as any)[key]}
                onChange={e => updateColor(key as any, e.target.value)}
                style={{ width: 32, height: 28, padding: 0, border: '1px solid var(--line)', borderRadius: 6, cursor: 'pointer', background: 'transparent', flexShrink: 0 }}
                aria-label={label}
              />
              <span style={{ fontSize: '0.85rem', opacity: 0.9, flex: 1 }}>{label}</span>
              {key === 'boardField' && (
                <span style={{ display: 'flex', alignItems: 'center', gap: 4 }} title="Field image opacity">
                  <input
                    type="range"
                    min={0}
                    max={100}
                    value={settings.boardFieldAlpha}
                    onChange={e => update({ boardFieldAlpha: Number(e.target.value) })}
                    style={{ width: 64, cursor: 'pointer' }}
                    aria-label="Field image opacity"
                  />
                  <span style={{ fontSize: '0.75rem', opacity: 0.7, minWidth: 30 }}>{settings.boardFieldAlpha}%</span>
                </span>
              )}
              {(key === 'whitePiece' || key === 'blackPiece') && (
                <span style={{ display: 'flex', alignItems: 'center', gap: 4 }} title={key === 'whitePiece' ? 'White piece image' : 'Black piece image'}>
                  <label className="btn small ghost" style={{ cursor: 'pointer', padding: '2px 6px' }}>
                    🖼
                    <input
                      type="file"
                      accept="image/*"
                      hidden
                      onChange={e => {
                        const f = e.target.files?.[0]
                        e.target.value = ''
                        if (f) fileToFieldImage(f).then(d => update(key === 'whitePiece' ? { whitePieceImage: d } : { blackPieceImage: d })).catch(() => {})
                      }}
                    />
                  </label>
                  {(key === 'whitePiece' ? settings.whitePieceImage : settings.blackPieceImage) && (
                    <button
                      className="btn small ghost"
                      style={{ padding: '2px 6px' }}
                      onClick={() => update(key === 'whitePiece' ? { whitePieceImage: null } : { blackPieceImage: null })}
                      title="Remove image"
                    >
                      ✕
                    </button>
                  )}
                </span>
              )}
              {key === 'boardField' && (
                <span style={{ display: 'flex', alignItems: 'center', gap: 4 }} title="Field image">
                  <label className="btn small ghost" style={{ cursor: 'pointer', padding: '2px 6px' }}>
                    🖼
                    <input
                      type="file"
                      accept="image/*"
                      hidden
                      onChange={e => {
                        const f = e.target.files?.[0]
                        e.target.value = ''
                        if (f) fileToFieldImage(f).then(d => update({ boardFieldImage: d })).catch(() => {})
                      }}
                    />
                  </label>
                  {settings.boardFieldImage && (
                    <button
                      className="btn small ghost"
                      style={{ padding: '2px 6px' }}
                      onClick={() => update({ boardFieldImage: null })}
                      title="Remove image"
                    >
                      ✕
                    </button>
                  )}
                </span>
              )}
              {key === 'diceBase' && (
                <span style={{ display: 'flex', alignItems: 'center', gap: 4 }} title="Dice image">
                  <label className="btn small ghost" style={{ cursor: 'pointer', padding: '2px 6px' }}>
                    🖼
                    <input
                      type="file"
                      accept="image/*"
                      hidden
                      onChange={e => {
                        const f = e.target.files?.[0]
                        e.target.value = ''
                        if (f) fileToFieldImage(f).then(d => update({ diceImage: d })).catch(() => {})
                      }}
                    />
                  </label>
                  {settings.diceImage && (
                    <button
                      className="btn small ghost"
                      style={{ padding: '2px 6px' }}
                      onClick={() => update({ diceImage: null })}
                      title="Remove image"
                    >
                      ✕
                    </button>
                  )}
                </span>
              )}
            </div>
          ))}
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }} title="Image over all triangles">
            <span style={{ fontSize: '0.85rem', opacity: 0.9, flex: 1 }}>Triangles image</span>
            <label className="btn small ghost" style={{ cursor: 'pointer', padding: '2px 6px' }}>
              🖼
              <input
                type="file"
                accept="image/*"
                hidden
                onChange={e => {
                  const f = e.target.files?.[0]
                  e.target.value = ''
                  if (f) fileToFieldImage(f).then(d => update({ triImage: d })).catch(() => {})
                }}
              />
            </label>
            <input
              type="range"
              min={0}
              max={100}
              value={settings.triImageAlpha ?? 100}
              onChange={e => update({ triImageAlpha: Number(e.target.value) })}
              style={{ width: 64, cursor: 'pointer' }}
              aria-label="Triangles image opacity"
            />
            <span style={{ fontSize: '0.75rem', opacity: 0.7, minWidth: 30 }}>{settings.triImageAlpha ?? 100}%</span>
            {settings.triImage && (
              <button
                className="btn small ghost"
                style={{ padding: '2px 6px' }}
                onClick={() => update({ triImage: null })}
                title="Remove image"
              >
                ✕
              </button>
            )}
          </div>
          <button className="btn small ghost" onClick={randomize} style={{ marginTop: 4 }}>
            Randomize all colors
          </button>
          <button className="btn small ghost" onClick={reset}>
            Reset colors to default
          </button>
        </div>
      )}
    </div>
  )
}
