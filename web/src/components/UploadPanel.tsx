import { useEffect, useState } from 'react'
import type { ReferenceUpload } from '../types'
import { formatBytes } from '../lib/format'
import { getReferenceUploadBlob } from '../api/uploads'

type Props = {
  uploads: ReferenceUpload[]
  primaryUploadId: string
  onPrimaryChange: (id: string) => void
  onUpload: (files: File[]) => void
  onDelete: (id: string) => void
}

export function UploadPanel({ uploads, primaryUploadId, onPrimaryChange, onUpload, onDelete }: Props) {
  const [previews, setPreviews] = useState<Record<string, string>>({})

  useEffect(() => {
    let disposed = false
    const created: string[] = []
    async function load() {
      const entries = await Promise.all(uploads.map(async (item) => {
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
      setPreviews(Object.fromEntries(entries))
    }
    void load()
    return () => {
      disposed = true
      created.forEach((url) => URL.revokeObjectURL(url))
    }
  }, [uploads])

  const primary = uploads.find((item) => item.id === primaryUploadId) || uploads[0]

  return (
    <section className="form-section upload-section">
      <div className="section-title upload-title-row">
        <span>参考图</span>
        <small>{uploads.length ? `当前 ${uploads.length}/8 张` : '图生图最多 8 张'}</small>
      </div>
      <label className="upload-dropzone">
        <input type="file" accept="image/png,image/jpeg,image/webp" multiple onChange={(event) => { onUpload(Array.from(event.target.files || [])); event.currentTarget.value = '' }} />
        <span>上传参考图</span>
        <small>PNG / JPG / WEBP，单张不超过 12MB</small>
      </label>

      {uploads.length ? (
        <div className="merge-direction-box">
          <strong>合一方向：{uploads.length > 1 ? `以「${primary?.originalName || '第 1 张'}」为主，融合其他参考图` : '当前参考图作为主图'}</strong>
          <span>点击「设为主图」可以指定往哪张图上融合；主图会排在请求图片第一位。</span>
        </div>
      ) : null}

      <div className="upload-list reference-grid">
        {uploads.map((item, index) => {
          const isPrimary = item.id === (primary?.id || primaryUploadId)
          return (
            <article className={`reference-card ${isPrimary ? 'primary-ref' : ''}`} key={item.id}>
              <div className="reference-thumb">
                {previews[item.id] ? <img src={previews[item.id]} alt={item.originalName} /> : <span>{extensionLabel(item.mime)}</span>}
              </div>
              <div className="reference-info">
                <strong>{item.originalName}</strong>
                <small>{item.mime} · {formatBytes(item.size)}</small>
                <div className="reference-role-row">
                  <span>{isPrimary ? '主图 / 被融合目标' : `参考图 ${index + 1}`}</span>
                  {!isPrimary ? <button type="button" onClick={() => onPrimaryChange(item.id)}>设为主图</button> : null}
                  <button type="button" className="danger-text" onClick={() => onDelete(item.id)}>删除</button>
                </div>
              </div>
            </article>
          )
        })}
      </div>
    </section>
  )
}

function extensionLabel(mime: string) {
  if (mime.includes('jpeg')) return 'JPG'
  if (mime.includes('webp')) return 'WEBP'
  return 'PNG'
}
