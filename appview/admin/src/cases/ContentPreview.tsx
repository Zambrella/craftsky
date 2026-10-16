import { useEffect, useRef, useState, type ReactNode } from 'react'
import { PreviewVideo } from './PreviewVideo'
import { PreviewImageView, safeURL } from './PreviewMedia'
import { date, label, type Preview, type PreviewImage } from './model'

export function ContentPreview({ preview }: { preview: Preview | null }) {
  return <section className="card content-preview"><h2>Content preview</h2>{preview?.state === 'available' ? <SubjectPreview key={`${preview.subjectDid}:${preview.createdAt}:${preview.title}`} preview={preview}/> : <p>{preview?.state === 'restricted' ? 'Restricted content. Use the specialist review workflow.' : 'Content is unavailable. It may have changed or may not have a safe captured version.'}</p>}</section>
}

function SubjectPreview({ preview: p }: { preview: Preview }) {
  const [expanded, setExpanded] = useState<{ image: PreviewImage; index: number; trigger: HTMLElement } | null>(null)
  return <article className="subject-preview">
    {p.banner && <PreviewImageView src={p.banner} alt="Account banner" className="preview-banner"/>}
    <div className="preview-author">{p.author?.avatar && <PreviewImageView src={p.author.avatar} alt="Account avatar" className="preview-avatar"/>}<div><span className="badge">{label(p.subjectType || 'Content')}</span><strong>{p.author?.displayName || p.title || p.author?.did || p.subjectDid || 'Content'}</strong><span className="did">{p.author?.did || p.subjectDid}</span>{p.createdAt && <small>{date(p.createdAt)}</small>}</div></div>
    {!!p.crafts?.length && <div className="preview-tags">{p.crafts.map(craft => <span className="badge" key={craft}>{label(craft)}</span>)}</div>}
    <p className="preserve">{p.text || 'No text accompanies this subject.'}</p>
    {!!p.images?.length && <div className="preview-images">{p.images.map((image, index) => <figure key={image.fullsize || index}><button className="preview-image-button" aria-label={`Enlarge image ${index + 1}`} onClick={event => setExpanded({ image, index, trigger: event.currentTarget })}><PreviewImageView src={image.fullsize || image.thumb} alt={image.alt || `Image ${index + 1}`}/></button>{image.alt && <figcaption className="preserve">{image.alt}</figcaption>}</figure>)}</div>}
    {p.video && <PreviewVideo key={p.video.playlist} video={p.video}/>}
    {p.project && <section className="preview-context"><h3>{p.project.common.title || 'Project details'}</h3><p>{label(p.project.common.craftType.split('#').at(-1) || 'Craft')}{p.project.common.status && ` · ${label(p.project.common.status.split('#').at(-1) || '')}`}</p>{p.project.common.pattern?.name && <p>Pattern: {p.project.common.pattern.name}</p>}{!!p.project.common.materials?.length && <p>Materials: {p.project.common.materials.map(material => material.text).join(', ')}</p>}{p.project.common.pattern?.url && <SafeLink href={p.project.common.pattern.url}>Pattern website</SafeLink>}</section>}
    {p.event && <section className="preview-context"><h3>Event details</h3><dl><dt>Starts</dt><dd>{date(p.event.startsAt)}</dd><dt>Ends</dt><dd>{date(p.event.endsAt)}</dd>{p.event.timeZone && <><dt>Event time zone</dt><dd>{p.event.timeZone}</dd></>}{p.event.venueName && <><dt>Venue</dt><dd>{p.event.venueName}</dd></>}{p.event.status && <><dt>Status</dt><dd>{label(p.event.status.value)}</dd></>}</dl><div className="button-row">{p.event.eventUri && <SafeLink href={p.event.eventUri}>Event website</SafeLink>}{p.event.registrationUri && <SafeLink href={p.event.registrationUri}>Registration website</SafeLink>}</div></section>}
    {p.external && <aside className="preview-context">{p.external.thumb && <PreviewImageView src={p.external.thumb.url} alt="Link preview image"/>}<strong>{p.external.title}</strong><p className="preserve">{p.external.description}</p><SafeLink href={p.external.uri}>Open linked page</SafeLink></aside>}
    {expanded && <ExpandedImage {...expanded} onClose={() => setExpanded(null)}/>}
    <p className="hint">Preview of the indexed content available for this case.</p>
  </article>
}

function ExpandedImage({ image, index, trigger, onClose }: { image: PreviewImage; index: number; trigger: HTMLElement; onClose: () => void }) {
  const dialog = useRef<HTMLDialogElement>(null)
  const close = useRef<HTMLButtonElement>(null)
  useEffect(() => {
    const element = dialog.current!
    if (typeof element.showModal === 'function') element.showModal()
    else element.setAttribute('open', '')
    close.current?.focus()
    return () => { element.close?.(); trigger.focus() }
  }, [trigger])
  return <dialog ref={dialog} className="preview-image-dialog" aria-label={`Image ${index + 1}`} onCancel={event => { event.preventDefault(); onClose() }} onKeyDown={event => {
    if (event.key === 'Escape') { event.preventDefault(); onClose() }
    if (event.key === 'Tab') { event.preventDefault(); close.current?.focus() }
  }}><div className="card-title"><h2>Image {index + 1}</h2><button ref={close} className="secondary" onClick={onClose}>Close image</button></div><PreviewImageView src={image.fullsize || image.thumb} alt={image.alt || `Image ${index + 1}`}/>{image.alt && <p className="preserve">{image.alt}</p>}</dialog>
}

function SafeLink({ href, children }: { href: string; children: ReactNode }) {
  const destination = safeURL(href)
  return destination ? <a href={destination} target="_blank" rel="noopener noreferrer">{children}</a> : <span>{children} unavailable</span>
}
