export const THEME_OPTIONS = [
  { key: 'white', label: '白色' },
  { key: 'black', label: '黑色' },
  { key: 'green', label: '绿色' },
  { key: 'pink', label: '粉色' },
  { key: 'blue', label: '蓝色' },
  { key: 'white-blue', label: '白蓝' },
  { key: 'black-purple', label: '黑紫' },
  { key: 'white-purple', label: '白紫' },
] as const

export type ThemeMode = (typeof THEME_OPTIONS)[number]['key']

export const DEFAULT_THEME: ThemeMode = 'white-blue'

const themeKeys = new Set<string>(THEME_OPTIONS.map((theme) => theme.key))

export function isThemeMode(value: string | null): value is ThemeMode {
  return Boolean(value && themeKeys.has(value))
}

export function resolveTheme(value: string | null, prefersDark = false): ThemeMode {
  if (isThemeMode(value)) return value
  if (value === 'dark') return 'black'
  if (value === 'light') return 'white'
  return prefersDark ? 'black' : DEFAULT_THEME
}

export function nextTheme(current: ThemeMode): ThemeMode {
  const index = THEME_OPTIONS.findIndex((theme) => theme.key === current)
  return THEME_OPTIONS[(index + 1) % THEME_OPTIONS.length]?.key || DEFAULT_THEME
}
