import { useEffect, useRef, useState, type FormEvent } from 'react'
import type { Review } from '../cases/model'
export type EvidenceCapabilities={preservationAvailable:boolean;accessAvailable:boolean;maxPreservationBytes:number;maxAccessBytes:number}
export function PreserveForm({capabilities,review,onReading}:{capabilities:EvidenceCapabilities|null;review:(r:Review)=>void;onReading:(v:boolean)=>void}){
 const [file,setFile]=useState<File|null>(null),[reason,setReason]=useState(''),[expires,setExpires]=useState(''),[reading,setReading]=useState(false),[error,setError]=useState('')
 const reader=useRef<FileReader|null>(null),alive=useRef(true)
 useEffect(()=>{alive.current=true;return()=>{alive.current=false;reader.current?.abort()}},[])
 async function submit(event:FormEvent){
  event.preventDefault();if(reading)return;setError('')
  const maximum=capabilities?.maxPreservationBytes??0
  if(!capabilities?.preservationAvailable||!file||file.size===0||file.size>maximum||!['image/png','image/jpeg','image/gif','image/webp'].includes(file.type)||!reason.trim()||!expires||Date.parse(expires)<=Date.now()){setError('Choose a supported image within the size limit, a preservation reason and a future retention deadline.');return}
  setReading(true);onReading(true)
  try{
   const buffer=await new Promise<ArrayBuffer>((resolve,reject)=>{const next=new FileReader();reader.current=next;next.onload=()=>resolve(next.result as ArrayBuffer);next.onerror=next.onabort=()=>reject(new Error('File unavailable'));next.readAsArrayBuffer(file)})
   if(!alive.current)return
   const bytes=new Uint8Array(buffer);let binary='';for(let offset=0;offset<bytes.length;offset+=32768)binary+=String.fromCharCode(...bytes.subarray(offset,offset+32768))
   review({title:'Evidence preservation',endpoint:'evidence',body:{bytes:btoa(binary),contentType:file.type,reason:reason.trim(),expiresAt:new Date(expires).toISOString()},summary:['Preserve the selected original image in restricted storage. This does not publish or forward it.',`File: ${file.name} · ${file.size} bytes · ${file.type}`,`Preservation reason: ${reason.trim()}`,`Retain until: ${new Date(expires).toLocaleString()}`]})
  }catch{if(alive.current)setError('The selected file could not be read. Choose it again.')}
  finally{if(alive.current){setReading(false);onReading(false)}}
 }
 return <details className="action-disclosure"><summary>Preserve restricted evidence</summary>{!capabilities?.preservationAvailable?<p>Evidence preservation is unavailable because restricted storage or the upload allowance is not configured.</p>:<form className="case-form" aria-label="Evidence preservation" onSubmit={submit}><p>Use only an image already obtained through the approved evidence procedure. The file stays in memory until submission; no preview is loaded automatically.</p><label>Evidence file<input aria-label="Evidence file" type="file" accept="image/png,image/jpeg,image/gif,image/webp" required disabled={reading} onChange={e=>setFile(e.target.files?.[0]??null)}/></label><p className="hint">Maximum {Math.floor(capabilities.maxPreservationBytes/1024)} KiB. JPEG, PNG, GIF or WebP.</p><label>Preservation reason<textarea aria-label="Preservation reason" required maxLength={512} value={reason} onChange={e=>setReason(e.target.value)}/></label><label>Retain until<input aria-label="Retain until" required type="datetime-local" value={expires} onChange={e=>setExpires(e.target.value)}/></label>{error&&<p role="alert">{error}</p>}<button className="primary" disabled={reading}>{reading&&<span className="spinner"/>}{reading?'Preparing review…':'Review evidence preservation'}</button></form>}</details>
}
