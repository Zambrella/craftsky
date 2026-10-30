import { useCallback, useEffect, useRef, useState } from 'react'
import { request, type Operator } from '../api'
import { CasesWorkspace } from '../cases/CasesWorkspace'
import { AccountWorkspace } from './AccountWorkspace'
import { SafetyStatus } from './SafetyStatus'
import { SafetyWorkspace } from './SafetyWorkspace'
import { areas } from './resources'
import { ActionForm } from './ActionForm'
import { caseAction } from './actions'
import { CommandPanel } from './CommandPanel'
const descriptions:Record<string,string>={
 cases:'Review reports and moderation cases, record decisions, and manage appeals and consequences.',
 accounts:'Look up account standing, moderation history and eligibility decisions.',
 evidence:'Review preserved evidence and its access history. Opening an image requires a separate, audited confirmation.',
 holds:'Track holds that preserve evidence, review their coverage, and release them when approved.',
 incidents:'Review detected safety issues, assign operators, track deadlines and reporting, and follow incidents through resolution.',
 intakes:'Record reports, complaints and appeals received outside the app, track correspondence, and link them to cases or specialist reviews.',
 workflows:'Manage intimate-image, credible-threat and authority-request reviews, including assessments, approvals and outcomes.',
 scans:'Review image scan results and processing failures, and retry failed scans after checking their current state.',
 operators:'Manage approved Atmosphere accounts, set their permissions, revoke sessions and disable access.',
 operations:'Review recorded operator commands, their outcomes and linked records.',
 'new-case':'Open a case for a post, account, event or profile link that needs review. Creating a case does not apply consequences.',
}
function currentArea(operator:Operator){return window.location.hash.split('/')[1]||(operator.permissions['moderation.read']?'cases':'incidents')}
export function OperatorWorkspace({operator,onSessionExpired,onLocked}:{operator:Operator;onSessionExpired:()=>void;onLocked:(v:boolean)=>void}){
 const [area,setArea]=useState(()=>currentArea(operator)),[locked,setLocked]=useState(false)
 const lock=useRef(false),stableHash=useRef(window.location.hash)
 const changeLock=useCallback((v:boolean)=>{lock.current=v;setLocked(v);onLocked(v)},[onLocked])
 useEffect(()=>{const change=()=>{if(lock.current){window.history.replaceState(null,'',stableHash.current||'#/cases');return}stableHash.current=window.location.hash;setArea(currentArea(operator))};window.addEventListener('hashchange',change);return()=>window.removeEventListener('hashchange',change)},[operator])
 const links=[...(operator.permissions['moderation.read']?[['cases','Cases'],['accounts','Accounts']]:operator.permissions['eligibility.review']?[['accounts','Accounts']]:[]),...Object.entries(areas).filter(([key,definition])=>operator.permissions[definition.permission]||key==='incidents'&&operator.permissionSet==='helper').map(([key,definition])=>[key,definition.title]),...(operator.permissions['moderation.manage']?[['new-case','Create case']]:[])]
 const description=links.some(([key])=>key===area)?area==='incidents'&&operator.permissionSet==='helper'?'Review safe details for incidents assigned to you.':descriptions[area]:undefined
 return <>{operator.permissions['safety.read']&&<SafetyStatus onSessionExpired={onSessionExpired} locked={locked}/>}
 <nav className="workspace-nav" aria-label="Moderation workspaces">{links.map(([key,title])=><a key={key} href={'#/'+key} aria-current={area===key?'page':undefined} aria-disabled={locked} onClick={e=>{if(lock.current)e.preventDefault()}}>{title}</a>)}</nav>
 {description&&<p className="workspace-description">{description}</p>}
 {area==='cases'?<CasesWorkspace operator={operator} onSessionExpired={onSessionExpired} onLocked={changeLock}/>:area==='accounts'?<AccountWorkspace operator={operator} onSessionExpired={onSessionExpired} onLocked={changeLock}/>:area==='new-case'&&operator.permissions['moderation.manage']?<CommandPanel basePath="/v1/admin/moderation/cases" reference="New moderation case" resourceLabel="Case" title="Open a case" disabled={false} refresh={async result=>{if(!result?.caseReference)throw new Error('Recorded case reference unavailable');await request('/v1/admin/moderation/cases/'+result.caseReference)}} onLocked={changeLock} onSessionExpired={onSessionExpired}>{(review,epoch)=><ActionForm key={epoch} action={caseAction()} review={review}/>}</CommandPanel>:areas[area]?<SafetyWorkspace key={area} area={area} operator={operator} onSessionExpired={onSessionExpired} onLocked={changeLock}/>:<section className="card"><h2>Access unavailable</h2><p>Choose an available workspace.</p></section>}
 </>
}
