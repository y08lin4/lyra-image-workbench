export type ReferenceRole = 'reference' | 'subject' | 'style'

export type CanvasPoint = {
  x: number
  y: number
}

type CanvasItemBase = CanvasPoint & {
  id: string
  name: string
  width: number
  height: number
  rotation: number
}

export type CanvasHistoryImage = {
  id: string
  src: string
  title: string
  subtitle: string
  taskId?: string
  index: number
  prompt?: string
}

export type CanvasImageItem = CanvasItemBase & {
  type: 'image'
  uploadId?: string
  resultSrc?: string
  localPreviewUrl?: string
  naturalWidth?: number
  naturalHeight?: number
  aspectRatio?: number
  role: ReferenceRole
  isReference: boolean
}

export type CanvasTextItem = CanvasItemBase & {
  type: 'text'
  text: string
  role: ReferenceRole
  isReference: boolean
  uploadId?: undefined
  resultSrc?: undefined
  localPreviewUrl?: undefined
}

export type CanvasItem = CanvasImageItem | CanvasTextItem

export type CanvasConnection = {
  id: string
  fromId: string
  toId: string
  label?: string
  text?: string
}

export type CanvasInteraction = {
  itemId: string
  type: 'move' | 'resize' | 'rotate'
  startClientX: number
  startClientY: number
  startAngle?: number
  origin: Pick<CanvasItem, 'x' | 'y' | 'width' | 'height' | 'rotation'>
}

export type CanvasContextMenu =
  | {
      target: 'item'
      itemId: string
      x: number
      y: number
    }
  | {
      target: 'stage'
      point: CanvasPoint
      x: number
      y: number
    }

export type ReferenceRoleMeta = {
  value: ReferenceRole
  label: string
  note: string
}

export const REFERENCE_ROLES: ReferenceRoleMeta[] = [
  { value: 'reference', label: '参考', note: '整体画面、内容和可借鉴特征' },
  { value: 'subject', label: '主体', note: '主体身份、轮廓和姿态' },
  { value: 'style', label: '风格', note: '画风、质感和调色' },
]