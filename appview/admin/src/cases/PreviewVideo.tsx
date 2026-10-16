import { useEffect, useRef, useState } from 'react'
import { PreviewImageView, safeURL } from './PreviewMedia'
import type { Preview } from './model'

type Video = NonNullable<Preview['video']>
export function PreviewVideo({ video }: { video: Video }) {
  const [loaded, setLoaded] = useState(false)
  return <figure className="preview-video">{loaded ? <Player video={video}/> : <>{video.thumbnail && <PreviewImageView src={video.thumbnail} alt="Video thumbnail"/>}<button className="secondary" onClick={() => setLoaded(true)}>Load video</button><p className="hint">Video loads on request. Use the player controls to start playback.</p></>}{video.alt && <figcaption className="preserve">{video.alt}</figcaption>}</figure>
}
function Player({ video }: { video: Video }) {
  const element = useRef<HTMLVideoElement>(null)
  const [error, setError] = useState('')
  useEffect(() => {
    const player = element.current!
    let disposed = false
    let hls: import('hls.js').default | undefined
    const playlist = safeURL(video.playlist)
    if (!playlist) { setError('This video is unavailable.'); return }
    if (player.canPlayType('application/vnd.apple.mpegurl')) player.src = playlist
    else void import('hls.js').then(({ default: Hls }) => {
      if (disposed) return
      if (!Hls.isSupported()) { setError('This browser cannot play this video.'); return }
      hls = new Hls({ enableWorker: false, maxBufferLength: 30, backBufferLength: 30 })
      hls.on(Hls.Events.ERROR, (_event, data) => { if (data.fatal && !disposed) { setError('The video could not be loaded.'); hls?.destroy() } })
      hls.loadSource(playlist)
      hls.attachMedia(player)
    }).catch(() => { if (!disposed) setError('The video player could not be loaded.') })
    return () => {
      disposed = true
      hls?.destroy()
      if (player.getAttribute('src')) { player.pause(); player.removeAttribute('src'); player.load() }
    }
  }, [video.playlist])
  return <><video ref={element} aria-label="Video preview" controls playsInline preload="none" crossOrigin="anonymous" onError={() => setError('The video could not be loaded.')}/>{error && <p role="status">{error}</p>}</>
}
