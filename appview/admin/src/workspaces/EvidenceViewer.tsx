import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError, browserToken } from '../api'
import { Confirmation } from '../cases/Confirmation'
import type { Review } from '../cases/model'
import type { Data } from './resources'
import type { EvidenceCapabilities } from './PreserveForm'

export function EvidenceViewer({record,capabilities,disabled,onLocked,onSessionExpired}:{record:Data;capabilities:EvidenceCapabilities|null;disabled:boolean;onLocked:(v:boolean)=>void;onSessionExpired:()=>void}){
 const [reason,setReason]=useState(''),[review,setReview]=useState<Review|null>(null),[running,setRunning]=useState(false),[url,setUrl]=useState(''),[notice,setNotice]=useState('')
 const activeURL=useRef(''),generation=useRef(0),controller=useRef<AbortController|null>(null),inFlight=useRef(false),returnFocus=useRef<HTMLElement|null>(null)
 const clear=useCallback(()=>{generation.current++;controller.current?.abort();controller.current=null;if(activeURL.current)URL.revokeObjectURL(activeURL.current);activeURL.current='';setUrl('');setReview(null);setRunning(false);inFlight.current=false},[])
 const hide=useCallback(()=>{clear();setNotice('Evidence hidden.')},[clear])
 useEffect(()=>{onLocked(!!review||running);return()=>onLocked(false)},[review,running,onLocked])
 useEffect(()=>{const hidden=()=>{if(document.visibilityState==='hidden')hide()};window.addEventListener('blur',hide);window.addEventListener('pagehide',hide);document.addEventListener('visibilitychange',hidden);return()=>{window.removeEventListener('blur',hide);window.removeEventListener('pagehide',hide);document.removeEventListener('visibilitychange',hidden);generation.current++;controller.current?.abort();if(activeURL.current)URL.revokeObjectURL(activeURL.current);activeURL.current=''}},[hide])
 useEffect(()=>{if(disabled)clear()},[disabled,clear])
 async function access(){
  if(inFlight.current||!review)return
  const approved=review;setReview(null);setRunning(true);setNotice('');inFlight.current=true
  const version=++generation.current,next=new AbortController();controller.current=next;const timeout=setTimeout(()=>next.abort(),30000)
  try{
   const csrf=await browserToken();if(version!==generation.current)return
   const response=await fetch('/v1/admin/safety/evidence/'+record.id+'/access',{method:'POST',credentials:'same-origin',cache:'no-store',redirect:'error',signal:next.signal,headers:{'Content-Type':'application/json','X-CSRF-Token':csrf},body:JSON.stringify(approved.body)})
   if(version!==generation.current){await response.body?.cancel().catch(()=>{});return}
   if(!response.ok){if(response.status===401||response.status===403){onSessionExpired();return}throw new Error('Evidence access unavailable')}
   if(response.headers.get('Content-Type')?.split(';')[0]!=='application/octet-stream'||!response.headers.get('Cache-Control')?.split(',').some(v=>v.trim()==='no-store')||!response.body)throw new Error('Invalid evidence transport')
   const reader=response.body.getReader(),chunks:Uint8Array[]=[];let size=0
   try{for(;;){const chunk=await reader.read();if(chunk.done)break;size+=chunk.value.length;if(size>(capabilities?.maxAccessBytes??0))throw new Error('Evidence exceeds display limit');chunks.push(chunk.value)}}finally{await reader.cancel().catch(()=>{})}
   if(!size||version!==generation.current||document.visibilityState==='hidden')return
   const bytes=new Uint8Array(size);let offset=0;for(const chunk of chunks){bytes.set(chunk,offset);offset+=chunk.length}
   const object=URL.createObjectURL(new Blob([bytes],{type:String(record.contentType)}));activeURL.current=object;setUrl(object);setNotice('Access recorded. Hide the image when your review is finished.')
  }catch(error){if(version===generation.current){if(error instanceof ApiError&&(error.status===401||error.status===403))onSessionExpired();else setNotice('Evidence could not be displayed. The access attempt may have been recorded. Refresh metadata before reviewing again.')}}
  finally{clearTimeout(timeout);if(version===generation.current){controller.current=null;inFlight.current=false;setRunning(false)}}
 }
 const available=capabilities?.accessAvailable&&!record.deletedAt&&['image/png','image/jpeg','image/gif','image/webp'].includes(String(record.contentType))&&Number(record.byteSize)>0&&Number(record.byteSize)<=capabilities.maxAccessBytes
 return <section className="card evidence-viewer"><h2>Restricted evidence access</h2>{!capabilities?.accessAvailable?<p>Image access is unavailable because restricted storage is not configured.</p>:record.deletedAt?<p>This evidence was deleted under retention handling. Its metadata and audit remain available.</p>:!available?<p>This item cannot be displayed within the console’s supported image and size limits.</p>:<form onSubmit={e=>{e.preventDefault();if(disabled||running||!reason.trim())return;returnFocus.current=document.activeElement as HTMLElement;setReview({title:'Evidence access',endpoint:'access',body:{incidentReference:record.incidentId,reason:reason.trim()},summary:['Open this image through audited restricted access. It is cleared when hidden, when you leave the workspace, or when this browser loses focus.',`Incident: ${record.incidentId}`,`Evidence: ${record.id}`,`Access reason: ${reason.trim()}`]})}}><label>Access reason<textarea aria-label="Access reason" required maxLength={512} disabled={disabled||running||!!review||!!url} value={reason} onChange={e=>setReason(e.target.value)}/></label><button className="primary" disabled={disabled||running||!!review||!!url||!reason.trim()}>{running&&<span className="spinner"/>}{running?'Opening evidence…':'Review evidence access'}</button></form>}
 {notice&&<p role="status" className="notice">{notice}</p>}{url&&<><button className="secondary" onClick={hide}>Hide evidence</button><img src={url} alt="Restricted evidence image" draggable={false} referrerPolicy="no-referrer" onError={()=>{hide();setNotice('The image could not be rendered. Access was recorded.')}}/></>}
 {review&&<Confirmation review={review} reference={String(record.id)} returnFocus={returnFocus.current} onCancel={()=>setReview(null)} onConfirm={()=>void access()}/>}</section>
}
