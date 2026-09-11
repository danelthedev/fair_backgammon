import { useState } from 'react'
import type { ReactNode } from 'react'
import { defaultColors, defaultSettings, useSettings, type BoardColors } from './useSettings'
import { SOUND_LABELS, type SoundName } from './sound'

const labels: Record<keyof BoardColors, string> = {
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

// ponytail: color rows grouped so the panel scans, not sprawls
const groups: { title: string; keys: (keyof BoardColors)[] }[] = [
  { title: 'Board', keys: ['boardBase', 'boardField', 'triLight', 'triDark', 'exterior'] },
  { title: 'Pieces', keys: ['whitePiece', 'blackPiece', 'pieceBorder'] },
  { title: 'Dice', keys: ['diceBase', 'diceDots'] },
]

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

// ponytail: 2MB cap keeps the dataURL inside localStorage quota
function fileToAudio(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    if (file.size > 2 * 1024 * 1024) { reject(new Error('audio too big (max 2MB)')); return }
    const r = new FileReader()
    r.onload = () => resolve(String(r.result))
    r.onerror = () => reject(new Error('bad audio file'))
    r.readAsDataURL(file)
  })
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <details ref={el => { if (el) el.open = true }} style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      <summary style={{ fontSize: '0.7rem', letterSpacing: '0.1em', textTransform: 'uppercase', opacity: 0.55, margin: 0, cursor: 'pointer', userSelect: 'none' }}>{title}</summary>
      {children}
    </details>
  )
}

function ColorRow({ label, value, onChange, children }: { label: string; value: string; onChange: (v: string) => void; children?: ReactNode }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-start', gap: 8 }}>
      <input
        type="color"
        value={value}
        onChange={e => onChange(e.target.value)}
        style={{ width: 32, height: 28, padding: 0, border: '1px solid var(--line)', borderRadius: 6, cursor: 'pointer', background: 'transparent', flexShrink: 0 }}
        aria-label={label}
      />
      <span style={{ fontSize: '0.85rem', opacity: 0.9, flex: 1 }}>{label}</span>
      {children}
    </div>
  )
}

function Alpha({ value, onChange, label }: { value: number; onChange: (v: number) => void; label: string }) {
  return (
    <span style={{ display: 'flex', alignItems: 'center', gap: 4 }} title={`${label} opacity`}>
      <input
        type="range"
        min={0}
        max={100}
        value={value}
        onChange={e => onChange(Number(e.target.value))}
        style={{ width: 64, cursor: 'pointer' }}
        aria-label={`${label} opacity`}
      />
      <span style={{ fontSize: '0.75rem', opacity: 0.7, minWidth: 30 }}>{value}%</span>
    </span>
  )
}

function ImgBtn({ title, onFile, onClear, has }: { title: string; onFile: (f: File) => void; onClear: () => void; has: boolean }) {
  return (
    <span style={{ display: 'flex', alignItems: 'center', gap: 4 }} title={title}>
      <label className="btn small ghost" style={{ cursor: 'pointer', padding: '2px 6px' }}>
        🖼
        <input
          type="file"
          accept="image/*"
          hidden
          onChange={e => {
            const f = e.target.files?.[0]
            e.target.value = ''
            if (f) onFile(f)
          }}
        />
      </label>
      {has && (
        <button
          className="btn small ghost"
          style={{ padding: '2px 6px' }}
          onClick={onClear}
          title="Remove image"
        >
          ✕
        </button>
      )}
    </span>
  )
}

export function SettingsButton() {
  const { settings, setSettings, update, updateColor, reset, randomize } = useSettings()
  const [open, setOpen] = useState(false)
  const [audioErr, setAudioErr] = useState<string | null>(null)
  const [ioErr, setIoErr] = useState<string | null>(null)

  // ponytail: settings already hold images + sounds as dataURLs, plain JSON round-trips
  const exportSettings = () => {
    try {
      const blob = new Blob([JSON.stringify(settings)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'fair-backgammon-settings.json'
      a.click()
      setTimeout(() => URL.revokeObjectURL(url), 1000)
    } catch { setIoErr('export failed') }
  }
  const importSettings = (f: File) => {
    const r = new FileReader()
    r.onload = () => {
      try {
        const parsed = JSON.parse(String(r.result))
        if (!parsed || typeof parsed !== 'object') throw new Error()
        setSettings({ ...defaultSettings, ...parsed, colors: { ...defaultColors, ...parsed.colors } })
        setIoErr(null)
      } catch { setIoErr('bad settings file') }
    }
    r.onerror = () => setIoErr('bad settings file')
    r.readAsText(f)
  }

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
            width: 300,
            maxWidth: 'calc(100vw - 24px)',
            maxHeight: 'calc(100dvh - 70px)',
            overflowY: 'auto',
            background: 'var(--card)',
            border: '1px solid var(--line)',
            borderRadius: 12,
            padding: 12,
            display: 'flex',
            flexDirection: 'column',
            gap: 14,
            boxShadow: '0 12px 32px rgba(0,0,0,0.5)',
          }}
          onClick={e => e.stopPropagation()}
        >
          <Section title="Gameplay">
            <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: '0.9rem', cursor: 'pointer' }}>
              <input type="checkbox" checked={settings.swapClicks} onChange={e => update({ swapClicks: e.target.checked })} />
              Swap left / right click
            </label>
          </Section>

          <Section title="Sound">
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
            {(Object.keys(SOUND_LABELS) as SoundName[]).map(n => (
              <div key={n} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <span style={{ fontSize: '0.85rem', opacity: 0.9, flex: 1 }}>{SOUND_LABELS[n]}</span>
                <span style={{ fontSize: '0.75rem', opacity: 0.6 }}>{settings.customSounds?.[n] ? 'custom' : 'default'}</span>
                <label className="btn small ghost" style={{ cursor: 'pointer', padding: '2px 6px' }} title={`Upload custom ${SOUND_LABELS[n].toLowerCase()} sound`}>
                  🎵
                  <input
                    type="file"
                    accept="audio/*"
                    hidden
                    onChange={e => {
                      const f = e.target.files?.[0]
                      e.target.value = ''
                      if (f) fileToAudio(f).then(d => { setAudioErr(null); update({ customSounds: { ...settings.customSounds, [n]: d } }) }).catch((err: Error) => setAudioErr(err.message))
                    }}
                  />
                </label>
                {settings.customSounds?.[n] && (
                  <button
                    className="btn small ghost"
                    style={{ padding: '2px 6px' }}
                    onClick={() => { const next = { ...settings.customSounds }; delete next[n]; update({ customSounds: next }) }}
                    title="Restore default sound"
                  >
                    ✕
                  </button>
                )}
              </div>
            ))}
            {audioErr && <div className="hint" style={{ color: '#f87171' }}>{audioErr}</div>}
          </Section>

          {groups.map(g => (
            <Section key={g.title} title={g.title}>
              {g.keys.map(key => (
                <ColorRow key={key} label={labels[key]} value={settings.colors[key]} onChange={v => updateColor(key, v)}>
                  {key === 'boardField' && (
                    <>
                      <Alpha label="Field image" value={settings.boardFieldAlpha} onChange={v => update({ boardFieldAlpha: v })} />
                      <ImgBtn
                        title="Field image"
                        has={!!settings.boardFieldImage}
                        onFile={f => fileToFieldImage(f).then(d => update({ boardFieldImage: d })).catch(() => {})}
                        onClear={() => update({ boardFieldImage: null })}
                      />
                    </>
                  )}
                  {(key === 'whitePiece' || key === 'blackPiece') && (
                    <ImgBtn
                      title={key === 'whitePiece' ? 'White piece image' : 'Black piece image'}
                      has={!!(key === 'whitePiece' ? settings.whitePieceImage : settings.blackPieceImage)}
                      onFile={f => fileToFieldImage(f).then(d => update(key === 'whitePiece' ? { whitePieceImage: d } : { blackPieceImage: d })).catch(() => {})}
                      onClear={() => update(key === 'whitePiece' ? { whitePieceImage: null } : { blackPieceImage: null })}
                    />
                  )}
                  {key === 'diceBase' && (
                    <ImgBtn
                      title="Dice image"
                      has={!!settings.diceImage}
                      onFile={f => fileToFieldImage(f).then(d => update({ diceImage: d })).catch(() => {})}
                      onClear={() => update({ diceImage: null })}
                    />
                  )}
                </ColorRow>
              ))}
              {g.title === 'Board' && (
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }} title="Image over all triangles">
                  <span style={{ fontSize: '0.85rem', opacity: 0.9, flex: 1 }}>Triangles image</span>
                  <Alpha label="Triangles image" value={settings.triImageAlpha ?? 100} onChange={v => update({ triImageAlpha: v })} />
                  <ImgBtn
                    title="Triangles image"
                    has={!!settings.triImage}
                    onFile={f => fileToFieldImage(f).then(d => update({ triImage: d })).catch(() => {})}
                    onClear={() => update({ triImage: null })}
                  />
                </div>
              )}
            </Section>
          ))}

          <Section title="Backup">
            <div style={{ display: 'flex', gap: 8 }}>
              <button className="btn small ghost" onClick={exportSettings} style={{ flex: 1 }}>
                Export settings
              </button>
              <label className="btn small ghost" style={{ flex: 1, textAlign: 'center', cursor: 'pointer' }}>
                Import settings
                <input
                  type="file"
                  accept="application/json,.json"
                  hidden
                  onChange={e => {
                    const f = e.target.files?.[0]
                    e.target.value = ''
                    if (f) importSettings(f)
                  }}
                />
              </label>
            </div>
            {ioErr && <div className="hint" style={{ color: '#f87171' }}>{ioErr}</div>}
          </Section>

          <div style={{ display: 'flex', gap: 8 }}>
            <button className="btn small ghost" onClick={randomize} style={{ flex: 1 }}>
              Randomize colors
            </button>
            <button className="btn small ghost" onClick={reset} style={{ flex: 1 }}>
              Reset to default
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
