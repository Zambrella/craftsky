import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError, request, type Operator } from '../api'
import { date, label, type Page } from '../cases/model'
import { CommandPanel } from './CommandPanel'
import { areas, eventColumns, referenceFor, textValue, type Column, type Data } from './resources'
import { creationActions, detailActions } from './actions'
import { EvidenceViewer } from './EvidenceViewer'
import { PreserveForm, type EvidenceCapabilities } from './PreserveForm'
import { ActionForm } from './ActionForm'

type Channel={title:string;path:string;columns:Column[];page:Page<Data>}
export function SafetyWorkspace({area,operator,onSessionExpired,onLocked}:{area:string;operator:Operator;onSessionExpired:()=>void;onLocked:(v:boolean)=>void}){
 const definition=areas[area]
 const helper=operator.permissionSet==='helper'
 const canConfirm=!!operator.permissions['incident.confirm'],canAuthority=!!operator.permissions['authority.report'],canManage=!!operator.permissions['operator.manage'],canSafety=!!operator.permissions['safety.read']
 const canPreserve=!!operator.permissions['evidence.preserve'],canAccess=!!operator.permissions['evidence.access'],canHold=!!operator.permissions['hold.manage']
 const [capabilities,setCapabilities]=useState<EvidenceCapabilities|null>(null),[evidenceReading,setEvidenceReading]=useState(false)
 const allowed=!!definition&&(!!operator.permissions[definition.permission]||area==='incidents'&&helper)
 const [selected,setSelected]=useState(()=>referenceFor(area))
 const [page,setPage]=useState<Page<Data>>({items:[]})
 const [state,setState]=useState(''),[kind,setKind]=useState(''),[jobState,setJobState]=useState('')
 const [queueBusy,setQueueBusy]=useState(false),[queueError,setQueueError]=useState('')
 const [detail,setDetail]=useState<Data|null>(null),[channels,setChannels]=useState<Channel[]>([])
 const [busy,setBusy]=useState(false),[error,setError]=useState(''),[locked,setLocked]=useState(false)
 const queueVersion=useRef(0),detailVersion=useRef(0),lock=useRef(false)
 const changeLock=useCallback((v:boolean)=>{lock.current=v;setLocked(v);onLocked(v)},[onLocked])
 const evidenceReadLock=useCallback((v:boolean)=>{setEvidenceReading(v);changeLock(v)},[changeLock])
 const denied=useCallback((err:unknown)=>{if(err instanceof ApiError&&(err.status===401||err.status===403)){onSessionExpired();return true}return false},[onSessionExpired])
 const assigned= !helper||area==='incidents'&&operator.assignedIncidentReferences.includes(selected)
 async function loadQueue(cursor?:string){
  if(!allowed||helper)return
  const version=++queueVersion.current;setQueueBusy(true);setQueueError('')
  const query=new URLSearchParams({limit:'50'});if(state)query.set('state',state);if(kind)query.set('kind',kind);if(jobState)query.set('jobState',jobState);if(cursor)query.set('cursor',cursor)
  try{const next=await request<Page<Data>>(`${definition.path}?${query}`);if(version===queueVersion.current)setPage(prior=>({...next,items:cursor?[...prior.items,...next.items]:next.items}));return true}
  catch(err){if(version===queueVersion.current&&!denied(err))setQueueError('The queue could not be loaded. Refresh to try again.');return false}
  finally{if(version===queueVersion.current)setQueueBusy(false)}
 }
 const loadDetail=useCallback(async()=>{
  if(!allowed||!selected||!assigned)return
  const version=++detailVersion.current;setBusy(true);setError('')
  const path=`${definition.path}/${selected}`
  const reads:Omit<Channel,'page'>[]=[]
  if(area==='incidents'){
   if(canAccess)reads.push({title:'Restricted evidence',path:'/v1/admin/safety/evidence?incidentId='+selected,columns:areas.evidence.columns})
   if(canHold)reads.push({title:'Legal holds',path:'/v1/admin/safety/holds?incidentId='+selected,columns:areas.holds.columns})
   if(canConfirm){reads.push({title:'Linked subjects',path:path+'/subjects',columns:[['subjectUri','Subject'],['subjectCid','Captured version'],['kind','Kind'],['imageSlot','Image slot']]},{title:'Assignments',path:path+'/assignments',columns:[['operatorId','Operator'],['role','Role'],['assignedAt','Assigned','date'],['releasedAt','Released','date']]})}
   if(canManage)reads.push({title:'Available operators',path:'/v1/admin/operators?state=active',columns:[]})
   if(canAuthority)reads.push({title:'Authority report records',path:path+'/reports',columns:[['id','Report reference'],['kind','Kind'],['submittedBy','Recorded by'],['submittedAt','Recorded','date'],['deadlineMet','Within deadline'],['retentionUntil','Retain until','date']]},{title:'Information requests',path:path+'/information-requests',columns:[['id','Request reference'],['state','State'],['dueAt','Due','date'],['receivedAt','Received','date'],['respondedAt','Responded','date']]})
  }
  if(area==='incidents'&&canSafety||area==='workflows'||area==='operators')reads.push({title:'Audit timeline',path:path+'/events',columns:eventColumns})
  if(area==='evidence')reads.push({title:'Access attempts',path:path+'/accesses',columns:[['actorId','Operator'],['action','Action'],['reason','Reason'],['allowed','Authorized'],['createdAt','Recorded','date']]},{title:'Legal holds',path:path+'/holds',columns:areas.holds.columns},{title:'Retention events',path:path+'/retention-events',columns:[['outcome','Outcome'],['occurredAt','Recorded','date']]},{title:'Retention processing',path:path+'/retention-jobs',columns:[['state','State'],['attemptCount','Attempts'],['maxAttempts','Attempt limit'],['nextAttemptAt','Next attempt','date'],['errorCategory','Failure category'],['updatedAt','Updated','date']]})
  if(area==='holds')reads.push({title:'Restricted evidence',path:path+'/evidence',columns:areas.evidence.columns},{title:'Hold release',path:path+'/releases',columns:[['actorId','Operator'],['reason','Reason'],['releasedAt','Released','date']]})
  if(area==='intakes')reads.push({title:'Correspondence',path:path+'/correspondence',columns:[['attachmentStatus','Attachment status'],['receivedAt','Received','date'],['createdAt','Recorded','date']]})
  try{
   const capabilityPromise=(canPreserve||canAccess)?request<EvidenceCapabilities>('/v1/admin/safety/evidence/capabilities'):Promise.resolve(null)
   const [capability,results]=await Promise.all([capabilityPromise,Promise.all([request<Data>(path),...reads.map(read=>request<Page<Data>>(read.path+(read.path.includes('?')?'&':'?')+'limit=50'))])])
   const [record,...extras]=results
   if(version===detailVersion.current){setCapabilities(capability);setDetail(record as Data);setChannels(reads.map((read,i)=>({...read,page:extras[i] as Page<Data>})))}
  }catch(err){if(version===detailVersion.current&&!denied(err))setError('Resource data could not be refreshed. Refresh before taking another action.');throw err}
  finally{if(version===detailVersion.current)setBusy(false)}
 },[allowed,assigned,area,selected,definition,canConfirm,canAuthority,canManage,canSafety,canPreserve,canAccess,canHold,denied])
 useEffect(()=>{const change=()=>{if(lock.current){window.history.replaceState(null,'',`#/${area}/${selected}`);return}setSelected(referenceFor(area))};window.addEventListener('hashchange',change);return()=>window.removeEventListener('hashchange',change)},[area,selected])
 useEffect(()=>{setPage({items:[]});void loadQueue();return()=>{queueVersion.current++}},[allowed,area,state,kind,jobState])
 useEffect(()=>{setDetail(null);setChannels([]);setError('');if(selected&&assigned)void loadDetail().catch(()=>{});return()=>{detailVersion.current++}},[loadDetail,selected,assigned])
 if(!allowed)return <section className="card"><h2>Access unavailable</h2><p>Your permission set does not include this workspace.</p></section>
 async function moreChannel(channel:Channel){const version=detailVersion.current;const next=await request<Page<Data>>(channel.path+(channel.path.includes('?')?'&':'?')+'limit=50&cursor='+encodeURIComponent(channel.page.cursor||''));if(version===detailVersion.current)setChannels(prior=>prior.map(c=>c.path===channel.path?{...c,page:{...next,items:[...c.page.items,...next.items]}}:c))}
 const items:Data[]=helper?operator.assignedIncidentReferences.map(id=>({id})):page.items
 const creations=creationActions(area,operator)
 const commandAccess=area==='incidents'&&(canConfirm||canAuthority||canManage||canPreserve||canHold)||area==='intakes'&&operator.permissions['intake.manage']||area==='workflows'&&operator.permissions['workflow.manage']||area==='holds'&&canHold||area==='operators'&&canManage||area==='scans'&&operator.permissions['scan.retry']
 const actions=detail?detailActions(area,selected,detail,operator,Object.fromEntries(channels.map(c=>[c.title,c.page.items]))):[]
 return <>{!!creations.length&&<div className="create-actions"><CommandPanel key={'create-'+area} basePath={definition.path} reference={'New '+definition.title.toLowerCase()+' record'} title="Start a review" resourceLabel={area==='workflows'?'Workflow':definition.title} disabled={locked||queueBusy||!!queueError} refresh={async()=>{if(!await loadQueue())throw new Error('refresh unavailable')}} onLocked={changeLock} onSessionExpired={onSessionExpired}>{(review,epoch)=>creations.map(action=><ActionForm key={action.path+epoch} action={action} review={review}/>)}</CommandPanel></div>}<div className="cases-workspace">
  <aside className="queue card" aria-label={definition.title+' queue'}><div className="card-title"><h2>{helper?'Assigned incidents':definition.title}</h2>{!helper&&<button className="secondary" disabled={queueBusy||locked} onClick={()=>void loadQueue()}>{queueBusy&&<span className="spinner"/>}Refresh queue</button>}</div>
   {!helper&&<fieldset className="filters action-fields" disabled={locked}>{definition.states&&<label>Status<select value={state} onChange={e=>setState(e.target.value)}><option value="">All statuses</option>{definition.states.map(s=><option key={s} value={s}>{label(s)}</option>)}</select></label>}{definition.kinds&&<label>Kind<select value={kind} onChange={e=>setKind(e.target.value)}><option value="">All kinds</option>{definition.kinds.map(s=><option key={s} value={s}>{label(s)}</option>)}</select></label>}{area==='scans'&&<label>Job status<select value={jobState} onChange={e=>setJobState(e.target.value)}><option value="">All jobs</option>{['queued','leased','dead_letter'].map(s=><option key={s} value={s}>{label(s)}</option>)}</select></label>}</fieldset>}
   {queueError&&<p role="alert">{queueError}</p>}{queueBusy&&<p role="status">Loading queue…</p>}{!queueBusy&&!items.length&&<p className="empty">{helper?'No incidents are assigned to you.':'No records match these filters.'}</p>}
   <ol className="case-list">{items.map(item=><li key={item.id}><a href={`#/${area}/${item.id}`} aria-label={'Review '+item.id} aria-current={selected===item.id?'page':undefined} aria-disabled={locked} onClick={e=>{if(lock.current)e.preventDefault();else setSelected(String(item.id))}}><strong>{textValue(item.did||item.reference||item.id)}</strong>{!helper&&<><span>{label(textValue(item.kind||item.permissionSet||item.state||item.operation))}</span>{typeof item.deadlineAt==='string'&&<small className={Date.parse(item.deadlineAt)<Date.now()?'overdue':''}>Deadline {date(item.deadlineAt)}</small>}</>}</a></li>)}</ol>
   {page.cursor&&<button className="secondary" disabled={queueBusy||locked} onClick={()=>void loadQueue(page.cursor)}>Load more records</button>}
  </aside>
  <section className="case-pane" aria-label="Resource review">
   {!selected?<section className="card empty"><h2>{definition.title}</h2><p>Choose a record to review its current state and available actions.</p></section>:!assigned?<section className="card"><h2>Assignment required</h2><p>This incident is not assigned to you.</p></section>:<>
    <div className="case-toolbar"><span className="hint">{helper?'Safe metadata for your assigned incident.':'Review current information before confirming an action.'}</span><button className="secondary" disabled={busy||locked} onClick={()=>void loadDetail().catch(()=>{})}>{busy&&<span className="spinner"/>}Refresh record</button></div>
    {error&&<p className="notice" role="alert">{error}</p>}{busy&&<p role="status">Loading record…</p>}
    {detail&&<><div className="case-record"><section className="card"><h2 className="case-reference">{selected}</h2>{typeof detail.revision==='number'&&<p className="hint">Revision {detail.revision}</p>}<Metadata record={detail} columns={definition.columns}/></section>{area==='evidence'&&<EvidenceViewer key={selected} record={{...detail,id:selected}} capabilities={capabilities} disabled={busy||!!error} onLocked={changeLock} onSessionExpired={onSessionExpired}/>} {channels.filter(c=>c.title!=='Available operators').map(channel=><ReadChannel key={selected+channel.path} channel={channel} locked={locked||busy} denied={denied} more={()=>moreChannel(channel)}/> )}</div>
     {commandAccess&&<CommandPanel basePath={`${definition.path}/${selected}`} reference={selected} revision={detail.revision} title="Available actions" resourceLabel={area==='holds'?'Hold':area==='intakes'?'Intake':area==='workflows'?'Workflow':area==='operators'?'Operator':area==='scans'?'Scan':'Incident'} disabled={busy||!!error||evidenceReading} refresh={async()=>{await loadDetail();void loadQueue()}} onSessionExpired={onSessionExpired} onLocked={changeLock}>{(review,epoch)=><>{!actions.length&&area!=='incidents'&&<p>No further actions are available for this record’s current state.</p>}{area==='incidents'&&canPreserve&&<PreserveForm key={'preserve-'+epoch} capabilities={capabilities} review={review} onReading={evidenceReadLock}/>} {actions.map(action=><ActionForm key={action.path+epoch} action={action} review={review}/>)}{channels.find(c=>c.title==='Linked subjects')?.page.items.map(subject=><div key={textValue(subject.subjectUri)} className="case-form"><p className="did">{textValue(subject.subjectUri)}</p><button className="primary" onClick={()=>review({title:'Detection',endpoint:'confirmations',body:{subjectUri:subject.subjectUri},summary:['Confirm the detection for this captured subject and create its moderation case.',textValue(subject.subjectUri),'This does not itself impose a strike or content decision.']})}>Review detection</button></div>)}</>}</CommandPanel>}
    </>}
   </>}
  </section>
 </div></>
}

export function Metadata({record,columns}:{record:Data;columns:Column[]}){return <dl>{columns.map(([key,title,type])=>record[key]!==undefined&&record[key]!==null&&record[key]!==''?<div className="metadata-pair" key={key}><dt>{title}</dt><dd className="preserve">{type==='date'?date(String(record[key])):textValue(record[key])}</dd></div>:null)}</dl>}
function ReadChannel({channel,locked,denied,more:loadMore}:{channel:Channel;locked:boolean;denied:(err:unknown)=>boolean;more:()=>Promise<void>}){
 const page=channel.page
 const [busy,setBusy]=useState(false),[error,setError]=useState('')
 async function more(){if(!page.cursor||busy)return;setBusy(true);setError('');try{await loadMore()}catch(err){if(!denied(err))setError('Earlier records could not be loaded. Try again.')}finally{setBusy(false)}}
 return <section className="card"><h2>{channel.title}</h2>{!page.items.length&&<p>No records yet.</p>}{page.items.map((item,i)=><article className="entry" key={String(item.id||item.subjectUri||i)}><Metadata record={item} columns={channel.columns}/>{['Restricted evidence','Legal holds'].includes(channel.title)&&item.id&&<a href={'#/'+(channel.title==='Legal holds'?'holds':'evidence')+'/'+encodeURIComponent(item.id)} aria-disabled={locked} onClick={e=>{if(locked)e.preventDefault()}}>Review {channel.title==='Legal holds'?'hold':'evidence'} {item.id}</a>}</article>)}{error&&<p role="alert">{error}</p>}{page.cursor&&<button className="secondary" disabled={locked||busy} onClick={()=>void more()}>{busy&&<span className="spinner"/>}Load earlier records</button>}</section>
}
