// ponytail: one Audio per file, unlocked on first gesture (autoplay policy)
const files: Record<string, string> = {
  move: '/sounds/piece_move.mp3',
  dice: '/sounds/dice_roll.mp3',
  turn: '/sounds/turn_start_notification.mp3',
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

function play(name: string, enabled: boolean, master = 1) {
  if (!enabled || master <= 0) return
  try {
    unlock()
    let a = cache.get(name)
    if (!a) {
      a = new Audio(files[name])
      cache.set(name, a)
    }
    a.currentTime = 0
    a.volume = Math.min(1, (volume[name] ?? 1) * master)
    void a.play().catch(() => {})
  } catch {}
}

export const playMove = (enabled: boolean, master = 1) => play('move', enabled, master)
export const playDice = (enabled: boolean, master = 1) => play('dice', enabled, master)
export const playTurn = (enabled: boolean, master = 1) => play('turn', enabled, master)
