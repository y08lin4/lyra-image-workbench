export function isEditableTarget(target: EventTarget | null) {
  if (!(target instanceof Element)) return false
  return Boolean(target.closest('input, textarea, select, [contenteditable="true"], [contenteditable="plaintext-only"], [contenteditable=""]'))
}

export function canvasConnectionClassName(selected: boolean) {
  return selected ? 'creative-connection selected' : 'creative-connection'
}

export function canvasConnectionLabelClassName(selected: boolean) {
  return selected ? 'creative-connection-label selected' : 'creative-connection-label'
}
