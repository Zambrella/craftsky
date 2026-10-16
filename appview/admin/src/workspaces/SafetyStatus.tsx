import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError, request } from '../api'
import { date, label } from '../cases/model'
type Status={imageScans:{queued:number;leased:number;deadLettered:number;oldestDueSeconds:number;maxAttempts:number;alert:boolean};work:{reference:string;kind:string;state:string;deadline?:string;owner?:string;cover?:string;alert:boolean}[];workTruncated?:boolean}
function duration(seconds:number){
 const total=Math.max(0,Math.floor(seconds))
 if(total<60)return `${total}s`
 const [unit,suffix,nextUnit,nextSuffix]=total>=86400?[86400,'d',3600,'h'] as const:total>=3600?[3600,'h',60,'m'] as const:[60,'m',1,'s'] as const
 const remainder=Math.floor(total%unit/nextUnit)
 return `${Math.floor(total/unit)}${suffix}${remainder?` ${remainder}${nextSuffix}`:''}`
}
export function SafetyStatus({onSessionExpired,locked}:{onSessionExpired:()=>void;locked:boolean}){
 const [data,setData]=useState<Status|null>(null),[busy,setBusy]=useState(false),[error,setError]=useState('');const version=useRef(0)
 const refresh=useCallback(async()=>{const current=++version.current;setBusy(true);setError('');try{const next=await request<Status>('/v1/admin/safety/status');if(current===version.current)setData(next)}catch(err){if(current!==version.current)return;if(err instanceof ApiError&&(err.status===401||err.status===403))onSessionExpired();else setError('Safety overview could not be refreshed. Use the queues to review current work.')}finally{if(current===version.current)setBusy(false)}},[onSessionExpired])
 useEffect(()=>{void refresh();return()=>{version.current++}},[refresh])
 return <details className="card safety-overview"><summary>Safety overview{data?.imageScans.alert?' · Scan queue needs attention':''}{data?.work.some(w=>w.alert)?' · Work needs attention':''}</summary><button className="secondary" disabled={locked||busy} onClick={()=>void refresh()}>{busy&&<span className="spinner"/>}Refresh overview</button>{error&&<p role="alert">{error}</p>}{data&&<><p>{data.imageScans.queued} queued · {data.imageScans.leased} processing · <strong>{data.imageScans.deadLettered} failed scans</strong> · oldest due {duration(data.imageScans.oldestDueSeconds)}</p>{data.work.map(item=><article className="entry" key={item.kind+item.reference}><strong>{label(item.kind)} · {label(item.state)}</strong><p>{item.owner?'Owner '+item.owner:'Unassigned'}{item.cover?' · Cover '+item.cover:''}{item.deadline?' · Deadline '+date(item.deadline):''}</p><a href={'#/'+(['incident','imageMatch'].includes(item.kind)?'incidents':'workflows')+'/'+item.reference} aria-disabled={locked} onClick={e=>{if(locked)e.preventDefault()}}>{item.alert?'Review work needing attention':'Review work'}</a></article>)}{!data.work.length&&<p>No active safety work.</p>}{data.workTruncated&&<p>More work is available in the incident and specialist queues.</p>}</>}</details>
}
