import type { ReferenceUpload } from '../types'
import { formatBytes } from '../lib/format'

export function UploadPanel({ uploads, onUpload, onDelete }: { uploads: ReferenceUpload[]; onUpload: (files: File[]) => void; onDelete: (id: string) => void }) {
  return (
    <section className="form-section upload-section">
      <div className="section-title">
        <span>参考图</span>
        <small>图生图最多 8 张</small>
      </div>
      <label className="upload-dropzone">
        <input type="file" accept="image/png,image/jpeg,image/webp" multiple onChange={(e) => onUpload(Array.from(e.target.files || []))} />
        <span>上传参考图</span>
        <small>PNG / JPG / WEBP，单张不超过 12MB</small>
      </label>
      <div className="upload-list">
        {uploads.map((item) => (
          <div className="upload-item" key={item.id}>
            <span>{item.originalName}</span>
            <small>{item.mime} · {formatBytes(item.size)}</small>
            <button type="button" onClick={() => onDelete(item.id)}>删除</button>
          </div>
        ))}
      </div>
    </section>
  )
}
