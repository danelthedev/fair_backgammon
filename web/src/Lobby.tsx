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
    <div className="lobby">
      <h1 className="title">fair_backgammon</h1>
      <p className="subtitle">buttery smooth. no lag. just play.</p>

      <input className="input user" placeholder="username" value={user} onChange={e => setUser(e.target.value)} maxLength={20} />

      <div className="cards">
        <div className="card">
          <button className="btn primary" onClick={handleCreate}>Create lobby</button>
          {created && (
            <div className="codeBox">
              <span className="code">{created}</span>
              <button className="btn small" onClick={() => navigator.clipboard.writeText(created)}>copy</button>
              <button className="btn small ghost" onClick={() => { onEnter(created, user.trim()) }}>enter →</button>
            </div>
          )}
        </div>

        <div className="card">
          <div className="joinRow">
            <input className="input codeInput" placeholder="CODE" value={code} onChange={e => setCode(e.target.value.toUpperCase())} maxLength={4} />
            <button className="btn" onClick={() => handleJoin()}>Join</button>
          </div>
          <span className="hint">code from friend, 4 chars</span>
        </div>
      </div>

      {err && <div className="error">{err}</div>}
    </div>
  )
}
