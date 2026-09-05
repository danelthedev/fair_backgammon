import { useEffect, useState } from 'react'

export type BoardColors = {
  boardBase: string
  boardField: string
  triLight: string
  triDark: string
  exterior: string
  accent: string
  whitePiece: string
  blackPiece: string
}

export const defaultColors: BoardColors = {
  boardBase: '#3e2723',
  boardField: '#0f6b3d',
  triLight: '#f5e6c8',
  triDark: '#8b5a2b',
  exterior: '#2b1a0e',
  accent: '#f59e0b',
  whitePiece: '#f8fafc',
  blackPiece: '#0f0f0f',
}

export type Settings = {
  swapClicks: boolean
  colors: BoardColors
}

const defaultSettings: Settings = {
  swapClicks: false,
  colors: defaultColors,
}

const KEY = 'fair_backgammon_settings'

export function useSettings() {
  const [settings, setSettings] = useState<Settings>(() => {
    try {
      const raw = localStorage.getItem(KEY)
      if (raw) return { ...defaultSettings, ...JSON.parse(raw), colors: { ...defaultColors, ...JSON.parse(raw).colors } }
    } catch {}
    return defaultSettings
  })

  useEffect(() => {
    localStorage.setItem(KEY, JSON.stringify(settings))
    // apply CSS variables immediately
    const r = document.documentElement
    r.style.setProperty('--board-base', settings.colors.boardBase)
    r.style.setProperty('--board-field', settings.colors.boardField)
    r.style.setProperty('--tri-light', settings.colors.triLight)
    r.style.setProperty('--tri-dark', settings.colors.triDark)
    r.style.setProperty('--exterior', settings.colors.exterior)
    r.style.setProperty('--accent', settings.colors.accent)
    // pieces via variables too
    r.style.setProperty('--white-piece', settings.colors.whitePiece)
    r.style.setProperty('--black-piece', settings.colors.blackPiece)
  }, [settings])

  // apply on mount
  useEffect(() => {
    const r = document.documentElement
    r.style.setProperty('--board-base', settings.colors.boardBase)
    r.style.setProperty('--board-field', settings.colors.boardField)
    r.style.setProperty('--tri-light', settings.colors.triLight)
    r.style.setProperty('--tri-dark', settings.colors.triDark)
    r.style.setProperty('--exterior', settings.colors.exterior)
    r.style.setProperty('--accent', settings.colors.accent)
    r.style.setProperty('--white-piece', settings.colors.whitePiece)
    r.style.setProperty('--black-piece', settings.colors.blackPiece)
  }, [])

  const reset = () => setSettings(defaultSettings)
  const update = (patch: Partial<Settings>) => setSettings(s => ({ ...s, ...patch }))
  const updateColor = (key: keyof BoardColors, value: string) =>
    setSettings(s => ({ ...s, colors: { ...s.colors, [key]: value } }))

  return { settings, setSettings, reset, update, updateColor, defaultColors }
}
