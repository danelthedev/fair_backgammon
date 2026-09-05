import { createContext, useContext, useEffect, useState } from 'react'
import type { ReactNode } from 'react'

export type BoardColors = {
  boardBase: string
  boardField: string
  triLight: string
  triDark: string
  exterior: string
  whitePiece: string
  blackPiece: string
}

export const defaultColors: BoardColors = {
  boardBase: '#3e2723',
  boardField: '#0f6b3d',
  triLight: '#f5e6c8',
  triDark: '#8b5a2b',
  exterior: '#2b1a0e',
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

type Ctx = {
  settings: Settings
  setSettings: React.Dispatch<React.SetStateAction<Settings>>
  reset: () => void
  update: (patch: Partial<Settings>) => void
  updateColor: (key: keyof BoardColors, value: string) => void
  defaultColors: BoardColors
}

const SettingsContext = createContext<Ctx | null>(null)

export function SettingsProvider({ children }: { children: ReactNode }) {
  const [settings, setSettings] = useState<Settings>(() => {
    try {
      const raw = localStorage.getItem(KEY)
      if (raw) return { ...defaultSettings, ...JSON.parse(raw), colors: { ...defaultColors, ...JSON.parse(raw).colors } }
    } catch {}
    return defaultSettings
  })

  useEffect(() => {
    localStorage.setItem(KEY, JSON.stringify(settings))
    const r = document.documentElement
    r.style.setProperty('--board-base', settings.colors.boardBase)
    r.style.setProperty('--board-field', settings.colors.boardField)
    r.style.setProperty('--tri-light', settings.colors.triLight)
    r.style.setProperty('--tri-dark', settings.colors.triDark)
    r.style.setProperty('--exterior', settings.colors.exterior)
    r.style.setProperty('--white-piece', settings.colors.whitePiece)
    r.style.setProperty('--black-piece', settings.colors.blackPiece)
  }, [settings])

  useEffect(() => {
    const onStorage = (e: StorageEvent) => {
      if (e.key === KEY && e.newValue) {
        try { setSettings({ ...defaultSettings, ...JSON.parse(e.newValue), colors: { ...defaultColors, ...JSON.parse(e.newValue).colors } }) } catch {}
      }
    }
    window.addEventListener('storage', onStorage)
    return () => window.removeEventListener('storage', onStorage)
  }, [])

  // apply on mount
  useEffect(() => {
    const r = document.documentElement
    r.style.setProperty('--board-base', settings.colors.boardBase)
    r.style.setProperty('--board-field', settings.colors.boardField)
    r.style.setProperty('--tri-light', settings.colors.triLight)
    r.style.setProperty('--tri-dark', settings.colors.triDark)
    r.style.setProperty('--exterior', settings.colors.exterior)
    r.style.setProperty('--white-piece', settings.colors.whitePiece)
    r.style.setProperty('--black-piece', settings.colors.blackPiece)
  }, [])

  const reset = () => setSettings(defaultSettings)
  const update = (patch: Partial<Settings>) => setSettings(s => ({ ...s, ...patch }))
  const updateColor = (key: keyof BoardColors, value: string) =>
    setSettings(s => ({ ...s, colors: { ...s.colors, [key]: value } }))

  return <SettingsContext.Provider value={{ settings, setSettings, reset, update, updateColor, defaultColors }}>{children}</SettingsContext.Provider>
}

export function useSettings() {
  const ctx = useContext(SettingsContext)
  if (ctx) return ctx
  // fallback for tests without provider
  return {
    settings: defaultSettings,
    setSettings: (() => {}) as any,
    reset: () => {},
    update: () => {},
    updateColor: () => {},
    defaultColors,
  }
}
