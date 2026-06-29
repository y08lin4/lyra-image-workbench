export type CanvasPoint = {
  x: number
  y: number
}

export type ViewportPoint = CanvasPoint

export type ViewportSize = {
  width: number
  height: number
}

export type CanvasBounds = CanvasPoint & ViewportSize

export type CanvasViewport = {
  x: number
  y: number
  zoom: number
}

export type CanvasViewportLimits = {
  minZoom: number
  maxZoom: number
}

export type FitViewOptions = Partial<CanvasViewportLimits> & {
  padding?: number
  fallbackViewport?: CanvasViewport
}

export type ResetViewOptions = Partial<CanvasViewportLimits> & {
  viewport?: CanvasViewport
}

export type CanvasWheelMode = 'pan' | 'zoom'

export type CanvasWheelInput = {
  deltaX: number
  deltaY: number
  deltaMode?: number
  ctrlKey?: boolean
  metaKey?: boolean
  shiftKey?: boolean
}

export type CanvasWheelViewportOptions = Partial<CanvasViewportLimits> & {
  mode?: CanvasWheelMode | 'auto'
  viewportSize?: ViewportSize
  zoomIntensity?: number
}

export type CanvasWheelState = {
  mode: CanvasWheelMode
  anchor: ViewportPoint
  delta: ViewportPoint
  previousViewport: CanvasViewport
  nextViewport: CanvasViewport
}

export type CanvasPanState = {
  type: 'pan'
  pointerId: number
  startClient: ViewportPoint
  lastClient: ViewportPoint
  delta: ViewportPoint
  startViewport: CanvasViewport
}

export type ViewportRect = {
  left: number
  top: number
  width: number
  height: number
}

export const DEFAULT_CANVAS_VIEWPORT: CanvasViewport = { x: 0, y: 0, zoom: 1 }
export const DEFAULT_CANVAS_VIEWPORT_LIMITS: CanvasViewportLimits = { minZoom: 0.1, maxZoom: 4 }

const WHEEL_LINE_HEIGHT = 16
const DEFAULT_WHEEL_PAGE_SIZE = 800
const DEFAULT_ZOOM_INTENSITY = 0.002

export function resetView(options: ResetViewOptions = {}): CanvasViewport {
  return normalizeViewport(options.viewport ?? DEFAULT_CANVAS_VIEWPORT, options)
}

export function fitView(bounds: CanvasBounds | null | undefined, viewportSize: ViewportSize, options: FitViewOptions = {}): CanvasViewport {
  const fallback = normalizeViewport(options.fallbackViewport ?? DEFAULT_CANVAS_VIEWPORT, options)
  if (!bounds || !hasPositiveSize(bounds) || !hasPositiveSize(viewportSize)) return fallback

  const padding = Math.max(0, finiteOr(options.padding, 0))
  const availableWidth = Math.max(1, viewportSize.width - padding * 2)
  const availableHeight = Math.max(1, viewportSize.height - padding * 2)
  const zoom = clampZoom(Math.min(availableWidth / bounds.width, availableHeight / bounds.height), options)
  const boundsCenter = {
    x: bounds.x + bounds.width / 2,
    y: bounds.y + bounds.height / 2,
  }

  return normalizeViewport({
    x: viewportSize.width / 2 - boundsCenter.x * zoom,
    y: viewportSize.height / 2 - boundsCenter.y * zoom,
    zoom,
  }, options)
}

export function viewportPointFromClient(clientX: number, clientY: number, rect: Pick<ViewportRect, 'left' | 'top'> | null | undefined): ViewportPoint {
  return {
    x: clientX - (rect?.left ?? 0),
    y: clientY - (rect?.top ?? 0),
  }
}

export function viewportPointToCanvasPoint(point: ViewportPoint, viewport: CanvasViewport, limits: Partial<CanvasViewportLimits> = {}): CanvasPoint {
  const normalized = normalizeViewport(viewport, limits)
  return {
    x: roundViewportMetric((point.x - normalized.x) / normalized.zoom),
    y: roundViewportMetric((point.y - normalized.y) / normalized.zoom),
  }
}

export function canvasPointToViewportPoint(point: CanvasPoint, viewport: CanvasViewport, limits: Partial<CanvasViewportLimits> = {}): ViewportPoint {
  const normalized = normalizeViewport(viewport, limits)
  return {
    x: roundViewportMetric(point.x * normalized.zoom + normalized.x),
    y: roundViewportMetric(point.y * normalized.zoom + normalized.y),
  }
}

export function viewportDeltaToCanvasDelta(delta: ViewportPoint, viewport: CanvasViewport, limits: Partial<CanvasViewportLimits> = {}): CanvasPoint {
  const normalized = normalizeViewport(viewport, limits)
  return {
    x: roundViewportMetric(delta.x / normalized.zoom),
    y: roundViewportMetric(delta.y / normalized.zoom),
  }
}

export function panViewport(viewport: CanvasViewport, delta: ViewportPoint, limits: Partial<CanvasViewportLimits> = {}): CanvasViewport {
  const normalized = normalizeViewport(viewport, limits)
  return normalizeViewport({
    x: normalized.x + delta.x,
    y: normalized.y + delta.y,
    zoom: normalized.zoom,
  }, limits)
}

export function zoomViewportAtPoint(
  viewport: CanvasViewport,
  anchor: ViewportPoint,
  requestedZoom: number,
  limits: Partial<CanvasViewportLimits> = {},
): CanvasViewport {
  const normalized = normalizeViewport(viewport, limits)
  const zoom = clampZoom(requestedZoom, limits)
  const canvasAnchor = viewportPointToCanvasPoint(anchor, normalized, limits)
  return normalizeViewport({
    x: anchor.x - canvasAnchor.x * zoom,
    y: anchor.y - canvasAnchor.y * zoom,
    zoom,
  }, limits)
}

export function wheelZoomFactor(deltaY: number, zoomIntensity = DEFAULT_ZOOM_INTENSITY): number {
  return Math.exp(-deltaY * Math.max(0, finiteOr(zoomIntensity, DEFAULT_ZOOM_INTENSITY)))
}

export function wheelViewportState(
  viewport: CanvasViewport,
  input: CanvasWheelInput,
  anchor: ViewportPoint,
  options: CanvasWheelViewportOptions = {},
): CanvasWheelState {
  const previousViewport = normalizeViewport(viewport, options)
  const delta = normalizeWheelDelta(input, options.viewportSize)
  const mode = resolveWheelMode(input, options.mode)
  const nextViewport = mode === 'zoom'
    ? zoomViewportAtPoint(previousViewport, anchor, previousViewport.zoom * wheelZoomFactor(delta.y, options.zoomIntensity), options)
    : panViewport(previousViewport, wheelPanDelta(delta, input), options)

  return {
    mode,
    anchor,
    delta,
    previousViewport,
    nextViewport,
  }
}

export function normalizeViewport(viewport: CanvasViewport, limits: Partial<CanvasViewportLimits> = {}): CanvasViewport {
  return {
    x: roundViewportMetric(finiteOr(viewport.x, DEFAULT_CANVAS_VIEWPORT.x)),
    y: roundViewportMetric(finiteOr(viewport.y, DEFAULT_CANVAS_VIEWPORT.y)),
    zoom: clampZoom(viewport.zoom, limits),
  }
}

export function clampZoom(value: number, limits: Partial<CanvasViewportLimits> = {}): number {
  const minZoom = Math.max(0.001, finiteOr(limits.minZoom, DEFAULT_CANVAS_VIEWPORT_LIMITS.minZoom))
  const maxZoom = Math.max(minZoom, finiteOr(limits.maxZoom, DEFAULT_CANVAS_VIEWPORT_LIMITS.maxZoom))
  return roundViewportMetric(clamp(finiteOr(value, DEFAULT_CANVAS_VIEWPORT.zoom), minZoom, maxZoom))
}

export function viewportCssTransform(viewport: CanvasViewport, limits: Partial<CanvasViewportLimits> = {}): string {
  const normalized = normalizeViewport(viewport, limits)
  return `translate3d(${normalized.x}px, ${normalized.y}px, 0) scale(${normalized.zoom})`
}

export function roundViewportMetric(value: number): number {
  return Math.round(value * 1000) / 1000
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

function normalizeWheelDelta(input: CanvasWheelInput, viewportSize: ViewportSize | undefined): ViewportPoint {
  return {
    x: wheelDeltaToPixels(input.deltaX, input.deltaMode, viewportSize?.width),
    y: wheelDeltaToPixels(input.deltaY, input.deltaMode, viewportSize?.height),
  }
}

function wheelDeltaToPixels(delta: number, deltaMode: number | undefined, pageSize: number | undefined) {
  if (deltaMode === 1) return delta * WHEEL_LINE_HEIGHT
  if (deltaMode === 2) return delta * (pageSize || DEFAULT_WHEEL_PAGE_SIZE)
  return delta
}

function resolveWheelMode(input: CanvasWheelInput, mode: CanvasWheelViewportOptions['mode']): CanvasWheelMode {
  if (mode === 'pan' || mode === 'zoom') return mode
  return input.ctrlKey || input.metaKey ? 'zoom' : 'pan'
}

function wheelPanDelta(delta: ViewportPoint, input: CanvasWheelInput): ViewportPoint {
  if (input.shiftKey && Math.abs(delta.x) < 1) {
    return { x: -delta.y, y: 0 }
  }
  return {
    x: -delta.x,
    y: -delta.y,
  }
}

function hasPositiveSize(size: ViewportSize): boolean {
  return Number.isFinite(size.width) && Number.isFinite(size.height) && size.width > 0 && size.height > 0
}

function finiteOr(value: number | undefined, fallback: number): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}
