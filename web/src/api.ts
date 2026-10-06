export async function setUsername(name: string) {
  const r = await fetch('/api/session', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ username: name }),
  })
  if (!r.ok) throw new Error(await r.text())
}

export type LobbyMods = { negative?: boolean; maxDie?: number; powers?: { reroll: number; skip: number; protect: number } }
export async function createLobby(mods?: LobbyMods): Promise<string> {
  const r = await fetch('/api/lobby', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(mods ?? {}),
  })
  if (!r.ok) throw new Error(await r.text())
  const j = await r.json()
  return j.code
}
export async function leaveLobby(code: string): Promise<void> {
  await fetch(`/api/lobby/${code}/leave`, { method: 'POST', credentials: 'include' }).catch(() => {})
}


export async function joinLobby(code: string): Promise<void> {
  const r = await fetch(`/api/lobby/${code}/join`, { method: 'POST', credentials: 'include' })
  if (!r.ok) throw new Error(await r.text())
}

export async function createVsFly(variant: string): Promise<string> {
  const r = await fetch('/api/lobby/vs-fly', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ variant }),
  })
  if (!r.ok) throw new Error(await r.text())
  const j = await r.json()
  return j.code
}
