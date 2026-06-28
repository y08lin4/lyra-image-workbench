import { useEffect, useMemo, useRef, useState } from 'react'
import type {
  ClipboardEvent as ReactClipboardEvent,
  DragEvent as ReactDragEvent,
  KeyboardEvent as ReactKeyboardEvent,
  MouseEvent as ReactMouseEvent,
  PointerEvent as ReactPointerEvent,
  WheelEvent as ReactWheelEvent,
} from 'react'
import type { CreateTaskRequest, Mode, ModelProvider, ReferenceUpload, Task } from '../types'
import { formatError } from '../api/client'
import { getReferenceUploadBlob } from '../api/uploads'
import { buildCanvasPromptDraft, generateCanvasPromptFromCanvas, optimizeCanvasTextPrompt } from './creativeCanvas/promptOptimization'
import type { CanvasConnection, CanvasContextMenu, CanvasHistoryImage, CanvasInteraction, CanvasItem, ReferenceRole } from './creativeCanvas/types'
import { REFERENCE_ROLES } from './creativeCanvas/types'
import {
  canvasTextPointFromClient,
  createCanvasTextItem,
  isCanvasImageItem,
  isCanvasTextItem,
  updateCanvasTextItemText,
} from './creativeCanvas/textItems'
import {
  buildReferencePromptLine,
  clampNumber,
  createCanvasItemFromHistory,
  createCanvasItemFromUpload,
  imageFilesFromClipboard,
  imageSrcForCanvasItem,
  isImageFile,
  referenceIndexForItem,
  roleMeta,
  safeParseDragData,
  unique,
} from './creativeCanvas/data'
import {
  autoItemPosition,
  canvasItemContentStyle,
  canvasItemStyle,
  canvasControlStyle,
  canvasPointFromClient,
  dropPointFromEvent,
  itemCenter,
  nearestConnectableItem,
  normalizeRotation,
  pointerAngleForItem,
  scaleCanvasItemByWheel,
  spreadPoint,
  updateCanvasItemsForInteraction,
} from './creativeCanvas/geometry'
import { aspectRatioValue, canvasImageSizeFromSrc, extensionLabel, modeLabel } from './creativeCanvas/imageSizing'
import {
  BANANA_MODEL_OPTIONS,
  BANANA_PROVIDER,
  DEFAULT_BANANA_MODEL,
  DEFAULT_IMAGE2_MODEL,
  IMAGE2_PROVIDER,
  getBananaModelOption,
  providerLabel,
} from '../lib/models'
import {
  OUTPUT_FORMATS,
  QUALITY_LEVELS,
  RATIOS,
  RESOLUTION_TIERS,
  getImageSize,
  getOutputFormatLabel,
  getQualityLabel,
  getResolutionLabel,
} from '../lib/ratios'
import {
  DEFAULT_CONNECTION_LABEL,
  appendCanvasContextPrompt,
  appendPromptLine,
  connectionLabel,
  normalizeConnectionLabel,
} from './creativeCanvas/connectionPrompt'
import { loadCreativeCanvasDraft, saveCreativeCanvasDraft } from './creativeCanvas/persistence'
import { canvasConnectionClassName, canvasConnectionLabelClassName, isEditableTarget } from './creativeCanvas/interaction'
import './NodeWorkflowPage.css'

export type NodeWorkflowUsePromptOptions = {
  provider: ModelProvider
  model: string
  ratio?: string
}

export type { CanvasHistoryImage } from './creativeCanvas/types'

export type NodeWorkflowPageProps = {
  provider: ModelProvider
  bananaModel: string
  prompt?: string
  injectedPrompt?: string
  injectedPromptRevision?: number
  initialPrompt?: string
  onUsePrompt: (prompt: string, options: NodeWorkflowUsePromptOptions) => void
  onCreateTask?: (payload: CreateTaskRequest) => void | Promise<void>
  onUploadReferences?: (files: File[]) => Promise<ReferenceUpload[]>
  onDeleteReferenceUpload?: (id: string) => void | Promise<void>
  onUseHistoryImageAsReference?: (src: string, index: number) => Promise<ReferenceUpload | undefined>
  referenceUploads?: ReferenceUpload[]
  recentResults?: CanvasHistoryImage[]
  latestTask?: Task
}

const HISTORY_DRAG_TYPE = 'application/x-lyra-history-result'
const UPLOAD_DRAG_TYPE = 'application/x-lyra-reference-upload'

export function NodeWorkflowPage({
  provider,
  bananaModel,
  prompt,
  injectedPrompt,
  injectedPromptRevision = 0,
  initialPrompt,
  onUsePrompt,
  onCreateTask,
  onUploadReferences,
  onDeleteReferenceUpload,
  onUseHistoryImageAsReference,
  referenceUploads = [],
  recentResults = [],
  latestTask,
}: NodeWorkflowPageProps) {
  const [initialCanvasDraft] = useState(loadCreativeCanvasDraft)
  const initialDraftPrompt = injectedPromptRevision > 0 && injectedPrompt ? injectedPrompt : initialCanvasDraft.hasStoredDraft ? initialCanvasDraft.prompt : prompt || initialPrompt || ''
  const [mode, setMode] = useState<Mode>(initialCanvasDraft.mode)
  const [draftPrompt, setDraftPrompt] = useState(initialDraftPrompt)
  const [localProvider, setLocalProvider] = useState<ModelProvider>(provider || IMAGE2_PROVIDER)
  const [localBananaModel, setLocalBananaModel] = useState(bananaModel || DEFAULT_BANANA_MODEL)
  const [ratio, setRatio] = useState('1:1')
  const [resolution, setResolution] = useState('standard')
  const [quality, setQuality] = useState('high')
  const [outputFormat, setOutputFormat] = useState('png')
  const [count, setCount] = useState(1)
  const [concurrency, setConcurrency] = useState(1)
  const [message, setMessage] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isGeneratingCanvasPrompt, setIsGeneratingCanvasPrompt] = useState(false)
  const [previewUrls, setPreviewUrls] = useState<Record<string, string>>({})
  const [canvasItems, setCanvasItems] = useState<CanvasItem[]>(initialCanvasDraft.items)
  const [connections, setConnections] = useState<CanvasConnection[]>(initialCanvasDraft.connections)
  const [selectedItemId, setSelectedItemId] = useState<string | null>(null)
  const [selectedConnectionId, setSelectedConnectionId] = useState<string | null>(null)
  const [generatedPromptOverride, setGeneratedPromptOverride] = useState('')
  const [connectionDraftFrom, setConnectionDraftFrom] = useState<string | null>(null)
  const [editingConnectionId, setEditingConnectionId] = useState<string | null>(null)
  const [connectionLabelDraft, setConnectionLabelDraft] = useState('')
  const [contextMenu, setContextMenu] = useState<CanvasContextMenu | null>(null)
  const [isDropActive, setIsDropActive] = useState(false)
  const [interaction, setInteraction] = useState<CanvasInteraction | null>(null)
  const [deletingUploadId, setDeletingUploadId] = useState<string | null>(null)
  const [editingTextItemId, setEditingTextItemId] = useState<string | null>(null)
  const [optimizingTextItemId, setOptimizingTextItemId] = useState<string | null>(null)

  const stageRef = useRef<HTMLElement | null>(null)
  const promptTextareaRef = useRef<HTMLTextAreaElement | null>(null)
  const textEditorRefs = useRef<Record<string, HTMLTextAreaElement | null>>({})
  const canvasItemsRef = useRef<CanvasItem[]>(initialCanvasDraft.items)
  const connectionsRef = useRef<CanvasConnection[]>(initialCanvasDraft.connections)
  const draftPromptRef = useRef(draftPrompt)
  const modeRef = useRef<Mode>(mode)
  const pastePointRef = useRef<{ x: number; y: number }>(autoItemPosition(0))
  const localPreviewUrlsRef = useRef<string[]>([])
  const autofocusedTextItemIdRef = useRef<string | null>(null)
  const canvasDropDepthRef = useRef(0)
  const hadStoredCanvasDraftRef = useRef(initialCanvasDraft.hasStoredDraft)
  const appliedPromptInjectionRevisionRef = useRef(0)

  useEffect(() => {
    setLocalProvider(provider || IMAGE2_PROVIDER)
  }, [provider])

  useEffect(() => {
    setLocalBananaModel(bananaModel || DEFAULT_BANANA_MODEL)
  }, [bananaModel])

  useEffect(() => {
    if (!injectedPrompt || injectedPromptRevision <= 0 || appliedPromptInjectionRevisionRef.current === injectedPromptRevision) return
    appliedPromptInjectionRevisionRef.current = injectedPromptRevision
    setDraftPrompt(injectedPrompt)
    setGeneratedPromptOverride('')
    window.setTimeout(() => {
      promptTextareaRef.current?.focus()
      promptTextareaRef.current?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }, 0)
  }, [injectedPrompt, injectedPromptRevision])

  useEffect(() => {
    if (!prompt || hadStoredCanvasDraftRef.current) return
    setDraftPrompt(prompt)
    setGeneratedPromptOverride('')
  }, [prompt])

  useEffect(() => {
    const handle = window.setTimeout(() => {
      saveCreativeCanvasDraft({ items: canvasItems, connections, prompt: draftPrompt, mode, hasStoredDraft: true })
    }, 300)
    return () => window.clearTimeout(handle)
  }, [canvasItems, connections, draftPrompt, mode])

  useEffect(() => () => {
    saveCreativeCanvasDraft({
      items: canvasItemsRef.current,
      connections: connectionsRef.current,
      prompt: draftPromptRef.current,
      mode: modeRef.current,
      hasStoredDraft: true,
    })
  }, [])

  useEffect(() => {
    let disposed = false
    const created: string[] = []

    async function loadPreviews() {
      const entries = await Promise.all(referenceUploads.map(async (item) => {
        try {
          const blob = await getReferenceUploadBlob(item.id)
          const url = URL.createObjectURL(blob)
          created.push(url)
          return [item.id, url] as const
        } catch {
          return [item.id, ''] as const
        }
      }))
      if (disposed) {
        created.forEach((url) => URL.revokeObjectURL(url))
        return
      }
      setPreviewUrls(Object.fromEntries(entries))
    }

    void loadPreviews()
    return () => {
      disposed = true
      created.forEach((url) => URL.revokeObjectURL(url))
    }
  }, [referenceUploads])

  useEffect(() => {
    if (!referenceUploads.length) return
    const firstUpload = referenceUploads[0]
    if (!(firstUpload.id in previewUrls)) return
    let disposed = false

    async function addInitialReference() {
      const size = await canvasImageSizeFromSrc(previewUrls[firstUpload.id])
      if (disposed) return
      setCanvasItems((items) => {
        if (items.length || hadStoredCanvasDraftRef.current) return items
        return [createCanvasItemFromUpload(firstUpload, { x: 88, y: 82 }, 0, undefined, size)]
      })
    }

    void addInitialReference()
    return () => {
      disposed = true
    }
  }, [previewUrls, referenceUploads])

  useEffect(() => () => {
    localPreviewUrlsRef.current.forEach((url) => URL.revokeObjectURL(url))
  }, [])

  useEffect(() => {
    canvasItemsRef.current = canvasItems
    connectionsRef.current = connections
    draftPromptRef.current = draftPrompt
    modeRef.current = mode
  }, [canvasItems, connections, draftPrompt, mode])

  useEffect(() => {
    const itemIds = new Set(canvasItems.map((item) => item.id))
    setSelectedConnectionId((current) => {
      if (!current) return current
      const exists = connections.some((item) => item.id === current && itemIds.has(item.fromId) && itemIds.has(item.toId))
      return exists ? current : null
    })
  }, [canvasItems, connections])

  useEffect(() => {
    setGeneratedPromptOverride('')
  }, [canvasItems, connections, mode, ratio, localBananaModel])

  useEffect(() => {
    if (!interaction) return
    const handlePointerMove = (event: PointerEvent) => {
      const nextItems = updateCanvasItemsForInteraction(canvasItemsRef.current, interaction, event, stageRef.current)
      canvasItemsRef.current = nextItems
      setCanvasItems(nextItems)
    }
    const handlePointerUp = (event: PointerEvent) => {
      const nextItems = updateCanvasItemsForInteraction(canvasItemsRef.current, interaction, event, stageRef.current)
      const wasDragged = Math.hypot(event.clientX - interaction.startClientX, event.clientY - interaction.startClientY) > 2
      canvasItemsRef.current = nextItems
      setCanvasItems(nextItems)
      if (interaction.type === 'move' && wasDragged) {
        const nearest = nearestConnectableItem(nextItems, interaction.itemId)
        if (nearest) addConnection(interaction.itemId, nearest.id)
      }
      setInteraction(null)
    }
    const handlePointerCancel = () => setInteraction(null)
    window.addEventListener('pointermove', handlePointerMove)
    window.addEventListener('pointerup', handlePointerUp)
    window.addEventListener('pointercancel', handlePointerCancel)
    return () => {
      window.removeEventListener('pointermove', handlePointerMove)
      window.removeEventListener('pointerup', handlePointerUp)
      window.removeEventListener('pointercancel', handlePointerCancel)
    }
  }, [interaction])

  useEffect(() => {
    if (!contextMenu) return
    const close = () => setContextMenu(null)
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') close()
    }
    window.addEventListener('click', close)
    window.addEventListener('keydown', closeOnEscape)
    return () => {
      window.removeEventListener('click', close)
      window.removeEventListener('keydown', closeOnEscape)
    }
  }, [contextMenu])

  useEffect(() => {
    if (!editingTextItemId || typeof window === 'undefined') {
      autofocusedTextItemIdRef.current = null
      return
    }
    if (autofocusedTextItemIdRef.current === editingTextItemId) return
    autofocusedTextItemIdRef.current = editingTextItemId

    const frame = window.requestAnimationFrame(() => {
      const editor = textEditorRefs.current[editingTextItemId]
      if (!editor || document.activeElement === editor) return
      editor.focus({ preventScroll: true })
      const end = editor.value.length
      editor.setSelectionRange(end, end)
    })
    return () => window.cancelAnimationFrame(frame)
  }, [editingTextItemId])

  const bananaOption = useMemo(() => getBananaModelOption(localBananaModel), [localBananaModel])
  const selectedModel = localProvider === BANANA_PROVIDER ? bananaOption.id : DEFAULT_IMAGE2_MODEL
  const effectiveRatio = localProvider === BANANA_PROVIDER ? bananaOption.ratio : ratio
  const effectiveResolution = localProvider === BANANA_PROVIDER ? bananaOption.resolution : resolution
  const effectiveQuality = localProvider === BANANA_PROVIDER ? 'auto' : quality
  const effectiveOutputFormat = localProvider === BANANA_PROVIDER ? 'auto' : outputFormat
  const imageSize = localProvider === BANANA_PROVIDER ? bananaOption.size : getImageSize(ratio, resolution)
  const trimmedPrompt = draftPrompt.trim()
  const selectedItem = useMemo(() => canvasItems.find((item) => item.id === selectedItemId), [canvasItems, selectedItemId])
  const selectedImageItem = useMemo(() => (isCanvasImageItem(selectedItem) ? selectedItem : null), [selectedItem])
  const selectedTextItem = useMemo(() => (isCanvasTextItem(selectedItem) ? selectedItem : null), [selectedItem])
  const referenceCanvasItems = useMemo(() => canvasItems.filter(isCanvasImageItem).filter((item) => item.isReference), [canvasItems])
  const canvasTextPromptItems = useMemo(() => canvasItems.filter(isCanvasTextItem).filter((item) => item.text.trim().length > 0), [canvasItems])
  const availableUploadIds = useMemo(() => new Set(referenceUploads.map((item) => item.id)), [referenceUploads])
  const markedUploadIds = useMemo(() => unique(referenceCanvasItems.map((item) => item.uploadId).filter((id): id is string => typeof id === 'string' && availableUploadIds.has(id))), [availableUploadIds, referenceCanvasItems])
  const taskUploadIds = mode === 'image-to-image' ? markedUploadIds : []
  const generatedPromptTrimmed = generatedPromptOverride.trim()
  const hasCanvasPromptContent = generatedPromptTrimmed.length > 0 || trimmedPrompt.length > 0 || referenceCanvasItems.length > 0 || canvasTextPromptItems.length > 0 || connections.length > 0
  const submissionPrompt = useMemo(() => generatedPromptTrimmed || appendCanvasContextPrompt(trimmedPrompt, canvasItems, connections), [canvasItems, connections, generatedPromptTrimmed, trimmedPrompt])
  const trimmedSubmissionPrompt = submissionPrompt.trim()

  const canCreateTask = hasCanvasPromptContent && (mode === 'text-to-image' || taskUploadIds.length > 0)
  const latestOkResult = useMemo(() => {
    const okResults = latestTask?.results.filter((item) => item.ok && item.imageUrl) || []
    return okResults[okResults.length - 1]
  }, [latestTask])
  const selectedItemPreview = selectedImageItem ? imageSrcForCanvasItem(selectedImageItem, previewUrls) : ''
  const renderPreviewSrc = selectedItemPreview || latestOkResult?.imageUrl || ''
  const renderPreviewAlt = selectedImageItem ? selectedImageItem.name : latestOkResult ? '最新生成结果' : selectedItem?.name || '画布预览'
  const visibleReferences = referenceUploads.slice(0, 10)
  const visibleHistory = recentResults.slice(0, 12)
  const contextMenuItem = contextMenu?.target === 'item' ? canvasItems.find((item) => item.id === contextMenu.itemId) : null

  const taskPayload = useMemo<CreateTaskRequest>(() => ({
    provider: localProvider,
    model: selectedModel,
    mode,
    prompt: trimmedSubmissionPrompt,
    ratio: effectiveRatio,
    resolution: effectiveResolution,
    quality: effectiveQuality,
    outputFormat: effectiveOutputFormat,
    count,
    concurrency,
    uploadIds: taskUploadIds,
  }), [
    concurrency,
    count,
    effectiveOutputFormat,
    effectiveQuality,
    effectiveRatio,
    effectiveResolution,
    localProvider,
    mode,
    selectedModel,
    trimmedSubmissionPrompt,
    taskUploadIds,
  ])

  function useCanvasPrompt() {
    if (!hasCanvasPromptContent || !trimmedSubmissionPrompt) {
      setMessage('先写一句提示词，或在画布里添加文字块/连线。')
      return
    }
    onUsePrompt(trimmedSubmissionPrompt, { provider: localProvider, model: selectedModel, ratio: effectiveRatio || undefined })
    setMessage('已同步到快捷生成页。')
  }

  async function createCanvasTask() {
    if (!hasCanvasPromptContent || !trimmedSubmissionPrompt) {
      setMessage('先写一句提示词，或在画布里添加文字块/连线。')
      return
    }
    if (mode === 'image-to-image' && taskUploadIds.length === 0) {
      setMessage('参考图生成需要先上传图片，或从历史结果选择至少一张参考图。')
      return
    }
    if (!onCreateTask) {
      setMessage('任务创建暂不可用，请稍后再试。')
      return
    }

    setIsSubmitting(true)
    setMessage('')
    try {
      await onCreateTask(taskPayload)
      setMessage('任务已创建，右侧预览会跟随当前结果更新。')
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '创建任务失败。')
    } finally {
      setIsSubmitting(false)
    }
  }

  async function resolveUploadPreviewUrl(uploadId: string) {
    const current = previewUrls[uploadId]
    if (current) return { src: current, localPreviewUrl: undefined }
    try {
      const blob = await getReferenceUploadBlob(uploadId)
      const url = URL.createObjectURL(blob)
      localPreviewUrlsRef.current.push(url)
      setPreviewUrls((items) => ({ ...items, [uploadId]: url }))
      return { src: url, localPreviewUrl: url }
    } catch {
      return { src: '', localPreviewUrl: undefined }
    }
  }

  async function addUploadToCanvas(upload: ReferenceUpload, point?: { x: number; y: number }) {
    if (!point) {
      const existing = canvasItemsRef.current.find((item) => isCanvasImageItem(item) && item.uploadId === upload.id)
      if (existing) {
        setCanvasItems((items) => items.map((item) => (isCanvasImageItem(item) && item.id === existing.id ? { ...item, isReference: true } : item)))
        setSelectedItemId(existing.id)
        setSelectedConnectionId(null)
        setMode('image-to-image')
        setMessage('已选中画布中的参考图。')
        return
      }
    }
    const index = canvasItems.length
    const preview = await resolveUploadPreviewUrl(upload.id)
    const size = await canvasImageSizeFromSrc(preview.src)
    const item = createCanvasItemFromUpload(upload, point || autoItemPosition(index), index, preview.localPreviewUrl, size)
    setCanvasItems((items) => [...items, item])
    setSelectedItemId(item.id)
    setSelectedConnectionId(null)
    setMode('image-to-image')
    setMessage('已加入画布参考。')
  }

  async function addFilesToCanvas(files: File[], point: { x: number; y: number }) {
    const imageFiles = files.filter(isImageFile)
    if (!imageFiles.length) {
      setMessage('只能拖入 PNG、JPG 或 WEBP 图片。')
      return
    }
    if (!onUploadReferences) {
      setMessage('参考图上传暂不可用，请稍后再试。')
      return
    }
    try {
      const localUrls = imageFiles.map((file) => {
        const url = URL.createObjectURL(file)
        localPreviewUrlsRef.current.push(url)
        return url
      })
      const sizePromises = localUrls.map((url) => canvasImageSizeFromSrc(url))
      const [created, sizes] = await Promise.all([onUploadReferences(imageFiles), Promise.all(sizePromises)])
      if (!created.length) return
      const nextItems = created.map((upload, index) => createCanvasItemFromUpload(upload, spreadPoint(point, index), canvasItems.length + index, localUrls[index], sizes[index]))
      setCanvasItems((items) => [...items, ...nextItems])
      setSelectedItemId(nextItems[0].id)
      setSelectedConnectionId(null)
      setMode('image-to-image')
      setMessage(`已加入 ${created.length} 张参考图。`)
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '参考图上传失败。')
    }
  }

  async function addHistoryImageToCanvas(image: CanvasHistoryImage, point: { x: number; y: number }) {
    const existing = canvasItemsRef.current.find((item) => isCanvasImageItem(item) && item.resultSrc === image.src)
    if (existing) {
      setCanvasItems((items) => items.map((item) => (isCanvasImageItem(item) && item.id === existing.id ? { ...item, isReference: true } : item)))
      setSelectedItemId(existing.id)
      setSelectedConnectionId(null)
      setMode('image-to-image')
      setMessage('已选中画布中的历史参考图。')
      return
    }
    try {
      const uploadPromise = onUseHistoryImageAsReference
        ? onUseHistoryImageAsReference(image.src, image.index)
        : Promise.resolve(undefined)
      const [upload, size] = await Promise.all([uploadPromise, canvasImageSizeFromSrc(image.src)])
      if (upload?.id) {
        const existingUpload = canvasItemsRef.current.find((item) => isCanvasImageItem(item) && item.uploadId === upload.id)
        if (existingUpload) {
          setCanvasItems((items) => items.map((item) => (isCanvasImageItem(item) && item.id === existingUpload.id ? { ...item, isReference: true, resultSrc: item.resultSrc || image.src } : item)))
          setSelectedItemId(existingUpload.id)
          setSelectedConnectionId(null)
          setMode('image-to-image')
          setMessage('已选中画布中的历史参考图。')
          return
        }
      }
      const item = createCanvasItemFromHistory(image, point, canvasItems.length, upload, size)
      setCanvasItems((items) => [...items, item])
      setSelectedItemId(item.id)
      setSelectedConnectionId(null)
      setMode('image-to-image')
      setMessage('历史结果已加入画布参考。')
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '加入历史结果失败。')
    }
  }
  async function deleteReferenceUploadFromStrip(event: ReactMouseEvent<HTMLButtonElement>, upload: ReferenceUpload) {
    event.preventDefault()
    event.stopPropagation()
    if (!onDeleteReferenceUpload || deletingUploadId) return
    setDeletingUploadId(upload.id)
    try {
      await onDeleteReferenceUpload(upload.id)
      const removedIds = new Set(canvasItems.filter(isCanvasImageItem).filter((item) => item.uploadId === upload.id).map((item) => item.id))
      setCanvasItems((items) => items.filter((item) => !isCanvasImageItem(item) || item.uploadId !== upload.id))
      setConnections((items) => items.filter((item) => !removedIds.has(item.fromId) && !removedIds.has(item.toId)))
      setSelectedItemId((current) => (current && removedIds.has(current) ? null : current))
      setConnectionDraftFrom((current) => (current && removedIds.has(current) ? null : current))
      setMessage('参考图已删除。')
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '删除参考图失败。')
    } finally {
      setDeletingUploadId(null)
    }
  }

  function shouldHandleCanvasDrop(event: ReactDragEvent<HTMLElement>) {
    const types = Array.from(event.dataTransfer.types || [])
    if (types.includes('Files') || types.includes(UPLOAD_DRAG_TYPE) || types.includes(HISTORY_DRAG_TYPE) || types.includes('text/uri-list')) return true
    if (!types.includes('text/plain')) return false
    const target = event.target as HTMLElement | null
    return !target?.closest('textarea,input,[contenteditable="true"]')
  }

  function handleCanvasDragEnter(event: ReactDragEvent<HTMLElement>) {
    if (!shouldHandleCanvasDrop(event)) return
    event.preventDefault()
    canvasDropDepthRef.current += 1
    setIsDropActive(true)
  }

  function handleCanvasDragOver(event: ReactDragEvent<HTMLElement>) {
    if (!shouldHandleCanvasDrop(event)) return
    event.preventDefault()
    event.dataTransfer.dropEffect = 'copy'
    setIsDropActive(true)
  }

  function handleCanvasDragLeave(event: ReactDragEvent<HTMLElement>) {
    if (!shouldHandleCanvasDrop(event)) return
    event.preventDefault()
    canvasDropDepthRef.current = Math.max(0, canvasDropDepthRef.current - 1)
    if (canvasDropDepthRef.current === 0) setIsDropActive(false)
  }

  function handleCanvasDrop(event: ReactDragEvent<HTMLElement>) {
    if (!shouldHandleCanvasDrop(event)) return
    event.preventDefault()
    event.stopPropagation()
    canvasDropDepthRef.current = 0
    setIsDropActive(false)
    const point = dropPointFromEvent(event, stageRef.current)
    const files = Array.from(event.dataTransfer.files || [])
    if (files.length) {
      void addFilesToCanvas(files, point)
      return
    }

    const uploadPayload = event.dataTransfer.getData(UPLOAD_DRAG_TYPE)
    if (uploadPayload) {
      const uploadId = safeParseDragData<{ uploadId: string }>(uploadPayload)?.uploadId
      const upload = referenceUploads.find((item) => item.id === uploadId)
      if (upload) void addUploadToCanvas(upload, point)
      return
    }

    const historyPayload = event.dataTransfer.getData(HISTORY_DRAG_TYPE)
    if (historyPayload) {
      const image = safeParseDragData<CanvasHistoryImage>(historyPayload)
      if (image?.src) void addHistoryImageToCanvas(image, point)
      return
    }

    const uri = draggedUri(event.dataTransfer)
    if (uri) {
      void addHistoryImageToCanvas({ id: `external-${Date.now()}`, src: uri, title: '外部图片', subtitle: '拖入图片', index: 0 }, point)
    }
  }

  function draggedUri(dataTransfer: DataTransfer) {
    const value = dataTransfer.getData('text/uri-list') || dataTransfer.getData('text/plain')
    return value.split(/\r?\n/).map((line) => line.trim()).find((line) => /^https?:\/\//i.test(line)) || ''
  }

  function isCanvasStageSurfaceTarget(target: EventTarget, currentTarget: EventTarget) {
    if (target === currentTarget) return true
    return target instanceof Element && Boolean(target.closest('.creative-canvas-empty'))
  }

  function handleStagePointerDown(event: ReactPointerEvent<HTMLElement>) {
    if (!isCanvasStageSurfaceTarget(event.target, event.currentTarget)) return
    pastePointRef.current = canvasPointFromClient(event.clientX, event.clientY, stageRef.current)
    setSelectedItemId(null)
    setSelectedConnectionId(null)
    stageRef.current?.focus({ preventScroll: true })
  }

  function pasteImageFilesFromClipboard(clipboardData: DataTransfer, target: EventTarget | null) {
    if (isEditableTarget(target)) return false
    const imageFiles = imageFilesFromClipboard(clipboardData)
    if (!imageFiles.length) return false
    void addFilesToCanvas(imageFiles, pastePointRef.current)
    stageRef.current?.focus({ preventScroll: true })
    return true
  }

  function handleStagePaste(event: ReactClipboardEvent<HTMLElement>) {
    if (!pasteImageFilesFromClipboard(event.clipboardData, event.target)) return
    event.preventDefault()
  }

  function handlePagePaste(event: ReactClipboardEvent<HTMLElement>) {
    if (!pasteImageFilesFromClipboard(event.clipboardData, event.target)) return
    event.preventDefault()
  }

  function handleStageContextMenu(event: ReactMouseEvent<HTMLElement>) {
    if (!isCanvasStageSurfaceTarget(event.target, event.currentTarget)) return
    event.preventDefault()
    event.stopPropagation()
    const point = canvasTextPointFromClient(event.clientX, event.clientY, stageRef.current)
    setSelectedItemId(null)
    setSelectedConnectionId(null)
    setContextMenu({ target: 'stage', point, x: event.clientX, y: event.clientY })
  }

  function addTextItemToCanvas(point: { x: number; y: number }) {
    const item = createCanvasTextItem(point, canvasItems.length)
    setCanvasItems((items) => [...items, item])
    setSelectedItemId(item.id)
    setSelectedConnectionId(null)
    setEditingTextItemId(item.id)
    setContextMenu(null)
    setMessage('已新建文字块。')
  }

  function setTextEditorRef(itemId: string, node: HTMLTextAreaElement | null) {
    if (node) textEditorRefs.current[itemId] = node
    else delete textEditorRefs.current[itemId]
  }

  function updateTextItem(itemId: string, text: string) {
    setCanvasItems((items) => updateCanvasTextItemText(items, itemId, text))
  }

  async function optimizeTextItem(itemId: string) {
    const item = canvasItems.find((entry) => entry.id === itemId)
    if (!isCanvasTextItem(item)) return
    if (!item.text.trim()) {
      setMessage('文字块为空，无法优化提示词。')
      return
    }
    setOptimizingTextItemId(itemId)
    setMessage('正在优化文字提示词...')
    try {
      const optimizedPrompt = await optimizeCanvasTextPrompt(item.text, { ratio: effectiveRatio })
      setCanvasItems((items) => updateCanvasTextItemText(items, itemId, optimizedPrompt))
      setEditingTextItemId(null)
      setMessage('文字提示词已优化。')
    } catch (error) {
      setMessage(formatError(error, '优化提示词失败。'))
    } finally {
      setOptimizingTextItemId(null)
    }
  }

  function handleItemPointerDown(event: ReactPointerEvent<HTMLElement>, item: CanvasItem, type: CanvasInteraction['type']) {
    event.preventDefault()
    event.stopPropagation()
    setSelectedItemId(item.id)
    setSelectedConnectionId(null)
    stageRef.current?.focus({ preventScroll: true })
    setContextMenu(null)
    canvasItemsRef.current = canvasItems
    const origin = { x: item.x, y: item.y, width: item.width, height: item.height, rotation: item.rotation }
    setInteraction({
      itemId: item.id,
      type,
      startClientX: event.clientX,
      startClientY: event.clientY,
      startAngle: type === 'rotate' ? pointerAngleForItem(item, event.clientX, event.clientY, stageRef.current) : undefined,
      origin,
    })
  }

  function handleItemClick(event: ReactMouseEvent<HTMLElement>, item: CanvasItem) {
    event.stopPropagation()
    if (isEditableTarget(event.target)) {
      setSelectedItemId(item.id)
      setSelectedConnectionId(null)
      return
    }
    if (connectionDraftFrom && connectionDraftFrom !== item.id) {
      addConnection(connectionDraftFrom, item.id)
      setConnectionDraftFrom(null)
      return
    }
    setSelectedItemId(item.id)
    setSelectedConnectionId(null)
    stageRef.current?.focus({ preventScroll: true })
  }

  function handleItemWheel(event: ReactWheelEvent<HTMLElement>, item: CanvasItem) {
    if (selectedItemId !== item.id || interaction || isEditableTarget(event.target)) return
    event.preventDefault()
    event.stopPropagation()
    setCanvasItems((items) => {
      const nextItems = items.map((entry) => (entry.id === item.id ? scaleCanvasItemByWheel(entry, event.deltaY, stageRef.current) : entry))
      canvasItemsRef.current = nextItems
      return nextItems
    })
  }

  function handleItemContextMenu(event: ReactMouseEvent<HTMLElement>, itemId: string) {
    event.preventDefault()
    event.stopPropagation()
    setSelectedItemId(itemId)
    setSelectedConnectionId(null)
    setContextMenu({ target: 'item', itemId, x: event.clientX, y: event.clientY })
  }

  function focusPromptTextarea() {
    if (typeof window === 'undefined') return
    window.requestAnimationFrame(() => {
      const textarea = promptTextareaRef.current
      if (!textarea) return
      textarea.focus()
      const end = textarea.value.length
      textarea.setSelectionRange(end, end)
    })
  }

  function appendReferenceLineToPrompt(itemId: string) {
    const item = canvasItems.find((entry) => entry.id === itemId)
    if (!isCanvasImageItem(item)) return
    const index = referenceIndexForItem(canvasItems, itemId)
    const line = buildReferencePromptLine(index)
    setDraftPrompt((current) => appendPromptLine(current, line))
    focusPromptTextarea()
    setMessage(`已写入提示词：@${index}`)
  }

  function markItemAsReference(itemId: string, role?: ReferenceRole, options?: { appendPrompt?: boolean }) {
    setCanvasItems((items) => items.map((item) => (
      isCanvasImageItem(item) && item.id === itemId ? { ...item, isReference: true, role: role || item.role } : item
    )))
    setMode('image-to-image')
    if (options?.appendPrompt) appendReferenceLineToPrompt(itemId)
  }

  function setItemRole(itemId: string, role: ReferenceRole) {
    setCanvasItems((items) => items.map((item) => (
      isCanvasImageItem(item) && item.id === itemId ? { ...item, role, isReference: true } : item
    )))
    setMode('image-to-image')
  }

  function toggleItemReference(itemId: string) {
    setCanvasItems((items) => items.map((item) => (
      isCanvasImageItem(item) && item.id === itemId ? { ...item, isReference: !item.isReference } : item
    )))
  }

  function rotateSelected(delta: number) {
    if (!selectedItemId) return
    setCanvasItems((items) => items.map((item) => (
      item.id === selectedItemId ? { ...item, rotation: normalizeRotation(item.rotation + delta) } : item
    )))
  }

  function removeItem(itemId: string) {
    setCanvasItems((items) => items.filter((item) => item.id !== itemId))
    setConnections((items) => items.filter((item) => item.fromId !== itemId && item.toId !== itemId))
    setSelectedItemId((current) => (current === itemId ? null : current))
    setSelectedConnectionId(null)
    setConnectionDraftFrom((current) => (current === itemId ? null : current))
    setEditingTextItemId((current) => (current === itemId ? null : current))
    setOptimizingTextItemId((current) => (current === itemId ? null : current))
    setEditingConnectionId((current) => {
      const editingConnection = connections.find((item) => item.id === current)
      return editingConnection?.fromId === itemId || editingConnection?.toId === itemId ? null : current
    })
    setContextMenu(null)
  }

  function removeConnection(connectionId: string) {
    setConnections((items) => items.filter((item) => item.id !== connectionId))
    setSelectedConnectionId((current) => (current === connectionId ? null : current))
    setEditingConnectionId((current) => (current === connectionId ? null : current))
    setConnectionLabelDraft('')
    setContextMenu(null)
  }

  function addConnection(fromId: string, toId: string) {
    setConnections((items) => {
      const exists = items.some((item) => (
        (item.fromId === fromId && item.toId === toId) || (item.fromId === toId && item.toId === fromId)
      ))
      if (exists) return items
      return [...items, { id: `${fromId}-${toId}-${Date.now()}`, fromId, toId, label: DEFAULT_CONNECTION_LABEL, text: DEFAULT_CONNECTION_LABEL }]
    })
  }

  function selectConnection(connectionId: string) {
    setSelectedConnectionId(connectionId)
    setSelectedItemId(null)
    setContextMenu(null)
    stageRef.current?.focus({ preventScroll: true })
  }

  function editConnectionLabel(connection: CanvasConnection) {
    selectConnection(connection.id)
    setEditingConnectionId(connection.id)
    setConnectionLabelDraft(connectionLabel(connection))
  }

  function beginConnectionLabelEdit(event: ReactMouseEvent<HTMLButtonElement>, connection: CanvasConnection) {
    event.preventDefault()
    event.stopPropagation()
    editConnectionLabel(connection)
  }

  function handleConnectionLabelClick(event: ReactMouseEvent<HTMLButtonElement>, connection: CanvasConnection) {
    event.preventDefault()
    event.stopPropagation()
    if (selectedConnectionId !== connection.id) {
      selectConnection(connection.id)
      return
    }
    beginConnectionLabelEdit(event, connection)
  }

  function saveConnectionLabel(connectionId: string) {
    const label = normalizeConnectionLabel(connectionLabelDraft)
    setConnections((items) => items.map((item) => (
      item.id === connectionId ? { ...item, label, text: label } : item
    )))
    setEditingConnectionId(null)
    setConnectionLabelDraft('')
  }

  function handleConnectionLabelKeyDown(event: ReactKeyboardEvent<HTMLInputElement>, _connectionId: string) {
    if (event.key !== 'Enter') return
    event.preventDefault()
    event.currentTarget.blur()
  }

  function handleCanvasKeyDown(event: ReactKeyboardEvent<HTMLElement>) {
    if (event.defaultPrevented || event.nativeEvent.isComposing || isEditableTarget(event.target)) return
    if (editingConnectionId || editingTextItemId) return

    if ((event.key === 'Enter' || event.key === 'F2') && selectedConnectionId) {
      const connection = connections.find((item) => item.id === selectedConnectionId)
      if (!connection) return
      event.preventDefault()
      event.stopPropagation()
      editConnectionLabel(connection)
      return
    }

    if (event.key !== 'Delete' && event.key !== 'Backspace') return

    if (selectedConnectionId) {
      event.preventDefault()
      event.stopPropagation()
      removeConnection(selectedConnectionId)
      return
    }

    if (selectedItemId) {
      event.preventDefault()
      event.stopPropagation()
      removeItem(selectedItemId)
    }
  }

  async function generatePromptFromCanvas() {
    if (!hasCanvasPromptContent) {
      setMessage('先在画布里添加参考、文字或连线。')
      return
    }
    const localDraft = buildCanvasPromptDraft(trimmedPrompt, canvasItems, connections, { ratio: effectiveRatio })
    setIsGeneratingCanvasPrompt(true)
    setMessage('正在根据画布整理提示词...')
    try {
      const generatedPrompt = await generateCanvasPromptFromCanvas(trimmedPrompt, canvasItems, connections, { ratio: effectiveRatio })
      setGeneratedPromptOverride(generatedPrompt)
      setDraftPrompt(generatedPrompt)
      setMessage('已根据画布生成提示词。')
    } catch {
      setGeneratedPromptOverride(localDraft)
      setDraftPrompt(localDraft)
      setMessage('AI 整理失败，已使用本地规则生成提示词。')
    } finally {
      setIsGeneratingCanvasPrompt(false)
    }
  }

  function clearCanvas() {
    setCanvasItems([])
    setConnections([])
    setSelectedItemId(null)
    setSelectedConnectionId(null)
    setGeneratedPromptOverride('')
    setConnectionDraftFrom(null)
    setEditingTextItemId(null)
    setOptimizingTextItemId(null)
    setEditingConnectionId(null)
    setConnectionLabelDraft('')
  }

  function handleHistoryDragStart(event: ReactDragEvent<HTMLElement>, image: CanvasHistoryImage) {
    event.dataTransfer.effectAllowed = 'copy'
    event.dataTransfer.setData(HISTORY_DRAG_TYPE, JSON.stringify(image))
    event.dataTransfer.setData('text/plain', image.src)
  }

  function handleUploadDragStart(event: ReactDragEvent<HTMLElement>, upload: ReferenceUpload) {
    event.dataTransfer.effectAllowed = 'copy'
    event.dataTransfer.setData(UPLOAD_DRAG_TYPE, JSON.stringify({ uploadId: upload.id }))
  }

  return (
    <main
      className={isDropActive ? 'creative-canvas-page is-page-drop-active' : 'creative-canvas-page'}
      aria-label="创作画布"
      onDragEnter={handleCanvasDragEnter}
      onDragOver={handleCanvasDragOver}
      onDragLeave={handleCanvasDragLeave}
      onDrop={handleCanvasDrop}
      onPaste={handlePagePaste}
    >
      <header className="creative-canvas-topbar">
        <div className="creative-canvas-title">
          <span>主入口</span>
          <h2>创作画布</h2>
          <p>{providerLabel(localProvider)} / {imageSize} / {modeLabel(mode)} / {referenceCanvasItems.length || taskUploadIds.length} 张参考</p>
        </div>
      </header>

      {message ? <div className="creative-canvas-message" role="status">{message}</div> : null}

      <section className="creative-canvas-workspace">
        <section
          ref={stageRef}
          className={`creative-canvas-stage ${isDropActive ? 'is-drop-active' : ''}`}
          aria-label="图片创作画布"
          tabIndex={0}
          onPointerDown={handleStagePointerDown}
          onContextMenu={handleStageContextMenu}
          onClick={(event) => {
            if (!isCanvasStageSurfaceTarget(event.target, event.currentTarget)) return
            setSelectedItemId(null)
            setSelectedConnectionId(null)
            stageRef.current?.focus({ preventScroll: true })
          }}
          onKeyDown={handleCanvasKeyDown}
          onPaste={handleStagePaste}
        >
          {canvasItems.length ? (
            <svg className="creative-connection-layer" aria-label="画布连线">
              {connections.map((connection) => {
                const from = canvasItems.find((item) => item.id === connection.fromId)
                const to = canvasItems.find((item) => item.id === connection.toId)
                if (!from || !to) return null
                const start = itemCenter(from)
                const end = itemCenter(to)
                const label = connectionLabel(connection)
                const selected = selectedConnectionId === connection.id
                return (
                  <g
                    key={connection.id}
                    className={canvasConnectionClassName(selected)}
                    role="button"
                    tabIndex={-1}
                    aria-label={`选中连接：${label}`}
                    onPointerDown={(event) => {
                      event.preventDefault()
                      event.stopPropagation()
                      selectConnection(connection.id)
                    }}
                    onClick={(event) => event.stopPropagation()}
                    onDoubleClick={(event) => {
                      event.preventDefault()
                      event.stopPropagation()
                      editConnectionLabel(connection)
                    }}
                  >
                    <line className="creative-connection-hit" x1={start.x} y1={start.y} x2={end.x} y2={end.y} />
                    <line className="creative-connection-line" x1={start.x} y1={start.y} x2={end.x} y2={end.y} />
                    <circle className="creative-connection-node" cx={end.x} cy={end.y} r="4" />
                  </g>
                )
              })}
            </svg>
          ) : null}

          {connections.map((connection) => {
            const from = canvasItems.find((item) => item.id === connection.fromId)
            const to = canvasItems.find((item) => item.id === connection.toId)
            if (!from || !to) return null
            const start = itemCenter(from)
            const end = itemCenter(to)
            const label = connectionLabel(connection)
            const isEditing = editingConnectionId === connection.id
            const selected = selectedConnectionId === connection.id
            return (
              <div
                key={`label-${connection.id}`}
                className={canvasConnectionLabelClassName(selected)}
                style={{ left: (start.x + end.x) / 2, top: (start.y + end.y) / 2 }}
                onPointerDown={(event) => event.stopPropagation()}
                onClick={(event) => event.stopPropagation()}
              >
                {isEditing ? (
                  <input
                    value={connectionLabelDraft}
                    autoFocus
                    aria-label="编辑连接关系"
                    onFocus={(event) => event.currentTarget.select()}
                    onChange={(event) => setConnectionLabelDraft(event.target.value)}
                    onBlur={() => saveConnectionLabel(connection.id)}
                    onKeyDown={(event) => handleConnectionLabelKeyDown(event, connection.id)}
                  />
                ) : (
                  <button
                    type="button"
                    title={selected ? '编辑连接关系' : '选中连接关系'}
                    aria-label={`${selected ? '编辑' : '选中'}连接关系：${label}`}
                    onClick={(event) => handleConnectionLabelClick(event, connection)}
                  >
                    {label}
                  </button>
                )}
              </div>
            )
          })}

          {canvasItems.map((item) => {
            const imageItem = isCanvasImageItem(item) ? item : null
            const textItem = isCanvasTextItem(item) ? item : null
            const src = imageItem ? imageSrcForCanvasItem(imageItem, previewUrls) : ''
            const selected = selectedItemId === item.id
            const role = imageItem ? roleMeta(imageItem.role) : null
            const isEditingText = editingTextItemId === item.id
            return (
              <article
                key={item.id}
                className={`creative-canvas-item ${selected ? 'selected' : ''} ${imageItem?.isReference ? 'is-reference' : ''} ${textItem ? 'is-text' : ''}`}
                style={canvasItemStyle(item)}
                aria-label={`${item.name}，${role?.label || '文字块'}`}
                onClick={(event) => handleItemClick(event, item)}
                onWheel={(event) => handleItemWheel(event, item)}
                onContextMenu={(event) => handleItemContextMenu(event, item.id)}
              >
                <div
                  className={`creative-canvas-item-content ${textItem ? 'creative-canvas-text-content' : ''}`}
                  style={canvasItemContentStyle(item)}
                  onPointerDown={(event) => handleItemPointerDown(event, item, 'move')}
                  onDoubleClick={(event) => {
                    if (!textItem) return
                    event.stopPropagation()
                    setEditingTextItemId(item.id)
                  }}
                >
                  {textItem ? (
                    <>
                      {isEditingText ? (
                        <textarea
                          ref={(node) => setTextEditorRef(item.id, node)}
                          className="creative-canvas-text-editor is-editing"
                          value={textItem.text}
                          aria-label="编辑文字提示词"
                          spellCheck={false}
                          placeholder="在这里输入提示词"
                          onFocus={() => {
                            setSelectedItemId(item.id)
                            setSelectedConnectionId(null)
                          }}
                          onChange={(event) => updateTextItem(item.id, event.currentTarget.value)}
                          onBlur={() => setEditingTextItemId((current) => (current === item.id ? null : current))}
                          onPointerDown={(event) => event.stopPropagation()}
                          onClick={(event) => event.stopPropagation()}
                          onKeyDown={(event) => {
                            if (event.key === 'Escape' || ((event.ctrlKey || event.metaKey) && event.key === 'Enter')) event.currentTarget.blur()
                          }}
                        />
                      ) : (
                        <div
                          className="creative-canvas-text-editor"
                          data-placeholder="在这里输入提示词"
                        >
                          {textItem.text}
                        </div>
                      )}
                      <div className="creative-canvas-item-badge">
                        <span>文字</span>
                      </div>
                    </>
                  ) : (
                    <>
                      {src ? <img src={src} alt={item.name} draggable={false} /> : <span>{extensionLabel('image/png')}</span>}
                      <div className="creative-canvas-item-badge">
                        <span>{imageItem?.isReference ? '@' : '画布'} {role?.label}</span>
                      </div>
                    </>
                  )}
                </div>
                {selected ? (
                  <div className="creative-canvas-controls" style={canvasControlStyle(item)} aria-hidden="false">
                    <button
                      type="button"
                      className="creative-rotate-handle"
                      aria-label={textItem ? '旋转文字' : '旋转图片'}
                      onPointerDown={(event) => handleItemPointerDown(event, item, 'rotate')}
                    />
                    <button
                      type="button"
                      className="creative-resize-handle"
                      aria-label={textItem ? '调整文字大小' : '调整图片大小'}
                      onPointerDown={(event) => handleItemPointerDown(event, item, 'resize')}
                    />
                  </div>
                ) : null}
              </article>
            )
          })}

          {!canvasItems.length ? (
            <div className="creative-canvas-empty">
              <strong>拖入图片开始</strong>
              <span>PNG / JPG / WEBP</span>
            </div>
          ) : null}

          <div className="creative-canvas-floating-meta" aria-label="当前参数">
            <span>{modeLabel(mode)}</span>
            <span>{effectiveRatio}</span>
            <span>{effectiveResolution}</span>
            <span>{count} 张</span>
            {connectionDraftFrom ? <span>连接中</span> : null}
          </div>
        </section>

        <aside className="creative-render-panel" aria-label="实时预览">
          <section className="creative-render-preview">
            <header>
              <strong>实时预览</strong>
              <span>{selectedImageItem ? roleMeta(selectedImageItem.role).label : latestTask ? `${latestTask.statusText} / ${latestTask.progress}%` : selectedTextItem ? '文字块' : '待生成'}</span>
            </header>
            <div className="creative-render-frame" style={{ aspectRatio: aspectRatioValue(effectiveRatio) }}>
              {renderPreviewSrc ? (
                <img src={renderPreviewSrc} alt={renderPreviewAlt} />
              ) : (
                <span>等待画布内容</span>
              )}
            </div>
          </section>

          <section className="creative-inspector" aria-label="参考语义">
            <header>
              <strong>{selectedItem ? selectedItem.name : '参考语义'}</strong>
              <span>{selectedImageItem ? roleMeta(selectedImageItem.role).note : selectedTextItem ? '可编辑提示词文字' : `${referenceCanvasItems.length} 张参考`}</span>
            </header>
            {selectedImageItem ? (
              <>
                <div className="creative-role-grid" role="group" aria-label="参考图角色">
                  {REFERENCE_ROLES.map((role) => (
                    <button
                      key={role.value}
                      type="button"
                      className={selectedImageItem.role === role.value ? 'active' : ''}
                      onClick={() => setItemRole(selectedImageItem.id, role.value)}
                    >
                      {role.label}
                    </button>
                  ))}
                </div>
                <div className="creative-inspector-actions">
                  <button type="button" onClick={() => markItemAsReference(selectedImageItem.id, undefined, { appendPrompt: true })}>@ 作为参考图</button>
                  <button type="button" onClick={() => toggleItemReference(selectedImageItem.id)}>{selectedImageItem.isReference ? '取消引用' : '加入引用'}</button>
                  <button type="button" onClick={() => setConnectionDraftFrom(selectedImageItem.id)}>连接到...</button>
                  <button type="button" onClick={() => rotateSelected(-90)}>左转 90°</button>
                  <button type="button" onClick={() => rotateSelected(90)}>右转 90°</button>
                  <button type="button" className="danger-text" onClick={() => removeItem(selectedImageItem.id)}>移除</button>
                </div>
              </>
            ) : selectedTextItem ? (
              <div className="creative-inspector-actions">
                <button type="button" onClick={() => setEditingTextItemId(selectedTextItem.id)}>编辑文字</button>
                <button type="button" onClick={() => void optimizeTextItem(selectedTextItem.id)} disabled={optimizingTextItemId === selectedTextItem.id}>{optimizingTextItemId === selectedTextItem.id ? '优化中' : '优化提示词'}</button>
                <button type="button" onClick={() => rotateSelected(-90)}>左转 90°</button>
                <button type="button" onClick={() => rotateSelected(90)}>右转 90°</button>
                <button type="button" className="danger-text" onClick={() => removeItem(selectedTextItem.id)}>移除</button>
              </div>
            ) : (
              <div className="creative-reference-summary">
                {referenceCanvasItems.length ? referenceCanvasItems.map((item) => (
                  <span key={item.id}>@ {roleMeta(item.role).label}</span>
                )) : <span>无引用</span>}
              </div>
            )}
          </section>

          <section className="creative-history-panel" aria-label="历史结果">
            <header>
              <strong>历史结果</strong>
              <span>{visibleHistory.length ? `${visibleHistory.length} 张` : '暂无'}</span>
            </header>
            <div className="creative-history-list">
              {visibleHistory.length ? visibleHistory.map((image) => (
                <button
                  key={image.id}
                  type="button"
                  draggable
                  onDragStart={(event) => handleHistoryDragStart(event, image)}
                  onClick={() => void addHistoryImageToCanvas(image, autoItemPosition(canvasItems.length))}
                  title={image.title}
                >
                  <img src={image.src} alt={image.title} draggable={false} />
                  <span>{image.subtitle}</span>
                </button>
              )) : <span className="creative-reference-empty">生成后会显示在这里</span>}
            </div>
          </section>
        </aside>

        <section className="creative-reference-strip" aria-label="素材与参考">
          <div className="creative-reference-heading">
            <strong>素材与参考</strong>
            <span>{referenceUploads.length ? `${referenceUploads.length} 张可用` : '无素材'}</span>
          </div>
          <div className="creative-reference-list">
            {visibleReferences.length ? visibleReferences.map((item) => (
              <div className="creative-reference-thumb-wrap" key={item.id}>
                <button
                  className="creative-reference-thumb"
                  type="button"
                  title={item.originalName}
                  aria-label={`加入画布：${item.originalName}`}
                  aria-pressed={markedUploadIds.includes(item.id)}
                  draggable
                  onDragStart={(event) => handleUploadDragStart(event, item)}
                  onClick={() => void addUploadToCanvas(item)}
                >
                  {previewUrls[item.id] ? <img src={previewUrls[item.id]} alt={item.originalName} draggable={false} /> : <span>{extensionLabel(item.mime)}</span>}
                </button>
                {onDeleteReferenceUpload ? (
                  <button
                    className="creative-reference-remove"
                    type="button"
                    aria-label={`删除参考图：${item.originalName}`}
                    title="删除参考图"
                    disabled={deletingUploadId === item.id}
                    onClick={(event) => void deleteReferenceUploadFromStrip(event, item)}
                  >
                    ×
                  </button>
                ) : null}
              </div>
            )) : (
              <span className="creative-reference-empty">拖入图片或使用历史结果</span>
            )}
          </div>
          <div className="creative-reference-actions">
            <button type="button" onClick={() => void generatePromptFromCanvas()} disabled={!hasCanvasPromptContent || isGeneratingCanvasPrompt}>{isGeneratingCanvasPrompt ? '整理中' : '画布生成提示词'}</button>
            <button type="button" onClick={useCanvasPrompt} disabled={!hasCanvasPromptContent}>同步到快捷生成</button>
            <button type="button" onClick={clearCanvas} disabled={!canvasItems.length}>清空画布</button>
            <button type="button" className="primary" onClick={() => void createCanvasTask()} disabled={!canCreateTask || isSubmitting}>{isSubmitting ? '生成中' : '生成'}</button>
          </div>
        </section>

        <form className="creative-canvas-composer" onSubmit={(event) => { event.preventDefault(); void createCanvasTask() }}>
          <textarea
            ref={promptTextareaRef}
            value={draftPrompt}
            onChange={(event) => setDraftPrompt(event.target.value)}
            placeholder="描述你想生成的图片，可输入 @ 来指定参考图..."
            rows={2}
          />

          <div className="creative-composer-controls">
            <div className="creative-composer-tools" aria-label="生成参数">
              <label>
                <span>模型</span>
                <select value={localProvider} onChange={(event) => setLocalProvider(event.target.value as ModelProvider)}>
                  <option value={IMAGE2_PROVIDER}>Image-2</option>
                  <option value={BANANA_PROVIDER}>Banana</option>
                </select>
              </label>

              {localProvider === BANANA_PROVIDER ? (
                <label className="creative-tool-wide">
                  <span>规格</span>
                  <select value={localBananaModel} onChange={(event) => setLocalBananaModel(event.target.value)}>
                    {BANANA_MODEL_OPTIONS.map((option) => <option key={option.id} value={option.id}>{option.label} / {option.size}</option>)}
                  </select>
                </label>
              ) : (
                <>
                  <label>
                    <span>比例</span>
                    <select value={ratio} onChange={(event) => setRatio(event.target.value)}>
                      {RATIOS.map((item) => <option key={item} value={item}>{item}</option>)}
                    </select>
                  </label>
                  <label>
                    <span>清晰度</span>
                    <select value={resolution} onChange={(event) => setResolution(event.target.value)}>
                      {RESOLUTION_TIERS.map((item) => <option key={item} value={item}>{getResolutionLabel(item)}</option>)}
                    </select>
                  </label>
                </>
              )}

              <label>
                <span>质量</span>
                <select value={quality} onChange={(event) => setQuality(event.target.value)} disabled={localProvider === BANANA_PROVIDER}>
                  {QUALITY_LEVELS.map((item) => <option key={item} value={item}>{getQualityLabel(item)}</option>)}
                </select>
              </label>
              <label>
                <span>格式</span>
                <select value={outputFormat} onChange={(event) => setOutputFormat(event.target.value)} disabled={localProvider === BANANA_PROVIDER}>
                  {OUTPUT_FORMATS.map((item) => <option key={item} value={item}>{getOutputFormatLabel(item)}</option>)}
                </select>
              </label>
              <label>
                <span>数量</span>
                <input type="number" min={1} max={8} value={count} onChange={(event) => setCount(clampNumber(event.target.value, 1, 8))} />
              </label>
            </div>

            <div className="creative-composer-actions">
              <div className="creative-segmented" role="group" aria-label="生成模式">
                <button type="button" className={mode === 'text-to-image' ? 'active' : ''} onClick={() => setMode('text-to-image')}>文生图</button>
                <button type="button" className={mode === 'image-to-image' ? 'active' : ''} onClick={() => setMode('image-to-image')}>图生图</button>
              </div>
              <button type="button" className="creative-icon-action" onClick={() => void generatePromptFromCanvas()} disabled={!hasCanvasPromptContent || isGeneratingCanvasPrompt} title="根据画布生成提示词" aria-label="根据画布生成提示词">{isGeneratingCanvasPrompt ? '…' : '✦'}</button>
              <button type="button" className="creative-icon-action" onClick={useCanvasPrompt} disabled={!hasCanvasPromptContent} title="同步到快捷生成" aria-label="同步到快捷生成">↗</button>
              <button type="submit" className="creative-submit-action" disabled={!canCreateTask || isSubmitting} aria-label="生成">{isSubmitting ? '...' : '→'}</button>
            </div>
          </div>
        </form>
      </section>

      {contextMenu ? (
        <div
          className="creative-context-menu"
          role="menu"
          style={{ left: contextMenu.x, top: contextMenu.y }}
          onClick={(event) => event.stopPropagation()}
        >
          {contextMenu.target === 'stage' ? (
            <button type="button" role="menuitem" onClick={() => addTextItemToCanvas(contextMenu.point)}>新建文字</button>
          ) : isCanvasTextItem(contextMenuItem) ? (
            <>
              <button type="button" role="menuitem" onClick={() => { setEditingTextItemId(contextMenuItem.id); setContextMenu(null) }}>编辑文字</button>
              <button type="button" role="menuitem" onClick={() => { void optimizeTextItem(contextMenuItem.id); setContextMenu(null) }} disabled={optimizingTextItemId === contextMenuItem.id}>{optimizingTextItemId === contextMenuItem.id ? '优化中' : '优化提示词'}</button>
              <button type="button" role="menuitem" className="danger-text" onClick={() => removeItem(contextMenuItem.id)}>移除</button>
            </>
          ) : isCanvasImageItem(contextMenuItem) ? (
            <>
              <button type="button" role="menuitem" onClick={() => { markItemAsReference(contextMenuItem.id, undefined, { appendPrompt: true }); setContextMenu(null) }}>@ 作为参考图</button>
              {REFERENCE_ROLES.map((role) => (
                <button key={role.value} type="button" role="menuitem" onClick={() => { setItemRole(contextMenuItem.id, role.value); setContextMenu(null) }}>{role.label}</button>
              ))}
              <button type="button" role="menuitem" onClick={() => { setConnectionDraftFrom(contextMenuItem.id); setContextMenu(null) }}>连接到...</button>
              <button type="button" role="menuitem" className="danger-text" onClick={() => removeItem(contextMenuItem.id)}>移除</button>
            </>
          ) : null}
        </div>
      ) : null}
    </main>
  )
}
