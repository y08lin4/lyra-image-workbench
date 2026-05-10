import { useEffect, useMemo, useRef, useState } from 'react'

export type ImageDimensions = { width: number; height: number }

type StageSize = { width: number; height: number }

type Props = {
  src: string
  title: string
  onDimensions: (dimensions: ImageDimensions) => void
}

export function PreviewImageStage({ src, title, onDimensions }: Props) {
  const stageRef = useRef<HTMLDivElement | null>(null)
  const [stageSize, setStageSize] = useState<StageSize>({ width: 0, height: 0 })
  const [dimensions, setDimensions] = useState<ImageDimensions>()

  useEffect(() => {
    setDimensions(undefined)
  }, [src])

  useEffect(() => {
    const node = stageRef.current
    if (!node) return

    const measure = () => {
      const rect = node.getBoundingClientRect()
      setStageSize({
        width: Math.max(0, Math.floor(rect.width)),
        height: Math.max(0, Math.floor(rect.height)),
      })
    }

    measure()
    if (typeof ResizeObserver !== 'undefined') {
      const observer = new ResizeObserver(measure)
      observer.observe(node)
      return () => observer.disconnect()
    }

    window.addEventListener('resize', measure)
    window.visualViewport?.addEventListener('resize', measure)
    return () => {
      window.removeEventListener('resize', measure)
      window.visualViewport?.removeEventListener('resize', measure)
    }
  }, [])

  const imageStyle = useMemo(() => getImageStyle(dimensions, stageSize), [dimensions, stageSize])

  function handleLoad(event: React.SyntheticEvent<HTMLImageElement>) {
    const next = {
      width: event.currentTarget.naturalWidth,
      height: event.currentTarget.naturalHeight,
    }
    setDimensions(next)
    onDimensions(next)
  }

  return (
    <div ref={stageRef} className="image-preview-stage">
      <img className="image-preview-img" src={src} alt={title} style={imageStyle} onLoad={handleLoad} />
    </div>
  )
}

function getImageStyle(dimensions: ImageDimensions | undefined, stageSize: StageSize): React.CSSProperties | undefined {
  if (!dimensions?.width || !dimensions.height || !stageSize.width || !stageSize.height) return undefined
  const scale = Math.min(stageSize.width / dimensions.width, stageSize.height / dimensions.height, 1)
  return {
    width: `${Math.max(1, Math.floor(dimensions.width * scale))}px`,
    height: `${Math.max(1, Math.floor(dimensions.height * scale))}px`,
  }
}
