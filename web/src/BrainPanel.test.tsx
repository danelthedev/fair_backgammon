import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BrainPanel } from './BrainPanel'

describe('BrainPanel', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn(async () => ({
      ok: true,
      json: async () => ({ n: 2, coords: [[0.2, 0.3], null] }),
    })) as any)
  })

  it('renders the brain canvas', () => {
    render(<BrainPanel frame={null} />)
    expect(screen.getByLabelText('fly brain activity')).toBeTruthy()
  })

  it('accepts a brain frame without crashing', () => {
    render(<BrainPanel frame={{ units: [[0, 0.9]], edges: [] }} />)
    expect(screen.getByLabelText('fly brain activity')).toBeTruthy()
  })
})
