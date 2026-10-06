import { useState } from 'react'
import { createLobby, createVsFly, joinLobby, setUsername } from './api'

// Bot registry: add future bots here; the dropdown renders from this list.
const BOTS = [
  { id: 'retarded', label: 'Fruit Fly Brain' },
]

export function Lobby({ onEnter }: { onEnter: (code: string, user: string) => void }) {
  const [user, setUser] = useState(() => localStorage.getItem('user') ?? '')
  const [code, setCode] = useState('')
  const [created, setCreated] = useState<string | null>(null)
  const [err, setErr] = useState<string | null>(null)
  const [variant, setVariant] = useState('retarded')
  const [showMods, setShowMods] = useState(false)
  const [negative, setNegative] = useState(false)
  const [bigDice, setBigDice] = useState(false)
  const [maxDie, setMaxDie] = useState(9)
  const ensureUser = async () => {
    if (!user.trim()) throw new Error('enter username')
    await setUsername(user.trim())
  }

  const handleCreate = async () => {
    try {
      await ensureUser()
      const c = await createLobby({ negative, maxDie: bigDice ? maxDie : 0 })
      setCreated(c)
      setErr(null)
      onEnter(c, user.trim())
    } catch (e: any) { setErr(e.message) }
  }

  const handleJoin = async (c = code) => {
    try {
      await ensureUser()
      const up = c.trim().toUpperCase()
      if (!up) throw new Error('enter code')
      await joinLobby(up)
      onEnter(up, user.trim())
    } catch (e: any) { setErr(e.message) }
  }

  const handleVsFly = async () => {
    try {
      await ensureUser()
      const c = await createVsFly(variant)
      setErr(null)
      onEnter(c, user.trim())
    } catch (e: any) { setErr(e.message) }
  }

  return (
    <div className="lobby" style={{ marginTop: '3vh' }}>
      <input className="input user" placeholder="username" value={user} onChange={e => { setUser(e.target.value); localStorage.setItem('user', e.target.value) }} maxLength={20} style={{ position: 'fixed', top: 12, left: 12, width: 200, textAlign: 'left', margin: 0, zIndex: 100 }} />
      <h1 className="title" style={{ textAlign: 'center' }}>Fair Backgammon</h1>


      <div className="modsWrap">
        <div className="cards" style={{ gridTemplateColumns: '1fr', maxWidth: 340, margin: 0, flex: '0 1 340px' }}>
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8 }}>
            <div style={{ display: 'flex', gap: 8, width: '100%' }}>
              <button className="btn primary large" style={{ flex: 1 }} onClick={handleCreate}>Create lobby</button>
              <button className="btn" style={{ fontSize: '.72rem', lineHeight: 1.2, padding: '.45rem .6rem', minWidth: 62, textAlign: 'center' }} onClick={() => setShowMods(v => !v)}>Add<br />mods</button>
            </div>
            {created && (
              <div className="codeBox">
                <span className="code">{created}</span>
                <button className="btn small" onClick={() => navigator.clipboard.writeText(created)}>copy</button>
                <button className="btn small ghost" onClick={() => { onEnter(created, user.trim()) }}>enter →</button>
              </div>
            )}
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            <div className="joinRow">
              <input className="input codeInput" placeholder="CODE" value={code} onChange={e => setCode(e.target.value.toUpperCase())} maxLength={4} />
              <button className="btn" onClick={() => handleJoin()}>Join</button>
            </div>
            <div className="joinRow">
              <select className="input codeInput" value={variant} onChange={e => setVariant(e.target.value)} aria-label="bot">
                {BOTS.map(b => <option key={b.id} value={b.id}>{b.label}</option>)}
              </select>
              <button className="btn primary" onClick={handleVsFly}>Play vs Bot</button>
            </div>
          </div>
        </div>

        {showMods && (
          <div className="card modsCard">
            <label className={`modsOpt ${negative ? 'on' : ''}`}>
              <input type="checkbox" checked={negative} onChange={e => setNegative(e.target.checked)} />
              <span>
                <strong>Negative dice</strong>
                <small>pieces can move backwards</small>
              </span>
            </label>
            <label className={`modsOpt ${bigDice ? 'on' : ''}`}>
              <input type="checkbox" checked={bigDice} onChange={e => setBigDice(e.target.checked)} />
              <span>
                <strong>Bigger dice</strong>
                <small>raise max face above 6</small>
              </span>
            </label>
            {bigDice && (
              <div className="modsMax">
                <span>Max value</span>
                <span className="stepper">
                  <button className="btn small" onClick={() => setMaxDie(v => Math.max(7, v - 1))} disabled={maxDie <= 7} aria-label="decrease max die">−</button>
                  <strong>{maxDie}</strong>
                  <button className="btn small" onClick={() => setMaxDie(v => Math.min(20, v + 1))} disabled={maxDie >= 20} aria-label="increase max die">+</button>
                </span>
              </div>
            )}
          </div>
        )}
      </div>

      {err && <div className="error" style={{ maxWidth: 420, marginLeft: 'auto', marginRight: 'auto' }}>{err}</div>}
    </div>
  )
}
