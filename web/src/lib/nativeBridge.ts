type NativeBridge = {
  copyText?: (text: string) => string | Promise<string>
  copyImage?: (dataUrl: string, fileName: string) => string | Promise<string>
  saveImage?: (dataUrl: string, fileName: string) => string | Promise<string>
}

type NativeResult = {
  handled: boolean
  ok: boolean
  message?: string
}

declare global {
  interface Window {
    AIImageApp?: NativeBridge
  }
}

export async function nativeCopyText(text: string): Promise<NativeResult> {
  return callNativeBridge('copyText', text)
}

export async function nativeCopyImage(src: string, fileName: string): Promise<NativeResult> {
  if (!hasNativeMethod('copyImage')) return { handled: false, ok: false }
  try {
    const dataUrl = await imageSourceToDataUrl(src)
    return callNativeBridge('copyImage', dataUrl, fileName)
  } catch (error) {
    return { handled: true, ok: false, message: error instanceof Error ? error.message : '读取图片失败' }
  }
}

export async function nativeSaveImage(src: string, fileName: string): Promise<NativeResult> {
  if (!hasNativeMethod('saveImage')) return { handled: false, ok: false }
  try {
    const dataUrl = await imageSourceToDataUrl(src)
    return callNativeBridge('saveImage', dataUrl, fileName)
  } catch (error) {
    return { handled: true, ok: false, message: error instanceof Error ? error.message : '读取图片失败' }
  }
}

function hasNativeMethod(method: keyof NativeBridge) {
  return typeof window.AIImageApp?.[method] === 'function'
}

async function callNativeBridge(method: 'copyText', text: string): Promise<NativeResult>
async function callNativeBridge(method: 'copyImage' | 'saveImage', dataUrl: string, fileName: string): Promise<NativeResult>
async function callNativeBridge(method: keyof NativeBridge, ...args: string[]): Promise<NativeResult> {
  const bridge = window.AIImageApp
  if (!bridge || typeof bridge[method] !== 'function') return { handled: false, ok: false }

  try {
    // Android WebView 的 @JavascriptInterface 方法必须从注入对象本身调用。
    const result = method === 'copyText'
      ? String(await bridge.copyText!(args[0] || '') || '')
      : method === 'copyImage'
        ? String(await bridge.copyImage!(args[0] || '', args[1] || '') || '')
        : String(await bridge.saveImage!(args[0] || '', args[1] || '') || '')

    if (result === 'ok' || result.startsWith('ok:')) return { handled: true, ok: true }
    return { handled: true, ok: false, message: result.replace(/^error:/, '') || 'App 原生操作失败' }
  } catch (error) {
    return { handled: true, ok: false, message: error instanceof Error ? error.message : 'App 原生操作失败' }
  }
}

async function imageSourceToDataUrl(src: string) {
  if (src.startsWith('data:image/')) return src
  const response = await fetch(src, { cache: 'no-store' })
  if (!response.ok) throw new Error(`读取图片失败：HTTP ${response.status}`)
  const blob = await response.blob()
  if (blob.type && !blob.type.startsWith('image/')) throw new Error('读取到的不是图片文件')
  return blobToDataUrl(blob, blob.type || mimeFromFileName(src) || 'image/png')
}

function blobToDataUrl(blob: Blob, fallbackMime: string) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onerror = () => reject(reader.error || new Error('图片读取失败'))
    reader.onload = () => {
      const result = String(reader.result || '')
      if (result.startsWith('data:')) {
        resolve(result)
        return
      }
      reject(new Error('图片转码失败'))
    }
    reader.readAsDataURL(blob.type ? blob : new Blob([blob], { type: fallbackMime }))
  })
}

function mimeFromFileName(value: string) {
  const clean = value.split(/[?#]/)[0].toLowerCase()
  if (clean.endsWith('.jpg') || clean.endsWith('.jpeg')) return 'image/jpeg'
  if (clean.endsWith('.webp')) return 'image/webp'
  if (clean.endsWith('.gif')) return 'image/gif'
  if (clean.endsWith('.png')) return 'image/png'
  return ''
}
