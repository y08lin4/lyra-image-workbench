import { type CSSProperties, type PointerEvent as ReactPointerEvent, type SyntheticEvent, useEffect, useMemo, useRef, useState } from 'react'
import { registerAppBackHandler } from '../../lib/appBack'

export type ImageDimensions = { width: number; height: number }

type StageSize = { width: number; height: number }
type ViewState = { scale: number; offsetX: number; offsetY: number }
type PointerPoint = { x: number; y: number }
type GestureState = {
  moved: boolean
  startX: number
  startY: number
  startOffsetX: number
  startOffsetY: number
  startScale: number
  startDistance: number
}

type Props = {
  src: string
  title: string
  onDimensions: (dimensions: ImageDimensions) => void
  onClose: () => void
}

const DEFAULT_VIEW: ViewState = { scale: 1, offsetX: 0, offsetY: 0 }
const MAX_SCALE = 4

export function PreviewImageStage({ src, title, onDimensions, onClose }: Props) {
  const stageRef = useRef<HTMLDivElement | null>(null)
  const pointersRef = useRef(new Map<number, PointerPoint>())
  const gestureRef = useRef<GestureState>(emptyGesture())
  const [stageSize, setStageSize] = useState<StageSize>({ width: 0, height: 0 })
  const [dimensions, setDimensions] = useState<ImageDimensions>()
  const [view, setView] = useState<ViewState>(DEFAULT_VIEW)
  const stageSizeRef = useRef(stageSize)
  const dimensionsRef = useRef(dimensions)
  const viewRef = useRef(view)

  stageSizeRef.current = stageSize
  dimensionsRef.current = dimensions
  viewRef.current = view

  useEffect(() => {
    setDimensions(undefined)
    setView(DEFAULT_VIEW)
    pointersRef.current.clear()
    gestureRef.current = emptyGesture()
  }, [src])

  useEffect(() => {
    const node = stageRef.current
    if (!node) return

    const measure = () => {
      const rect = node.getBoundingClientRect()
      setStageSize({
        width: Math.max(0, Math.floor(rect.width)),
        height: Math.max(0, Math.floor(rect.height)),
      })
    }

    measure()
    if (typeof ResizeObserver !== 'undefined') {
      const observer = new ResizeObserver(measure)
      observer.observe(node)
      return () => observer.disconnect()
    }

    window.addEventListener('resize', measure)
    window.visualViewport?.addEventListener('resize', measure)
    return () => {
      window.removeEventListener('resize', measure)
      window.visualViewport?.removeEventListener('resize', measure)
    }
  }, [])

  useEffect(() => {
    setView((current) => clampView(current, dimensionsRef.current, stageSizeRef.current))
  }, [stageSize.width, stageSize.height, dimensions?.width, dimensions?.height])

  useEffect(() => registerAppBackHandler(() => {
    if (isViewChanged(viewRef.current)) {
      setView(DEFAULT_VIEW)
      return true
    }
    onClose()
    return true
  }), [onClose])

  const fitSize = useMemo(() => getFitSize(dimensions, stageSize), [dimensions, stageSize])
  const imageStyle = useMemo(() => getImageStyle(fitSize, view), [fitSize, view])

  function handleLoad(event: SyntheticEvent<HTMLImageElement>) {
    const next = {
      width: event.currentTarget.naturalWidth,
      height: event.currentTarget.naturalHeight,
    }
    setDimensions(next)
    onDimensions(next)
  }

  function setClampedView(next: ViewState) {
    setView(clampView(next, dimensionsRef.current, stageSizeRef.current))
  }

  function handlePointerDown(event: ReactPointerEvent<HTMLDivElement>) {
    if (!dimensionsRef.current) return
    event.currentTarget.setPointerCapture?.(event.pointerId)
    pointersRef.current.set(event.pointerId, { x: event.clientX, y: event.clientY })

    const pointers = Array.from(pointersRef.current.values())
    const current = viewRef.current
    gestureRef.current = {
      moved: false,
      startX: event.clientX,
      startY: event.clientY,
      startOffsetX: current.offsetX,
      startOffsetY: current.offsetY,
      startScale: current.scale,
      startDistance: pointers.length >= 2 ? pointerDistance(pointers[0], pointers[1]) : 0,
    }
  }

  function handlePointerMove(event: ReactPointerEvent<HTMLDivElement>) {
    if (!pointersRef.current.has(event.pointerId)) return
    pointersRef.current.set(event.pointerId, { x: event.clientX, y: event.clientY })
    const pointers = Array.from(pointersRef.current.values())
    const gesture = gestureRef.current

    if (pointers.length >= 2 && gesture.startDistance > 0) {
      const distance = pointerDistance(pointers[0], pointers[1])
      const scale = clampScale(gesture.startScale * (distance / gesture.startDistance))
      setClampedView({ ...viewRef.current, scale })
      gesture.moved = true
      return
    }

    if (pointers.length !== 1) return
    const dx = event.clientX - gesture.startX
    const dy = event.clientY - gesture.startY
    if (Math.hypot(dx, dy) > 5) gesture.moved = true
    if (viewRef.current.scale <= 1) return

    setClampedView({
      ...viewRef.current,
      offsetX: gesture.startOffsetX + dx,
      offsetY: gesture.startOffsetY + dy,
    })
  }

  function handlePointerUp(event: ReactPointerEvent<HTMLDivElement>) {
    const wasMoved = gestureRef.current.moved
    try {
      event.currentTarget.releasePointerCapture?.(event.pointerId)
    } catch {
      // ignore release errors from browsers that already released the pointer
    }
    pointersRef.current.delete(event.pointerId)

    const remaining = Array.from(pointersRef.current.values())
    if (remaining.length === 1) {
      const current = viewRef.current
      gestureRef.current = {
        moved: false,
        startX: remaining[0].x,
        startY: remaining[0].y,
        startOffsetX: current.offsetX,
        startOffsetY: current.offsetY,
        startScale: current.scale,
        startDistance: 0,
      }
      return
    }

    if (remaining.length === 0) {
      pointersRef.current.clear()
      gestureRef.current = emptyGesture()
      if (!wasMoved && event.target instanceof HTMLImageElement) onClose()
    }
  }

  function handlePointerCancel() {
    pointersRef.current.clear()
    gestureRef.current = emptyGesture()
  }

  return (
    <div
      ref={stageRef}
      className="image-preview-stage"
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
      onPointerCancel={handlePointerCancel}
      onContextMenu={(event) => event.preventDefault()}
    >
      <img className="image-preview-img" src={src} alt={title} style={imageStyle} onLoad={handleLoad} draggable={false} />
    </div>
  )
}

function getImageStyle(fitSize: StageSize | undefined, view: ViewState): CSSProperties | undefined {
  if (!fitSize?.width || !fitSize.height) return undefined
  return {
    width: `${fitSize.width}px`,
    height: `${fitSize.height}px`,
    transform: `translate3d(${view.offsetX}px, ${view.offsetY}px, 0) scale(${view.scale})`,
    cursor: view.scale > 1 ? 'grab' : 'zoom-out',
  }
}

function getFitSize(dimensions: ImageDimensions | undefined, stageSize: StageSize): StageSize | undefined {
  if (!dimensions?.width || !dimensions.height || !stageSize.width || !stageSize.height) return undefined
  const scale = Math.min(stageSize.width / dimensions.width, stageSize.height / dimensions.height, 1)
  return {
    width: Math.max(1, Math.floor(dimensions.width * scale)),
    height: Math.max(1, Math.floor(dimensions.height * scale)),
  }
}

function clampView(view: ViewState, dimensions: ImageDimensions | undefined, stageSize: StageSize): ViewState {
  const scale = clampScale(view.scale)
  const fitSize = getFitSize(dimensions, stageSize)
  if (!fitSize || scale <= 1) return DEFAULT_VIEW
  const maxX = Math.max(0, (fitSize.width * scale - stageSize.width) / 2)
  const maxY = Math.max(0, (fitSize.height * scale - stageSize.height) / 2)
  return {
    scale,
    offsetX: clamp(view.offsetX, -maxX, maxX),
    offsetY: clamp(view.offsetY, -maxY, maxY),
  }
}

function isViewChanged(view: ViewState) {
  return Math.abs(view.scale - 1) > 0.01 || Math.abs(view.offsetX) > 1 || Math.abs(view.offsetY) > 1
}

function pointerDistance(a: PointerPoint, b: PointerPoint) {
  return Math.hypot(a.x - b.x, a.y - b.y)
}

function clampScale(value: number) {
  return clamp(value, 1, MAX_SCALE)
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}

function emptyGesture(): GestureState {
  return { moved: false, startX: 0, startY: 0, startOffsetX: 0, startOffsetY: 0, startScale: 1, startDistance: 0 }
}
