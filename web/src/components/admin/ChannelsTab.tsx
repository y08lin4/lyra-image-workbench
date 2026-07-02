import { readNumberInput, type NumericInputValue } from './adminHelpers'

export type AdminImageChannelModelDraft = {
  id: string
  label: string
  enabled: boolean
  price: NumericInputValue
  ratioSelectable: boolean
  defaultResolution: string
}

export type AdminImageChannelDraft = {
  type: string
  name: string
  baseURL: string
  keySet?: boolean
  keyPreview?: string
  key: string
  clearKey: boolean
  enabled: boolean
  models: AdminImageChannelModelDraft[]
}

type ChannelsTabProps = {
  channels: AdminImageChannelDraft[]
  saving: boolean
  onChange: (channels: AdminImageChannelDraft[]) => void
  onSave: () => void
}

const NEW_MODEL: AdminImageChannelModelDraft = {
  id: 'image-2',
  label: 'image-2',
  enabled: true,
  price: 1,
  ratioSelectable: false,
  defaultResolution: 'auto',
}

export function ChannelsTab({ channels, saving, onChange, onSave }: ChannelsTabProps) {
  function updateChannel(index: number, patch: Partial<AdminImageChannelDraft>) {
    onChange(channels.map((channel, currentIndex) => currentIndex === index ? { ...channel, ...patch } : channel))
  }

  function updateModel(channelIndex: number, modelIndex: number, patch: Partial<AdminImageChannelModelDraft>) {
    onChange(channels.map((channel, currentIndex) => {
      if (currentIndex !== channelIndex) return channel
      return {
        ...channel,
        models: channel.models.map((model, currentModelIndex) => currentModelIndex === modelIndex ? { ...model, ...patch } : model),
      }
    }))
  }

  function addChannel() {
    onChange([
      ...channels,
      {
        type: 'openai-compatible',
        name: `custom-${channels.length + 1}`,
        baseURL: '',
        key: '',
        clearKey: false,
        enabled: true,
        models: [{ ...NEW_MODEL }],
      },
    ])
  }

  function removeChannel(index: number) {
    if (!window.confirm('确认删除这个图片渠道？')) return
    onChange(channels.filter((_, currentIndex) => currentIndex !== index))
  }

  function addModel(channelIndex: number) {
    onChange(channels.map((channel, currentIndex) => currentIndex === channelIndex
      ? { ...channel, models: [...channel.models, { ...NEW_MODEL, id: '', label: '' }] }
      : channel))
  }

  function removeModel(channelIndex: number, modelIndex: number) {
    onChange(channels.map((channel, currentIndex) => currentIndex === channelIndex
      ? { ...channel, models: channel.models.filter((_, currentModelIndex) => currentModelIndex !== modelIndex) }
      : channel))
  }

  return (
    <section className="admin-tab-panel admin-channels-panel" id="admin-panel-channels" role="tabpanel" aria-labelledby="admin-tab-channels">
      <fieldset className="admin-billing-box">
        <legend>模型渠道 / 图片渠道</legend>
        <div className="admin-section-title">
          <div>
            <strong>图片生成渠道</strong>
            <span>配置渠道启用状态、上游地址、密钥和渠道内可用模型。</span>
          </div>
          <button type="button" onClick={addChannel}>新增渠道</button>
        </div>

        {channels.length ? channels.map((channel, channelIndex) => (
          <fieldset className="admin-billing-box" key={`${channel.name || 'channel'}-${channelIndex}`}>
            <legend>{channel.name || `渠道 ${channelIndex + 1}`}</legend>
            <div className="admin-section-title">
              <div>
                <strong>{channel.name || '未命名渠道'}</strong>
                <span>{channel.enabled ? '已启用' : '已停用'} · {channel.type || '未设置类型'}</span>
              </div>
              <span className={`admin-key-status ${channel.keySet && !channel.clearKey ? 'ready' : 'missing'}`}>
                {channel.clearKey ? '将清空 Key' : channel.keySet ? `Key 已设置（${channel.keyPreview || '已隐藏'}）` : 'Key 未设置'}
              </span>
            </div>

            <label className="check-row admin-debug-toggle admin-debug-row">
              <input type="checkbox" checked={channel.enabled} onChange={(event) => updateChannel(channelIndex, { enabled: event.target.checked })} />
              <span>启用此图片渠道</span>
            </label>

            <div className="admin-billing-grid">
              <label>渠道名称<input value={channel.name} onChange={(event) => updateChannel(channelIndex, { name: event.target.value })} placeholder="image-2" /></label>
              <label>渠道 type<input value={channel.type} onChange={(event) => updateChannel(channelIndex, { type: event.target.value })} placeholder="openai-compatible" /></label>
              <label>Base URL<input value={channel.baseURL} onChange={(event) => updateChannel(channelIndex, { baseURL: event.target.value })} placeholder="https://api.example.com/v1" /></label>
              <label>渠道 Key<input type="password" value={channel.key} onChange={(event) => updateChannel(channelIndex, { key: event.target.value, clearKey: false })} placeholder={channel.keySet ? '已保存，输入新 Key 可覆盖' : '仅保存，不会明文显示'} autoComplete="new-password" /></label>
            </div>

            <div className="admin-inline-notes">
              <label className="check-row">
                <input type="checkbox" checked={channel.clearKey} disabled={!channel.keySet} onChange={(event) => updateChannel(channelIndex, { clearKey: event.target.checked, key: event.target.checked ? '' : channel.key })} />
                <span>清空已保存的渠道 Key</span>
              </label>
              <div className="status-line">保存时留空 Key 会保留已有密钥；填写新 Key 会覆盖。</div>
            </div>

            <div className="admin-section-title">
              <div>
                <strong>模型</strong>
                <span>配置模型 ID、展示名称、价格和默认分辨率。</span>
              </div>
              <button type="button" onClick={() => addModel(channelIndex)}>新增模型</button>
            </div>

            {channel.models.map((model, modelIndex) => (
              <fieldset className="admin-billing-box" key={`${model.id || 'model'}-${modelIndex}`}>
                <legend>{model.label || model.id || `模型 ${modelIndex + 1}`}</legend>
                <label className="check-row admin-debug-toggle admin-debug-row">
                  <input type="checkbox" checked={model.enabled} onChange={(event) => updateModel(channelIndex, modelIndex, { enabled: event.target.checked })} />
                  <span>启用此模型</span>
                </label>
                <div className="admin-billing-grid">
                  <label>模型 ID<input value={model.id} onChange={(event) => updateModel(channelIndex, modelIndex, { id: event.target.value })} placeholder="image-2" /></label>
                  <label>展示名称<input value={model.label} onChange={(event) => updateModel(channelIndex, modelIndex, { label: event.target.value })} placeholder="image-2" /></label>
                  <label>价格倍率<input type="number" min={0} value={model.price} onChange={(event) => updateModel(channelIndex, modelIndex, { price: readNumberInput(event.target.value) })} /></label>
                  <label>默认分辨率<input value={model.defaultResolution} onChange={(event) => updateModel(channelIndex, modelIndex, { defaultResolution: event.target.value })} placeholder="auto" /></label>
                </div>
                <div className="admin-inline-notes">
                  <label className="check-row">
                    <input type="checkbox" checked={model.ratioSelectable} onChange={(event) => updateModel(channelIndex, modelIndex, { ratioSelectable: event.target.checked })} />
                    <span>允许用户选择比例 / 尺寸</span>
                  </label>
                  <button type="button" onClick={() => removeModel(channelIndex, modelIndex)} disabled={channel.models.length <= 1}>删除模型</button>
                </div>
              </fieldset>
            ))}

            <div className="admin-billing-actions">
              <button type="button" onClick={() => removeChannel(channelIndex)}>删除渠道</button>
            </div>
          </fieldset>
        )) : <div className="info">暂无图片渠道，点击“新增渠道”创建。</div>}

        <div className="admin-billing-actions">
          <button className="primary" type="button" onClick={onSave} disabled={saving || channels.length === 0}>
            {saving ? '保存中...' : '保存图片渠道配置'}
          </button>
        </div>
      </fieldset>
    </section>
  )
}