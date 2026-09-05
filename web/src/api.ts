export async function setUsername(name: string) {
  const r = await fetch('/api/session', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ username: name }),
  })
  if (!r.ok) throw new Error(await r.text())
}

export async function createLobby(): Promise<string> {
  const r = await fetch('/api/lobby', { method: 'POST', credentials: 'include' })
  if (!r.ok) throw new Error(await r.text())
  const j = await r.json()
  return j.code
}

export async function joinLobby(code: string): Promise<void> {
  const r = await fetch(`/api/lobby/${code}/join`, { method: 'POST', credentials: 'include' })
  if (!r.ok) throw new Error(await r.text())
}
