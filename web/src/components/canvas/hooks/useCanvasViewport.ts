import { type CSSProperties, type PointerEvent as ReactPointerEvent, type RefCallback, type WheelEvent as ReactWheelEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  DEFAULT_CANVAS_VIEWPORT,
  DEFAULT_CANVAS_VIEWPORT_LIMITS,
  canvasPointToViewportPoint,
  fitView as fitViewport,
  normalizeViewport,
  panViewport,
  resetView as resetViewport,
  viewportCssTransform,
  viewportPointFromClient,
  viewportPointToCanvasPoint,
  wheelViewportState,
  zoomViewportAtPoint,
  type CanvasBounds,
  type CanvasPanState,
  type CanvasPoint,
  type CanvasViewport,
  type CanvasViewportLimits,
  type CanvasWheelMode,
  type CanvasWheelState,
  type FitViewOptions,
  type ViewportPoint,
  type ViewportSize,
} from '../model/viewport'

export type CanvasViewportUpdate = CanvasViewport | ((current: CanvasViewport) => CanvasViewport)

export type UseCanvasViewportOptions = Partial<CanvasViewportLimits> & {
  initialViewport?: CanvasViewport
  wheelMode?: CanvasWheelMode | 'auto'
  wheelZoomIntensity?: number
}

export type UseCanvasViewportResult = {
  viewport: CanvasViewport
  viewportRef: React.MutableRefObject<CanvasViewport>
  viewportSize: ViewportSize
  panState: CanvasPanState | null
  containerRef: RefCallback<HTMLElement>
  containerElementRef: React.MutableRefObject<HTMLElement | null>
  viewportStyle: CSSProperties
  setViewport: (next: CanvasViewportUpdate) => void
  resetView: (nextViewport?: CanvasViewport) => void
  fitView: (bounds: CanvasBounds | null | undefined, options?: FitViewOptions) => void
  panBy: (delta: ViewportPoint) => void
  zoomAt: (anchor: ViewportPoint, requestedZoom: number) => void
  clientToViewportPoint: (clientX: number, clientY: number) => ViewportPoint
  clientToCanvasPoint: (clientX: number, clientY: number) => CanvasPoint
  viewportPointToCanvasPoint: (point: ViewportPoint) => CanvasPoint
  canvasPointToViewportPoint: (point: CanvasPoint) => ViewportPoint
  handleWheel: (event: ReactWheelEvent<HTMLElement>) => CanvasWheelState
  beginPan: (event: ReactPointerEvent<HTMLElement>) => CanvasPanState
  updatePan: (event: ReactPointerEvent<HTMLElement>) => CanvasViewport | null
  endPan: (event: ReactPointerEvent<HTMLElement>) => CanvasPanState | null
  cancelPan: () => CanvasPanState | null
}

const EMPTY_SIZE: ViewportSize = { width: 0, height: 0 }

export function useCanvasViewport(options: UseCanvasViewportOptions = {}): UseCanvasViewportResult {
  const limits = useMemo<CanvasViewportLimits>(() => ({
    minZoom: options.minZoom ?? DEFAULT_CANVAS_VIEWPORT_LIMITS.minZoom,
    maxZoom: options.maxZoom ?? DEFAULT_CANVAS_VIEWPORT_LIMITS.maxZoom,
  }), [options.maxZoom, options.minZoom])
  const initialViewport = useMemo(
    () => normalizeViewport(options.initialViewport ?? DEFAULT_CANVAS_VIEWPORT, limits),
    [limits, options.initialViewport],
  )
  const [viewport, setViewportState] = useState(initialViewport)
  const [viewportSize, setViewportSize] = useState<ViewportSize>(EMPTY_SIZE)
  const [containerElement, setContainerElement] = useState<HTMLElement | null>(null)
  const [panState, setPanState] = useState<CanvasPanState | null>(null)

  const viewportRef = useRef(viewport)
  const viewportSizeRef = useRef(viewportSize)
  const containerElementRef = useRef<HTMLElement | null>(null)
  const panStateRef = useRef<CanvasPanState | null>(null)
  const limitsRef = useRef(limits)
  const initialViewportRef = useRef(initialViewport)
  const wheelOptionsRef = useRef({
    mode: options.wheelMode ?? 'auto',
    zoomIntensity: options.wheelZoomIntensity,
  })

  viewportRef.current = viewport
  viewportSizeRef.current = viewportSize
  limitsRef.current = limits
  initialViewportRef.current = initialViewport
  wheelOptionsRef.current = {
    mode: options.wheelMode ?? 'auto',
    zoomIntensity: options.wheelZoomIntensity,
  }

  const setViewport = useCallback((next: CanvasViewportUpdate) => {
    setViewportState((current) => normalizeViewport(typeof next === 'function' ? next(current) : next, limitsRef.current))
  }, [])

  const containerRef = useCallback((node: HTMLElement | null) => {
    containerElementRef.current = node
    setContainerElement(node)
  }, [])

  useEffect(() => {
    setViewportState((current) => normalizeViewport(current, limits))
  }, [limits])

  useEffect(() => {
    const node = containerElement
    if (!node) {
      setViewportSize(EMPTY_SIZE)
      return
    }

    const measure = () => {
      const rect = node.getBoundingClientRect()
      setViewportSize({
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
  }, [containerElement])

  const viewportStyle = useMemo<CSSProperties>(() => ({
    transform: viewportCssTransform(viewport, limits),
    transformOrigin: '0 0',
  }), [limits, viewport])

  const resetView = useCallback((nextViewport?: CanvasViewport) => {
    setViewport(resetViewport({ viewport: nextViewport ?? initialViewportRef.current, ...limitsRef.current }))
  }, [setViewport])

  const fitView = useCallback((bounds: CanvasBounds | null | undefined, fitOptions: FitViewOptions = {}) => {
    setViewport(fitViewport(bounds, viewportSizeRef.current, {
      ...limitsRef.current,
      fallbackViewport: initialViewportRef.current,
      ...fitOptions,
    }))
  }, [setViewport])

  const panBy = useCallback((delta: ViewportPoint) => {
    setViewport((current) => panViewport(current, delta, limitsRef.current))
  }, [setViewport])

  const zoomAt = useCallback((anchor: ViewportPoint, requestedZoom: number) => {
    setViewport((current) => zoomViewportAtPoint(current, anchor, requestedZoom, limitsRef.current))
  }, [setViewport])

  const clientToViewportPoint = useCallback((clientX: number, clientY: number) => {
    return viewportPointFromClient(clientX, clientY, containerElementRef.current?.getBoundingClientRect())
  }, [])

  const viewportPointToCanvasPointForView = useCallback((point: ViewportPoint) => {
    return viewportPointToCanvasPoint(point, viewportRef.current, limitsRef.current)
  }, [])

  const canvasPointToViewportPointForView = useCallback((point: CanvasPoint) => {
    return canvasPointToViewportPoint(point, viewportRef.current, limitsRef.current)
  }, [])

  const clientToCanvasPoint = useCallback((clientX: number, clientY: number) => {
    return viewportPointToCanvasPointForView(clientToViewportPoint(clientX, clientY))
  }, [clientToViewportPoint, viewportPointToCanvasPointForView])

  const handleWheel = useCallback((event: ReactWheelEvent<HTMLElement>) => {
    if (event.cancelable) event.preventDefault()
    const anchor = viewportPointFromClient(event.clientX, event.clientY, event.currentTarget.getBoundingClientRect())
    const wheelState = wheelViewportState(viewportRef.current, {
      deltaX: event.deltaX,
      deltaY: event.deltaY,
      deltaMode: event.deltaMode,
      ctrlKey: event.ctrlKey,
      metaKey: event.metaKey,
      shiftKey: event.shiftKey,
    }, anchor, {
      ...limitsRef.current,
      mode: wheelOptionsRef.current.mode,
      viewportSize: viewportSizeRef.current,
      zoomIntensity: wheelOptionsRef.current.zoomIntensity,
    })
    setViewport(wheelState.nextViewport)
    return wheelState
  }, [setViewport])

  const beginPan = useCallback((event: ReactPointerEvent<HTMLElement>) => {
    event.currentTarget.setPointerCapture?.(event.pointerId)
    const nextPanState: CanvasPanState = {
      type: 'pan',
      pointerId: event.pointerId,
      startClient: { x: event.clientX, y: event.clientY },
      lastClient: { x: event.clientX, y: event.clientY },
      delta: { x: 0, y: 0 },
      startViewport: viewportRef.current,
    }
    panStateRef.current = nextPanState
    setPanState(nextPanState)
    return nextPanState
  }, [])

  const updatePan = useCallback((event: ReactPointerEvent<HTMLElement>) => {
    const currentPanState = panStateRef.current
    if (!currentPanState || currentPanState.pointerId !== event.pointerId) return null

    const delta = {
      x: event.clientX - currentPanState.startClient.x,
      y: event.clientY - currentPanState.startClient.y,
    }
    const nextViewport = panViewport(currentPanState.startViewport, delta, limitsRef.current)
    const nextPanState: CanvasPanState = {
      ...currentPanState,
      lastClient: { x: event.clientX, y: event.clientY },
      delta,
    }

    panStateRef.current = nextPanState
    setPanState(nextPanState)
    setViewport(nextViewport)
    return nextViewport
  }, [setViewport])

  const endPan = useCallback((event: ReactPointerEvent<HTMLElement>) => {
    const currentPanState = panStateRef.current
    if (!currentPanState || currentPanState.pointerId !== event.pointerId) return null
    try {
      event.currentTarget.releasePointerCapture?.(event.pointerId)
    } catch {
      // Pointer capture may already be released by the browser.
    }
    panStateRef.current = null
    setPanState(null)
    return currentPanState
  }, [])

  const cancelPan = useCallback(() => {
    const currentPanState = panStateRef.current
    panStateRef.current = null
    setPanState(null)
    return currentPanState
  }, [])

  return {
    viewport,
    viewportRef,
    viewportSize,
    panState,
    containerRef,
    containerElementRef,
    viewportStyle,
    setViewport,
    resetView,
    fitView,
    panBy,
    zoomAt,
    clientToViewportPoint,
    clientToCanvasPoint,
    viewportPointToCanvasPoint: viewportPointToCanvasPointForView,
    canvasPointToViewportPoint: canvasPointToViewportPointForView,
    handleWheel,
    beginPan,
    updatePan,
    endPan,
    cancelPan,
  }
}
