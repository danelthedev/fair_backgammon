// ponytail: one Audio per file, unlocked on first gesture (autoplay policy)
export type SoundName = 'move' | 'dice' | 'turn' | 'capture'
export const SOUND_LABELS: Record<SoundName, string> = { move: 'Piece move', dice: 'Dice roll', turn: 'Your turn', capture: 'Capture' }
const files: Record<SoundName, string> = {
  move: '/sounds/piece_move.mp3',
  dice: '/sounds/dice_roll.mp3',
  turn: '/sounds/turn_start_notification.mp3',
  capture: '/sounds/captured_piece.mp3',
}

// ponytail: per-sound gain, dice file runs hot
const volume: Record<string, number> = { dice: 0.35 }
const cache = new Map<string, HTMLAudioElement>()
let unlocked = false

function unlock() {
  if (unlocked) return
  unlocked = true
  for (const [k, src] of Object.entries(files)) {
    try {
      const a = new Audio(src)
      a.preload = 'auto'
      a.load()
      cache.set(k, a)
    } catch {}
  }
}

if (typeof window !== 'undefined') {
  window.addEventListener('pointerdown', unlock, { once: true })
  window.addEventListener('keydown', unlock, { once: true })
}

function play(name: SoundName, enabled: boolean, master = 1, custom?: string | null) {
  if (!enabled || master <= 0) return
  try {
    unlock()
    const key = custom ? `custom:${name}` : name
    let a = cache.get(key)
    if (!a) {
      a = new Audio(custom || files[name])
      cache.set(key, a)
    } else if (custom && a.src !== custom) a.src = custom
    a.currentTime = 0
    a.volume = Math.min(1, (volume[name] ?? 1) * master)
    void a.play().catch(() => {})
  } catch {}
}

export const playMove = (enabled: boolean, master = 1, custom?: string | null) => play('move', enabled, master, custom)
export const playDice = (enabled: boolean, master = 1, custom?: string | null) => play('dice', enabled, master, custom)
export const playTurn = (enabled: boolean, master = 1, custom?: string | null) => play('turn', enabled, master, custom)
export const playCapture = (enabled: boolean, master = 1, custom?: string | null) => play('capture', enabled, master, custom)
