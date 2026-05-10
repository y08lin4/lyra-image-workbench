export type PreviewAction = () => void | string | Promise<void | string>

type Props = {
  onCopyImage?: PreviewAction
  onCopyUrl?: PreviewAction
  onDownload?: PreviewAction
  onUseAsReference?: PreviewAction
  onNotice: (value: string) => void
}

export function PreviewToolbar({ onDownload, onCopyImage, onCopyUrl, onUseAsReference, onNotice }: Props) {
  async function runAction(action: PreviewAction | undefined, fallback: string) {
    if (!action) return
    try {
      const result = await action()
      onNotice(typeof result === 'string' && result ? result : fallback)
    } catch (err) {
      onNotice(err instanceof Error ? err.message : '????')
    }
  }

  return (
    <div className="image-preview-toolbar">
      {onDownload ? <button type="button" onClick={() => void runAction(onDownload, '?????')}>??</button> : null}
      {onCopyImage ? <button type="button" onClick={() => void runAction(onCopyImage, '?????')}>????</button> : null}
      {onCopyUrl ? <button type="button" onClick={() => void runAction(onCopyUrl, '?????')}>????</button> : null}
      {onUseAsReference ? <button type="button" onClick={() => void runAction(onUseAsReference, '??????')}>?????</button> : null}
    </div>
  )
}
