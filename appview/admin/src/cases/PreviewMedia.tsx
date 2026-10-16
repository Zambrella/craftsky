import { useEffect, useState } from 'react'

export function PreviewImageView({ src, alt, className }: { src?: string; alt: string; className?: string }) {
  const [failed, setFailed] = useState(false)
  useEffect(() => setFailed(false), [src])
  if (!src || !safeURL(src) || failed) return <span className="preview-media-unavailable" role="status">Image unavailable{alt && ` · ${alt}`}</span>
  return <img src={src} alt={alt} className={className} loading="lazy" referrerPolicy="no-referrer" onError={() => setFailed(true)}/>
}


export function safeURL(value: string): string | undefined {
 try {
  const url=new URL(value,window.location.origin)
  if (url.username || url.password) return undefined
  if (url.protocol==='https:' || url.origin===window.location.origin && value.startsWith('/')) return url.href
 } catch { /* Invalid destinations are displayed without a link. */ }
 return undefined
}
