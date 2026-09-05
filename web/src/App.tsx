import { useState } from 'react'
import { Lobby } from './Lobby'
import { Board } from './Board'
import './index.css'

export default function App() {
  const [code, setCode] = useState<string | null>(() => localStorage.getItem('code'))
  const [user, setUser] = useState<string | null>(() => localStorage.getItem('user'))

  const enter = (c: string, u: string) => {
    localStorage.setItem('code', c)
    localStorage.setItem('user', u)
    setCode(c); setUser(u)
  }
  const leave = () => {
    localStorage.removeItem('code')
    setCode(null)
  }

  if (code && user) return <Board code={code} username={user} onLeave={leave} />
  return <Lobby onEnter={enter} />
}
