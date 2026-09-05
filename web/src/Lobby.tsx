import { useState } from 'react'
import { createLobby, joinLobby, setUsername } from './api'

export function Lobby({ onEnter }: { onEnter: (code: string, user: string) => void }) {
  const [user, setUser] = useState('')
  const [code, setCode] = useState('')
  const [created, setCreated] = useState<string | null>(null)
  const [err, setErr] = useState<string | null>(null)

  const ensureUser = async () => {
    if (!user.trim()) throw new Error('enter username')
    await setUsername(user.trim())
  }

  const handleCreate = async () => {
    try {
      await ensureUser()
      const c = await createLobby()
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

  return (
    <div className="lobby" style={{ marginTop: '3vh' }}>
      <input className="input user" placeholder="username" value={user} onChange={e => setUser(e.target.value)} maxLength={20} style={{ position: 'fixed', top: 12, left: 12, width: 200, textAlign: 'left', margin: 0, zIndex: 100 }} />
      <h1 className="title" style={{ textAlign: 'center' }}>Fair Backgammon</h1>


      <div className="cards" style={{ gridTemplateColumns: '1fr', maxWidth: 420, marginLeft: 'auto', marginRight: 'auto', marginTop: '2.5rem' }}>
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8 }}>
          <button className="btn primary large" style={{ width: '100%' }} onClick={handleCreate}>Create lobby</button>
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
        </div>
      </div>

      {err && <div className="error" style={{ maxWidth: 420, marginLeft: 'auto', marginRight: 'auto' }}>{err}</div>}
    </div>
  )
}
