const REPO_URL = 'https://github.com/y08lin4/lyra-image-workbench'

export function GitHubLink({ compact = false }: { compact?: boolean }) {
  return (
    <a className={`github-link ghost-link ${compact ? 'compact' : ''}`} href={REPO_URL} target="_blank" rel="noreferrer" aria-label="打开 GitHub 项目地址" title="GitHub 项目地址">
      <svg viewBox="0 0 16 16" aria-hidden="true" focusable="false">
        <path d="M8 0C3.58 0 0 3.64 0 8.13c0 3.59 2.29 6.63 5.47 7.71.4.08.55-.18.55-.39 0-.19-.01-.83-.01-1.51-2.01.38-2.53-.5-2.69-.95-.09-.23-.48-.95-.82-1.14-.28-.15-.68-.52-.01-.53.63-.01 1.08.59 1.23.83.72 1.23 1.87.88 2.33.67.07-.53.28-.88.51-1.08-1.78-.21-3.64-.91-3.64-4.02 0-.89.31-1.62.82-2.19-.08-.21-.36-1.04.08-2.16 0 0 .67-.22 2.2.84A7.5 7.5 0 0 1 8 3.94c.68 0 1.36.09 2 .28 1.53-1.06 2.2-.84 2.2-.84.44 1.12.16 1.95.08 2.16.51.57.82 1.3.82 2.19 0 3.12-1.91 3.81-3.73 4.02.29.26.55.76.55 1.54 0 1.11-.01 2-.01 2.27 0 .22.15.48.55.39A8.03 8.03 0 0 0 16 8.13C16 3.64 12.42 0 8 0Z" />
      </svg>
      {compact ? null : <span>GitHub</span>}
    </a>
  )
}
