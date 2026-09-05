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
}

export function SettingsButton() {
  const { settings, update, updateColor, reset } = useSettings()
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
            </div>
          ))}
          <button className="btn small ghost" onClick={reset} style={{ marginTop: 4 }}>
            Reset colors to default
          </button>
        </div>
      )}
    </div>
  )
}
