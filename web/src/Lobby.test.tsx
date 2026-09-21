import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { Lobby } from './Lobby'

describe('Lobby vs fly', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.unstubAllGlobals()
    vi.stubGlobal('fetch', vi.fn(async (url: any, opts: any) => {
      if (String(url).includes('/api/session')) return { ok: true, text: async () => '' }
      if (String(url).includes('/vs-fly')) {
        return { ok: true, json: async () => ({ code: 'AB12', variant: JSON.parse(opts.body).variant }) }
      }
      throw new Error('unexpected ' + url)
    }))
  })

  it('creates vs-bot room with chosen bot', async () => {
    const seen: string[] = []
    ;(fetch as any).mockImplementation(async (url: any, opts: any) => {
      if (String(url).includes('/api/session')) return { ok: true, text: async () => '' }
      seen.push(JSON.parse(opts.body).variant)
      return { ok: true, json: async () => ({ code: 'AB12' }) }
    })
    const onEnter = vi.fn()
    render(<Lobby onEnter={onEnter} />)
    fireEvent.change(screen.getByPlaceholderText('username'), { target: { value: 'alice' } })
    fireEvent.click(screen.getByText('Play vs Bot'))
    await waitFor(() => expect(onEnter).toHaveBeenCalledWith('AB12', 'alice'))
    expect(seen).toEqual(['retarded'])
  })
})
