import '@testing-library/jest-dom'

// ponytail: jsdom has no Audio; stub so sound.ts plays silent
if (typeof (globalThis as any).Audio === 'undefined') {
  ;(globalThis as any).Audio = class {
    currentTime = 0
    preload = ''
    load() {}
    play() { return Promise.resolve() }
  }
}
