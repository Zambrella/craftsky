import { useEffect, useRef, useState, type ReactNode } from 'react'
import { PreviewVideo } from './PreviewVideo'
import { PreviewImageView, safeURL } from './PreviewMedia'
import { date, label, type Preview, type PreviewImage, type PreviewVersion } from './model'

export function ContentPreview({ preview }: { preview: Preview | null }) {
  return <section className="card content-preview"><h2>Content preview</h2>{preview?.version && <VersionNotice version={preview.version}/>}{preview?.state === 'available' ? <SubjectPreview key={`${preview.subjectDid}:${preview.createdAt}:${preview.title}`} preview={preview}/> : <UnavailablePreview preview={preview}/>} {preview?.current && <section className="preview-current"><h3>Current indexed content</h3><p className="hint">This is the current record, shown separately from the case reference.</p>{preview.current.version && <VersionNotice version={preview.current.version}/>} {preview.current.state === 'available' ? <SubjectPreview preview={preview.current}/> : <UnavailablePreview preview={preview.current}/>}</section>}</section>
}

function UnavailablePreview({ preview }: { preview: Preview | null }) {
  if (preview?.state === 'restricted') return <p>Restricted content. Use the specialist review workflow.</p>
  if (preview?.version?.status === 'changed' || preview?.version?.status === 'missing') return <p>The content for the captured version was not retained.</p>
  return <p>Content is unavailable. It may have changed or may not have a safe captured version.</p>
}
function VersionNotice({ version: v }: { version: PreviewVersion }) {
  const message = v.basis === 'current' ? 'Current indexed content.' : v.status === 'untracked' ? 'No record version was captured for this case. Showing the current indexed content.' : v.status === 'changed' ? v.basis === 'case' ? 'Changed since the case was opened.' : 'The referenced post has changed.' : v.status === 'missing' ? 'The referenced record is no longer indexed.' : v.basis === 'case' ? v.records?.length ? 'Personal profile matches the captured case reference. Other profile records show their current indexed versions.' : 'Indexed content matches the captured case reference.' : 'Matches the version referenced by this post.'
  return <div className="preview-version"><p className={v.status === 'changed' || v.status === 'untracked' ? 'notice' : 'hint'}>{message}</p><details><summary>Record versions</summary><dl>{v.caseOpenedAt && <><dt>Case opened</dt><dd>{date(v.caseOpenedAt)}</dd></>}{v.capturedCid && <><dt>Captured record CID</dt><dd className="did">{v.capturedCid}</dd></>}{v.currentCid && <><dt>Current record CID</dt><dd className="did">{v.currentCid}</dd></>}{v.records?.map(record => <div className="preview-version-record" key={record.collection}><dt>Current {record.collection}</dt><dd className="did">{record.cid}</dd></div>)}</dl></details></div>
}

function SubjectPreview({ preview: p, includeContext = true }: { preview: Preview; includeContext?: boolean }) {
  const [expanded, setExpanded] = useState<{ image: PreviewImage; index: number; trigger: HTMLElement } | null>(null)
  return <article className="subject-preview">
    {p.banner && <PreviewImageView src={p.banner} alt="Account banner" className="preview-banner"/>}
    <div className="preview-author">{p.author?.avatar && <PreviewImageView src={p.author.avatar} alt="Account avatar" className="preview-avatar"/>}<div><span className="badge">{label(p.subjectType || 'Content')}</span><strong>{p.author?.displayName || p.title || p.author?.did || p.subjectDid || 'Content'}</strong><span className="did">{p.author?.did || p.subjectDid}</span>{p.createdAt && <small>{date(p.createdAt)}</small>}</div></div>
    {p.subjectType === 'post' && p.author && <p className="hint">Author details reflect the current indexed profile.</p>}
    {!!p.crafts?.length && <div className="preview-tags">{p.crafts.map(craft => <span className="badge" key={craft}>{label(craft)}</span>)}</div>}
    <p className="preserve">{p.text || 'No text accompanies this subject.'}</p>
    {!!p.images?.length && <div className="preview-images">{p.images.map((image, index) => <figure key={image.fullsize || index}><button className="preview-image-button" aria-label={`Enlarge image ${index + 1}`} onClick={event => setExpanded({ image, index, trigger: event.currentTarget })}><PreviewImageView src={image.fullsize || image.thumb} alt={image.alt || `Image ${index + 1}`}/></button>{image.alt && <figcaption className="preserve">{image.alt}</figcaption>}</figure>)}</div>}
    {p.business && <section className="preview-context"><h3>Business profile</h3><p className="hint">Current indexed business record.</p>{p.business.tagline && <p className="preserve">{p.business.tagline}</p>}<div className="preview-tags">{[...(p.business.businessTypes || []), ...(p.business.offerings || [])].map(value => <span className="badge" key={value.value}>{label(value.value.replaceAll('-', ' '))}</span>)}</div><dl>{p.business.location && <><dt>Location</dt><dd>{[p.business.location.locality, p.business.location.country].filter(Boolean).join(', ')}</dd></>}{p.business.serviceArea && <><dt>Service area</dt><dd>{p.business.serviceArea}</dd></>}{p.business.hoursNote && <><dt>Hours</dt><dd className="preserve">{p.business.hoursNote}</dd></>}</dl>{p.business.primaryAction && <p><SafeLink href={p.business.primaryAction.destination} allowMailto={p.business.primaryAction.type === 'email'}>{label(p.business.primaryAction.type.replaceAll('-', ' '))}</SafeLink></p>}<div className="preview-products">{p.business.products?.map((product, index) => <section className="preview-product" key={index}><h4>{product.title}</h4>{product.image && <button className="preview-image-button" aria-label={`Enlarge product image ${index + 1}`} onClick={event => setExpanded({ image: product.image!, index, trigger: event.currentTarget })}><PreviewImageView src={product.image.fullsize || product.image.thumb} alt={product.image.alt || product.title}/></button>}{product.price && <p>{product.price.amount} {product.price.currency}</p>}{product.uri && <SafeLink href={product.uri}>View product</SafeLink>}</section>)}</div></section>}
    {p.businessState === 'unavailable' && <p className="notice">Business profile unavailable for safe preview.</p>}
    {p.video && <PreviewVideo key={p.video.playlist} video={p.video}/>}
    {p.project && <section className="preview-context"><h3>{p.project.common.title || 'Project details'}</h3><p>{label(p.project.common.craftType.split('#').at(-1) || 'Craft')}{p.project.common.status && ` · ${label(p.project.common.status.split('#').at(-1) || '')}`}</p>{p.project.common.pattern?.name && <p>Pattern: {p.project.common.pattern.name}</p>}{!!p.project.common.materials?.length && <p>Materials: {p.project.common.materials.map(material => material.text).join(', ')}</p>}{p.project.common.pattern?.url && <SafeLink href={p.project.common.pattern.url}>Pattern website</SafeLink>}</section>}
    {p.event && <section className="preview-context"><h3>Event details</h3><dl><dt>Starts</dt><dd>{date(p.event.startsAt)}</dd><dt>Ends</dt><dd>{date(p.event.endsAt)}</dd>{p.event.timeZone && <><dt>Event time zone</dt><dd>{p.event.timeZone}</dd></>}{p.event.venueName && <><dt>Venue</dt><dd>{p.event.venueName}</dd></>}{p.event.status && <><dt>Status</dt><dd>{label(p.event.status.value)}</dd></>}</dl><div className="button-row">{p.event.eventUri && <SafeLink href={p.event.eventUri}>Event website</SafeLink>}{p.event.registrationUri && <SafeLink href={p.event.registrationUri}>Registration website</SafeLink>}</div></section>}
    {p.external && <aside className="preview-context">{p.external.thumb && <PreviewImageView src={p.external.thumb.url} alt="Link preview image"/>}<strong>{p.external.title}</strong><p className="preserve">{p.external.description}</p><SafeLink href={p.external.uri}>Open linked page</SafeLink></aside>}
    {includeContext && !!p.context?.length && <div className="preview-context-list">{p.context.slice(0, 3).map(item => <section className="preview-context" key={item.kind}><h3>{{quote: 'Quoted post', parent: 'Reply parent', root: 'Thread root'}[item.kind]}</h3>{item.preview.version && <VersionNotice version={item.preview.version}/>} {item.preview.state === 'available' ? <SubjectPreview preview={item.preview} includeContext={false}/> : <p>{item.preview.state === 'restricted' ? 'Restricted context. Use the specialist review workflow.' : 'This referenced version is unavailable. It may have changed or been removed.'}</p>}</section>)}</div>}
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

function SafeLink({ href, children, allowMailto = false }: { href: string; children: ReactNode; allowMailto?: boolean }) {
  const destination = allowMailto && href.startsWith('mailto:') ? href : safeURL(href)
  return destination ? <a href={destination} target="_blank" rel="noopener noreferrer">{children}</a> : <span>{children} unavailable</span>
}
