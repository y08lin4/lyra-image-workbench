import type { Task, TaskResult } from '../../types'
import { imageSrcForCanvasItem, roleMeta } from './data'
import type { CanvasHistoryImage, CanvasItem } from './types'

export type CanvasPreviewSource = 'canvas' | 'history' | 'task' | 'empty'

export type CanvasPreviewState = {
  source: CanvasPreviewSource
  title: string
  status: string
  src: string
  alt: string
}

export function latestSuccessfulTaskResult(task?: Task): TaskResult | undefined {
  const okResults = task?.results.filter((item) => item.ok && item.imageUrl) || []
  return okResults[okResults.length - 1]
}

export function buildCanvasPreviewState(options: {
  selectedItem?: CanvasItem | null
  selectedHistoryImage?: CanvasHistoryImage | null
  latestTask?: Task
  previewUrls: Record<string, string>
}): CanvasPreviewState {
  const { selectedItem, selectedHistoryImage, latestTask, previewUrls } = options

  if (selectedItem?.type === 'image') {
    const src = imageSrcForCanvasItem(selectedItem, previewUrls)
    if (src) {
      return {
        source: 'canvas',
        title: '实时预览',
        status: roleMeta(selectedItem.role).label,
        src,
        alt: selectedItem.name,
      }
    }
  }

  if (selectedItem?.type === 'text') {
    return {
      source: 'empty',
      title: '实时预览',
      status: '文字块',
      src: '',
      alt: selectedItem.name,
    }
  }

  if (selectedHistoryImage?.src) {
    return {
      source: 'history',
      title: '历史预览',
      status: selectedHistoryImage.subtitle || '历史结果',
      src: selectedHistoryImage.src,
      alt: selectedHistoryImage.title || '历史结果',
    }
  }

  const latestResult = latestSuccessfulTaskResult(latestTask)
  if (latestResult?.imageUrl) {
    return {
      source: 'task',
      title: '实时预览',
      status: latestTask ? `${latestTask.statusText} / ${latestTask.progress}%` : latestResult.statusText,
      src: latestResult.imageUrl,
      alt: '最新生成结果',
    }
  }

  return {
    source: 'empty',
    title: '实时预览',
    status: latestTask ? `${latestTask.statusText} / ${latestTask.progress}%` : '待生成',
    src: '',
    alt: selectedItem?.name || '画布预览',
  }
}

export function isHistoryPreviewSelected(image: CanvasHistoryImage, selectedHistoryImage?: CanvasHistoryImage | null) {
  if (!selectedHistoryImage) return false
  return image.id === selectedHistoryImage.id || image.src === selectedHistoryImage.src
}
