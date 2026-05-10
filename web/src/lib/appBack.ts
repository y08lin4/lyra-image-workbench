type AppBackReason = 'android-back' | 'edge-swipe' | 'browser-back' | string
type AppBackHandler = (reason: AppBackReason) => boolean | void

declare global {
  interface Window {
    __LyAIAppBack?: (reason?: AppBackReason) => boolean
  }
}

const handlers: AppBackHandler[] = []
let globalInstalled = false
let edgeGestureInstalled = false

export function registerAppBackHandler(handler: AppBackHandler) {
  handlers.push(handler)
  ensureAppBackBridge()
  return () => {
    const index = handlers.lastIndexOf(handler)
    if (index >= 0) handlers.splice(index, 1)
  }
}

export function dispatchAppBack(reason: AppBackReason = 'browser-back') {
  for (let index = handlers.length - 1; index >= 0; index -= 1) {
    const handled = handlers[index]?.(reason)
    if (handled !== false) return true
  }
  return false
}

export function ensureAppBackBridge() {
  if (globalInstalled) return
  window.__LyAIAppBack = (reason = 'browser-back') => dispatchAppBack(reason)
  globalInstalled = true
}

export function installEdgeBackGesture() {
  ensureAppBackBridge()
  if (edgeGestureInstalled) return () => undefined
  edgeGestureInstalled = true

  let tracking = false
  let triggered = false
  let startX = 0
  let startY = 0

  const reset = () => {
    tracking = false
    triggered = false
    startX = 0
    startY = 0
  }

  const onTouchStart = (event: TouchEvent) => {
    if (event.touches.length !== 1) return reset()
    const touch = event.touches[0]
    tracking = touch.clientX <= 28
    triggered = false
    startX = touch.clientX
    startY = touch.clientY
  }

  const onTouchMove = (event: TouchEvent) => {
    if (!tracking || triggered || event.touches.length !== 1) return
    const touch = event.touches[0]
    const dx = touch.clientX - startX
    const dy = Math.abs(touch.clientY - startY)
    if (dx > 72 && dy < 80) {
      triggered = true
      event.preventDefault()
      dispatchAppBack('edge-swipe')
    }
  }

  document.addEventListener('touchstart', onTouchStart, { passive: true })
  document.addEventListener('touchmove', onTouchMove, { passive: false })
  document.addEventListener('touchend', reset, { passive: true })
  document.addEventListener('touchcancel', reset, { passive: true })

  return () => {
    document.removeEventListener('touchstart', onTouchStart)
    document.removeEventListener('touchmove', onTouchMove)
    document.removeEventListener('touchend', reset)
    document.removeEventListener('touchcancel', reset)
    edgeGestureInstalled = false
    reset()
  }
}
