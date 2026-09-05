import { useState } from 'react'
import { leaveLobby } from './api'
import { Lobby } from './Lobby'
import { Board } from './Board'
import { SettingsButton } from './Settings'
import { SettingsProvider } from './useSettings'
import './index.css'
export default function App() {
  const [code, setCode] = useState<string | null>(null)
  const [user, setUser] = useState<string | null>(() => localStorage.getItem('user'))

  const enter = (c: string, u: string) => {
    localStorage.setItem('user', u)
    setCode(c)
    setUser(u)
  }
  const leave = async () => {
    if (code) {
      await leaveLobby(code).catch(() => {})
    }
    setCode(null)
  }

  if (code && user)
    return (
      <SettingsProvider>
        <SettingsButton />
        <Board code={code} username={user} onLeave={leave} />
      </SettingsProvider>
    )
  return (
    <SettingsProvider>
      <SettingsButton />
      <Lobby onEnter={enter} />
    </SettingsProvider>
  )
}
