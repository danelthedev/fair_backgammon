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
  boardBase: '#7a3f2a',
  boardField: '#e19247',
  triLight: '#f4b862',
  triDark: '#9f452d',
  exterior: '#3e2723',
  whitePiece: '#fffaf0',
  blackPiece: '#1e1e1e',
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
      if (raw) {
        const parsed = JSON.parse(raw)
        // migrate old palettes to exact Live colors
        const old = parsed.colors
        const isOld = old?.triLight !== '#f4b862' || old?.triDark !== '#9f452d' || old?.boardField !== '#e19247'
        if (old?.boardField === '#0f6b3d' || old?.boardField === '#e8c9a6' || old?.boardField === '#e8b67a' || old?.boardBase === '#3e2723' || old?.boardBase === '#8d6e63' || old?.boardBase === '#7a3f2a') {
          if (isOld) return defaultSettings
        }
        return { ...defaultSettings, ...parsed, colors: { ...defaultColors, ...parsed.colors } }
      }
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
