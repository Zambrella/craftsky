import { expect, test, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SafetyWorkspace } from '../src/workspaces/SafetyWorkspace'

const id='64000000-0000-4000-8000-000000000001'
const operator={did:'did:example:operator',actorId:'operator:fixture',permissionSet:'mod' as const,permissions:{'safety.read':true,'incident.readSafe':true,'incident.confirm':true},assignedIncidentReferences:[]}
const incident={id,reference:id,revision:0,kind:'imageMatch',state:'detected',priority:'urgent',owner:'operator:owner',deadlineAt:'2026-10-15T09:00:00Z',detectedAt:'2026-10-15T08:00:00Z',scannerId:'fixture',policyVersion:'fixture',corpusVersion:'fixture',subjectCounts:{post:1}}
const reply=(v:unknown,status=200)=>new Response(JSON.stringify(v),{status,headers:{'Content-Type':'application/json'}})
function fixture(override?:(path:string,init?:RequestInit)=>Response|Promise<Response>|undefined){vi.stubGlobal('fetch',vi.fn(async(input:string,init?:RequestInit)=>{
 const path=String(input),custom=override?.(path,init);if(custom!==undefined)return custom
 if(path.includes('/auth/csrf'))return reply({csrfToken:'fixture'})
 if(path.endsWith('/status'))return reply({imageScans:{queued:1,deadLettered:0,alert:false},work:[]})
 if(path.includes('/subjects'))return reply({items:[{subjectUri:'at://did:example:owner/social.craftsky.feed.post/one',subjectCid:'fixture',kind:'post',imageSlot:'image:0'}]})
 if(path.includes('/events')||path.includes('/assignments'))return reply({items:[]})
 if(path.includes('/incidents?'))return reply({items:[incident]})
 if(path.endsWith('/incidents/'+id))return reply({...incident,objectKey:'DO-NOT-RENDER'})
 throw new Error('Unexpected fixture endpoint')
}))}

test('moderator opens a safe incident and reviews its owner, deadline and confirmable subject',async()=>{
 fixture();const user=userEvent.setup()
 render(<SafetyWorkspace area="incidents" operator={operator} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.click(await screen.findByRole('link',{name:'Review '+id}))
 expect(await screen.findByText('operator:owner')).toBeTruthy()
 expect(screen.getByRole('heading',{name:'Linked subjects'})).toBeTruthy()
 expect(screen.getAllByText('at://did:example:owner/social.craftsky.feed.post/one').length).toBeGreaterThan(0)
 expect(screen.queryByText('DO-NOT-RENDER')).toBeNull()
 expect(screen.getByRole('button',{name:'Review detection'})).toBeTruthy()
})

test('external intake records metadata without uploading attachments or choosing an actor',async()=>{
 const writes:RequestInit[]=[];fixture((path,init)=>{if(init?.method==='POST'){writes.push(init);return reply({id,revision:0})}if(path.includes('/external-intakes?'))return reply({items:[]})})
 const user=userEvent.setup();render(<SafetyWorkspace area="intakes" operator={{...operator,permissions:{'intake.manage':true}}} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.click(screen.getByText('Record external intake'))
 const form=screen.getByRole('form',{name:'External intake'})
 await user.type(within(form).getByLabelText('Message reference'),'synthetic-message-001')
 await user.type(within(form).getByLabelText('Reported subject'),'at://did:example:owner/social.craftsky.feed.post/one')
 await user.selectOptions(within(form).getByLabelText('Intake kind'),'complaint')
 await user.selectOptions(within(form).getByLabelText('Attachment handling'),'quarantined')
 await user.click(within(form).getByRole('button',{name:'Review external intake'}));await user.click(screen.getByRole('button',{name:'Confirm external intake'}))
 await screen.findByText('Command committed. External intake data refreshed.')
 const body=JSON.parse(String(writes[0].body));expect(body).toMatchObject({kind:'complaint',attachmentStatus:'quarantined',hasMediaAttachment:true,providerMessageReference:'synthetic-message-001'})
 expect(body).not.toHaveProperty('actorId');expect(body).not.toHaveProperty('bytes')
})

test('admin creates a credible-threat review with a consequence-aware confirmation',async()=>{
 const writes:RequestInit[]=[]
 fixture((path,init)=>{if(init?.method==='POST'){writes.push(init);return reply({id,revision:0})}if(path.includes('/workflows?'))return reply({items:[]})})
 const user=userEvent.setup()
 render(<SafetyWorkspace area="workflows" operator={{...operator,permissionSet:'admin',permissions:{...operator.permissions,'workflow.manage':true}}} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.click(screen.getByText('Start credible threat review'))
 const form=screen.getByRole('form',{name:'Credible threat review'})
 await user.type(within(form).getByLabelText('Preservation assessment'),'Relevant records retained in approved storage.')
 await user.click(within(form).getByRole('button',{name:'Review credible threat review'}))
 expect(writes).toHaveLength(0)
 await user.click(screen.getByRole('button',{name:'Confirm credible threat review'}))
 await screen.findByText('Command committed. Workflow data refreshed.')
 expect(JSON.parse(String(writes[0].body))).toMatchObject({preservationCode:'Relevant records retained in approved storage.'})
 expect(JSON.parse(String(writes[0].body))).not.toHaveProperty('expectedRevision')
})

test('helper loads only assigned safe detail and cannot access global queues or confirm detection',async()=>{
 fixture();const user=userEvent.setup()
 render(<SafetyWorkspace area="incidents" operator={{...operator,permissionSet:'helper',permissions:{'incident.readSafe':true},assignedIncidentReferences:[id]}} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.click(screen.getByRole('link',{name:'Review '+id}))
 expect(await screen.findByText('operator:owner')).toBeTruthy()
 expect(screen.queryByRole('button',{name:'Review detection'})).toBeNull()
 expect(vi.mocked(fetch).mock.calls.map(([path])=>String(path))).toEqual(['/v1/admin/safety/incidents/'+id])
})

test('incident confirmation retains the captured subject and reviewed revision',async()=>{
 const writes:RequestInit[]=[]
 fixture((_path,init)=>{if(init?.method==='POST'){writes.push(init);return reply({caseReference:'MOD-'+id,revision:1})}})
 const user=userEvent.setup();render(<SafetyWorkspace area="incidents" operator={operator} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.click(await screen.findByRole('link',{name:'Review '+id}))
 await user.click(await screen.findByRole('button',{name:'Review detection'}))
 expect(writes).toHaveLength(0)
 await user.click(screen.getByRole('button',{name:'Confirm detection'}))
 await screen.findByText('Command committed. Incident data refreshed.')
 expect(JSON.parse(String(writes[0].body))).toEqual({expectedRevision:0,subjectUri:'at://did:example:owner/social.craftsky.feed.post/one'})
})


test('intake correspondence reviews the current intake revision without creating another intake',async()=>{
 window.location.hash='#/intakes/'+id
 const writes:{path:string;body:unknown}[]=[]
 fixture((path,init)=>{if(init?.method==='POST'){writes.push({path,body:JSON.parse(String(init.body))});return reply({id,revision:4})}if(path.includes('/external-intakes?'))return reply({items:[]});if(path.endsWith('/external-intakes/'+id))return reply({id,reference:'CS-SAF-fixture',revision:3,kind:'complaint'});if(path.includes('/correspondence'))return reply({items:[]})})
 const user=userEvent.setup();render(<SafetyWorkspace area="intakes" operator={{...operator,permissions:{'intake.manage':true}}} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.click(await screen.findByText('Add correspondence'))
 const form=screen.getByRole('form',{name:'Correspondence'})
 await user.type(within(form).getByLabelText('Message reference'),'synthetic-follow-up')
 await user.click(within(form).getByRole('button',{name:'Review correspondence'}));await user.click(screen.getByRole('button',{name:'Confirm correspondence'}))
 await screen.findByText('Command committed. Intake data refreshed.')
 expect(writes).toEqual([{path:'/v1/admin/safety/external-intakes/'+id+'/correspondence',body:expect.objectContaining({expectedRevision:3,providerMessageReference:'synthetic-follow-up',hasMediaAttachment:false})}])
})

test('authority disclosure records a reviewed field set and never transmits the data',async()=>{
 window.location.hash='#/workflows/'+id
 const writes:{path:string;body:unknown}[]=[]
 fixture((path,init)=>{if(init?.method==='POST'){writes.push({path,body:JSON.parse(String(init.body))});return reply({revision:3})}if(path.includes('/workflows?'))return reply({items:[]});if(path.endsWith('/workflows/'+id))return reply({id,revision:2,kind:'authorityRequest',state:'inProgress'});})
 const user=userEvent.setup();render(<SafetyWorkspace area="workflows" operator={{...operator,permissionSet:'admin',permissions:{...operator.permissions,'workflow.manage':true,'authority.report':true,'disclosure.approve':true}}} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.click(await screen.findByText('Record approved disclosure'))
 const form=screen.getByRole('form',{name:'Disclosure record'})
 await user.type(within(form).getByLabelText('Legal basis'),'Verified test authority request')
 await user.click(within(form).getByLabelText('Account DID'))
 await user.click(within(form).getByRole('button',{name:'Review disclosure record'}))
 expect(writes).toHaveLength(0)
 expect(screen.getByRole('dialog').textContent).toContain('does not send')
 await user.click(screen.getByRole('button',{name:'Confirm disclosure record'}))
 await screen.findByText('Command committed. Workflow data refreshed.')
 expect(writes).toEqual([{path:'/v1/admin/safety/workflows/authority-requests/'+id+'/disclosures',body:{expectedRevision:2,legalBasis:'Verified test authority request',fields:['accountDid']}}])
})

test('scan retry uses both the job revision and captured scan version',async()=>{
 window.location.hash='#/scans/'+id
 const writes:unknown[]=[]
 fixture((path,init)=>{if(init?.method==='POST'){writes.push(JSON.parse(String(init.body)));return reply({revision:8})}if(path.includes('/scans?'))return reply({items:[]});if(path.endsWith('/scans/'+id))return reply({id,revision:7,version:4,jobState:'dead_letter',state:'error'})})
 const user=userEvent.setup();render(<SafetyWorkspace area="scans" operator={{...operator,permissions:{'safety.read':true,'scan.retry':true}}} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.click(await screen.findByText('Retry failed scan'))
 await user.click(screen.getByRole('button',{name:'Review scan retry'}));await user.click(screen.getByRole('button',{name:'Confirm scan retry'}))
 await screen.findByText('Command committed. Scan data refreshed.')
 expect(writes).toEqual([{expectedRevision:7,expectedVersion:4}])
})

test('administrator changes only a preset permission set with a reviewed operator revision',async()=>{
 window.location.hash='#/operators/'+id
 const writes:{path:string;body:unknown}[]=[]
 fixture((path,init)=>{if(init?.method==='POST'){writes.push({path,body:JSON.parse(String(init.body))});return reply({operatorId:id,revision:6})}if(path.includes('/operators?'))return reply({items:[]});if(path.endsWith('/operators/'+id))return reply({id,did:'did:example:helper',revision:5,permissionSet:'helper',active:true})})
 const user=userEvent.setup();render(<SafetyWorkspace area="operators" operator={{...operator,permissionSet:'admin',permissions:{'operator.manage':true}}} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.click(await screen.findByText('Change permission set'))
 const form=screen.getByRole('form',{name:'Permission change'})
 await user.selectOptions(within(form).getByLabelText('Permission set'),'mod')
 await user.click(within(form).getByRole('button',{name:'Review permission change'}));await user.click(screen.getByRole('button',{name:'Confirm permission change'}))
 await screen.findByText('Command committed. Operator data refreshed.')
 expect(writes).toEqual([{path:'/v1/admin/operators/'+id+'/permission-set',body:{expectedRevision:5,permissionSet:'mod'}}])
})

test('forced operator workspace access sends no private requests for a moderator',()=>{
 fixture();render(<SafetyWorkspace area="operators" operator={operator} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 expect(screen.getByText('Access unavailable')).toBeTruthy();expect(fetch).not.toHaveBeenCalled()
})

test('incident information response uses the request revision rather than the incident revision',async()=>{
 window.location.hash='#/incidents/'+id
 const writes:{path:string;body:unknown}[]=[]
 const requestId='64000000-0000-4000-8000-000000000002'
 fixture((path,init)=>{if(init?.method==='POST'){writes.push({path,body:JSON.parse(String(init.body))});return reply({revision:3})}if(path.includes('/information-requests'))return reply({items:[{id:requestId,revision:2,state:'open'}]});if(path.includes('/reports'))return reply({items:[]})})
 const user=userEvent.setup();render(<SafetyWorkspace area="incidents" operator={{...operator,permissions:{...operator.permissions,'authority.report':true}}} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.click(await screen.findByText('Record response '+requestId))
 await user.click(screen.getByRole('button',{name:'Review information response'}));await user.click(screen.getByRole('button',{name:'Confirm information response'}))
 await screen.findByText('Command committed. Incident data refreshed.')
 expect(writes).toEqual([{path:'/v1/admin/safety/information-requests/'+requestId+'/response',body:{expectedRevision:2}}])
})

test('background identity refresh keeps an in-progress intake draft when capabilities are unchanged',async()=>{
 window.location.hash='#/intakes/'+id
 fixture((path)=>{if(path.includes('/external-intakes?'))return reply({items:[]});if(path.endsWith('/external-intakes/'+id))return reply({id,revision:3,kind:'complaint'});if(path.includes('/correspondence'))return reply({items:[]})})
 const current={...operator,permissions:{'intake.manage':true}}
 const expired=()=>{},locked=()=>{},user=userEvent.setup()
 const view=render(<SafetyWorkspace area="intakes" operator={current} onSessionExpired={expired} onLocked={locked}/>)
 await user.click(await screen.findByText('Add correspondence'))
 await user.type(screen.getByRole('form',{name:'Correspondence'}).querySelector('input')!,'retained-draft')
 const requests=vi.mocked(fetch).mock.calls.length
 view.rerender(<SafetyWorkspace area="intakes" operator={{...current,permissions:{...current.permissions}}} onSessionExpired={expired} onLocked={locked}/>)
 expect(await screen.findByDisplayValue('retained-draft')).toBeTruthy()
 expect(vi.mocked(fetch).mock.calls.length).toBe(requests)
})

test('unconfigured workflow target is a definite rejection and leaves the operator able to continue',async()=>{
 fixture((path,init)=>{if(init?.method==='POST')return reply({error:'deadline_target_not_configured'},422);if(path.includes('/workflows?'))return reply({items:[]})})
 const user=userEvent.setup();render(<SafetyWorkspace area="workflows" operator={{...operator,permissions:{...operator.permissions,'workflow.manage':true}}} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.click(screen.getByText('Start intimate image review'))
 const form=screen.getByRole('form',{name:'Intimate image review'})
 await user.type(within(form).getByLabelText('Standing assessment'),'Synthetic standing');await user.type(within(form).getByLabelText('Declarations assessment'),'Synthetic declaration')
 await user.click(within(form).getByRole('button',{name:'Review intimate image review'}));await user.click(screen.getByRole('button',{name:'Confirm intimate image review'}))
 expect(await screen.findByText('An approved response target must be configured before this workflow can start. Ask a maintainer to configure it.')).toBeTruthy()
 expect(screen.queryByRole('button',{name:'Retry exact command'})).toBeNull()
 expect(within(form).getByRole('button',{name:'Review intimate image review'}).closest('fieldset')?.disabled).toBe(false)
})

test('completing the final workflow action retains committed feedback after the available actions disappear',async()=>{
 window.location.hash='#/workflows/'+id
 let complete=false
 fixture((path,init)=>{if(init?.method==='POST'){complete=true;return reply({revision:1})}if(path.includes('/workflows?'))return reply({items:[]});if(path.endsWith('/workflows/'+id))return reply({id,revision:complete?1:0,kind:'credibleThreat',state:complete?'resolved':'inProgress'})})
 const user=userEvent.setup();render(<SafetyWorkspace area="workflows" operator={{...operator,permissions:{...operator.permissions,'workflow.manage':true}}} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.click(await screen.findByText('Complete threat review'));await user.type(screen.getByLabelText('Outcome assessment'),'Synthetic outcome')
 await user.click(screen.getByRole('button',{name:'Review threat outcome'}));await user.click(screen.getByRole('button',{name:'Confirm threat outcome'}))
 expect(await screen.findByText('Command committed. Workflow data refreshed.')).toBeTruthy()
 expect(screen.queryByText('Complete threat review')).toBeNull()
})

test('verified appeal linking displays the correspondence reference needed for case follow-up',async()=>{
 window.location.hash='#/intakes/'+id
 fixture((path,init)=>{if(init?.method==='POST')return reply({correspondenceId:'64000000-0000-4000-8000-000000000099',revision:1});if(path.includes('/external-intakes?'))return reply({items:[]});if(path.endsWith('/external-intakes/'+id))return reply({id,revision:0,kind:'appeal'});if(path.includes('/correspondence'))return reply({items:[]})})
 const user=userEvent.setup();render(<SafetyWorkspace area="intakes" operator={{...operator,permissions:{'intake.manage':true}}} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.click(await screen.findByText('Link verified appeal'));await user.type(screen.getByLabelText('Case reference'),'MOD-'+id);await user.type(screen.getByLabelText('Verified owner DID'),'did:example:subject')
 await user.click(screen.getByRole('button',{name:'Review appeal link'}));await user.click(screen.getByRole('button',{name:'Confirm appeal link'}))
 expect(await screen.findByText('Recorded reference: 64000000-0000-4000-8000-000000000099')).toBeTruthy()
 expect(screen.getByRole('link',{name:'Open recorded item'}).getAttribute('href')).toBe('#/cases/MOD-'+id)
})
