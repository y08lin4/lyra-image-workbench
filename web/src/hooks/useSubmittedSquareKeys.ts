import { useCallback, useState } from 'react'

const SUBMITTED_SQUARE_KEYS_STORAGE = 'lyra:prompt-square:submitted-result-keys:v1'
const MAX_SUBMITTED_SQUARE_KEYS = 500

export function resultSquareKey(taskId: string, imageIndex: number) {
  return `${taskId}:${imageIndex}`
}

export function useSubmittedSquareKeys() {
  const [submittedSquareKeys, setSubmittedSquareKeys] = useState<Set<string>>(() => loadSubmittedSquareKeys())

  const markSubmittedSquareKey = useCallback((taskId: string, imageIndex: number) => {
    const key = resultSquareKey(taskId, imageIndex)
    setSubmittedSquareKeys((prev) => {
      const next = new Set(prev)
      next.add(key)
      saveSubmittedSquareKeys(next)
      return next
    })
    return key
  }, [])

  return { submittedSquareKeys, markSubmittedSquareKey }
}

function loadSubmittedSquareKeys() {
  if (typeof window === 'undefined') return new Set<string>()
  try {
    const raw = window.localStorage.getItem(SUBMITTED_SQUARE_KEYS_STORAGE)
    const items = raw ? JSON.parse(raw) : []
    return new Set<string>(Array.isArray(items) ? items.filter((item) => typeof item === 'string') : [])
  } catch {
    return new Set<string>()
  }
}

function saveSubmittedSquareKeys(keys: Set<string>) {
  try {
    window.localStorage.setItem(SUBMITTED_SQUARE_KEYS_STORAGE, JSON.stringify(Array.from(keys).slice(-MAX_SUBMITTED_SQUARE_KEYS)))
  } catch {
    // localStorage can be unavailable in hardened browser contexts.
  }
}
