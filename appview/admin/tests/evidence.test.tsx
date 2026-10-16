import { expect, test, vi } from 'vitest'
import { act, fireEvent, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SafetyWorkspace } from '../src/workspaces/SafetyWorkspace'
const id='65000000-0000-4000-8000-000000000001',evidenceId='65000000-0000-4000-8000-000000000002'
const admin={did:'did:example:admin',actorId:'operator:fixture',permissionSet:'admin' as const,permissions:{'safety.read':true,'evidence.access':true,'evidence.preserve':true,'hold.manage':true},assignedIncidentReferences:[]}
const reply=(v:unknown,status=200)=>new Response(JSON.stringify(v),{status,headers:{'Content-Type':'application/json'}})
function fixture(override?:(path:string,init?:RequestInit)=>Response|Promise<Response>|undefined){vi.stubGlobal('fetch',vi.fn(async(input:string,init?:RequestInit)=>{const path=String(input),value=override?.(path,init);if(value!==undefined)return value;if(path.includes('/csrf'))return reply({csrfToken:'fixture'});if(path.endsWith('/capabilities'))return reply({preservationAvailable:true,accessAvailable:true,maxPreservationBytes:524288,maxAccessBytes:524288});if(path.includes('?')||/\/(accesses|holds|retention-events|retention-jobs|evidence|releases)$/.test(path))return reply({items:[]});if(path.endsWith('/incidents/'+id))return reply({reference:id,revision:4,kind:'imageMatch',state:'confirmed'});if(path.endsWith('/evidence/'+evidenceId))return reply({id:evidenceId,incidentId:id,revision:0,contentType:'image/png',byteSize:20,held:false,retentionExpiresAt:'2031-01-01T00:00:00Z'});throw new Error('Unexpected evidence fixture endpoint')}))}

test('preservation reviews file metadata without putting image contents into the page and retains incident revision',async()=>{
 window.location.hash='#/incidents/'+id
 const writes:RequestInit[]=[];fixture((path,init)=>{if(init?.method==='POST'){writes.push(init);return reply({id:evidenceId,revision:5})}})
 const user=userEvent.setup();render(<SafetyWorkspace area="incidents" operator={admin} onLocked={()=>{}} onSessionExpired={()=>{}}/>)
 await user.click(await screen.findByText('Preserve restricted evidence'))
 const form=screen.getByRole('form',{name:'Evidence preservation'})
 await user.upload(within(form).getByLabelText('Evidence file'),new File(['BENIGN_CONTENT_CANARY'],'synthetic.png',{type:'image/png'}))
 await user.type(within(form).getByLabelText('Preservation reason'),'Synthetic approved preservation')
 await user.type(within(form).getByLabelText('Retain until'),'2031-01-01T10:00')
 // jsdom's native validator cannot see user-event's FileList shim.
 // Submit the public form event; the application validates file/size/type itself.
 fireEvent.submit(form)
 expect(await screen.findByRole('dialog')).toBeTruthy();expect(document.body.textContent).not.toContain('BENIGN_CONTENT_CANARY');expect(document.body.textContent).not.toContain(btoa('BENIGN_CONTENT_CANARY'));expect(writes).toHaveLength(0)
 await user.click(screen.getByRole('button',{name:'Confirm evidence preservation'}));await screen.findByText('Command committed. Incident data refreshed.')
 expect(JSON.parse(String(writes[0].body))).toMatchObject({expectedRevision:4,contentType:'image/png',bytes:btoa('BENIGN_CONTENT_CANARY'),reason:'Synthetic approved preservation'})
 expect(screen.getByRole('link',{name:'Open recorded item'}).getAttribute('href')).toBe('#/evidence/'+evidenceId)
})

test('binary access is explicit and incident-bound, and hide revokes the in-memory image URL',async()=>{
 window.location.hash='#/evidence/'+evidenceId
 const RealURL=URL,created=vi.fn(()=> 'blob:synthetic-evidence'),revoked=vi.fn()
 vi.stubGlobal('URL',class extends RealURL{static createObjectURL=created;static revokeObjectURL=revoked})
 const writes:RequestInit[]=[];fixture((path,init)=>{if(path.endsWith('/access')&&init?.method==='POST'){writes.push(init);return new Response(new Uint8Array(20),{status:200,headers:{'Content-Type':'application/octet-stream','Cache-Control':'no-store'}})}})
 const user=userEvent.setup();render(<SafetyWorkspace area="evidence" operator={admin} onLocked={()=>{}} onSessionExpired={()=>{}}/>)
 expect(await screen.findByRole('heading',{name:'Restricted evidence access'})).toBeTruthy();expect(writes).toHaveLength(0);expect(screen.queryByRole('img')).toBeNull()
 await user.type(screen.getByLabelText('Access reason'),'Synthetic approved review');await user.click(screen.getByRole('button',{name:'Review evidence access'}));expect(writes).toHaveLength(0)
 await user.click(screen.getByRole('button',{name:'Confirm evidence access'}));await screen.findByRole('img',{name:'Restricted evidence image'})
 expect((screen.getByRole('button',{name:'Review evidence access'}) as HTMLButtonElement).disabled).toBe(true)
 expect(JSON.parse(String(writes[0].body))).toEqual({incidentReference:id,reason:'Synthetic approved review'})
 expect(writes[0]).toMatchObject({credentials:'same-origin',cache:'no-store',redirect:'error'})
 await user.click(screen.getByRole('button',{name:'Hide evidence'}));expect(screen.queryByRole('img')).toBeNull();expect(revoked).toHaveBeenCalledWith('blob:synthetic-evidence')
})

test('an incident hold is limited to reviewed evidence from that incident',async()=>{
 window.location.hash='#/incidents/'+id
 const writes:{path:string;body:unknown}[]=[];fixture((path,init)=>{if(init?.method==='POST'){writes.push({path,body:JSON.parse(String(init.body))});return reply({id:'65000000-0000-4000-8000-000000000003',revision:5})}if(path.startsWith('/v1/admin/safety/evidence?incidentId='+id))return reply({items:[{id:evidenceId,incidentId:id,contentType:'image/png',byteSize:20}]})})
 const user=userEvent.setup();render(<SafetyWorkspace area="incidents" operator={admin} onLocked={()=>{}} onSessionExpired={()=>{}}/>)
 await user.click(await screen.findByText('Create legal hold'))
 const form=screen.getByRole('form',{name:'Legal hold'})
 await user.click(within(form).getByLabelText(evidenceId));await user.type(within(form).getByLabelText('Approved basis'),'Synthetic approved preservation basis');await user.type(within(form).getByLabelText('Hold expires at'),'2031-01-02T10:00')
 await user.click(within(form).getByRole('button',{name:'Review legal hold'}));await user.click(screen.getByRole('button',{name:'Confirm legal hold'}));await screen.findByText('Command committed. Incident data refreshed.')
 expect(writes).toEqual([{path:'/v1/admin/safety/incidents/'+id+'/holds',body:expect.objectContaining({expectedRevision:4,evidenceIds:[evidenceId],basis:'Synthetic approved preservation basis'})}])
})

test('evidence pagination retains its incident filter',async()=>{
 window.location.hash='#/incidents/'+id
 const older='65000000-0000-4000-8000-000000000004'
 fixture(path=>{if(path==='/v1/admin/safety/evidence?incidentId='+id+'&limit=50')return reply({items:[{id:evidenceId}],cursor:'opaque-next'});if(path==='/v1/admin/safety/evidence?incidentId='+id+'&limit=50&cursor=opaque-next')return reply({items:[{id:older}]})})
 const user=userEvent.setup();render(<SafetyWorkspace area="incidents" operator={admin} onLocked={()=>{}} onSessionExpired={()=>{}}/>)
 await screen.findByRole('link',{name:'Review evidence '+evidenceId});await user.click(screen.getByRole('button',{name:'Load earlier records'}))
 expect(await screen.findByRole('link',{name:'Review evidence '+older})).toBeTruthy()
})

test('backgrounding the console clears a displayed image and revokes its URL',async()=>{
 window.location.hash='#/evidence/'+evidenceId
 const RealURL=URL,revoked=vi.fn();vi.stubGlobal('URL',class extends RealURL{static createObjectURL=vi.fn(()=> 'blob:synthetic-background');static revokeObjectURL=revoked})
 fixture((path,init)=>path.endsWith('/access')&&init?.method==='POST'?new Response(new Uint8Array(20),{headers:{'Content-Type':'application/octet-stream','Cache-Control':'no-store'}}):undefined)
 const user=userEvent.setup();render(<SafetyWorkspace area="evidence" operator={admin} onLocked={()=>{}} onSessionExpired={()=>{}}/>)
 await user.type(await screen.findByLabelText('Access reason'),'Synthetic access');await user.click(screen.getByRole('button',{name:'Review evidence access'}));await user.click(screen.getByRole('button',{name:'Confirm evidence access'}));await screen.findByRole('img')
 fireEvent(window,new Event('blur'));expect(screen.queryByRole('img')).toBeNull();expect(revoked).toHaveBeenCalledWith('blob:synthetic-background')
})


test('a late binary response after leaving the workspace cannot affect the new session or display bytes',async()=>{
 window.location.hash='#/evidence/'+evidenceId
 let finish!:(r:Response)=>void
 const pending=new Promise<Response>(resolve=>{finish=resolve}),expired=vi.fn(),created=vi.fn()
 const RealURL=URL;vi.stubGlobal('URL',class extends RealURL{static createObjectURL=created;static revokeObjectURL=vi.fn()})
 fixture((path,init)=>path.endsWith('/access')&&init?.method==='POST'?pending:undefined)
 const user=userEvent.setup(),view=render(<SafetyWorkspace area="evidence" operator={admin} onLocked={()=>{}} onSessionExpired={expired}/>)
 await user.type(await screen.findByLabelText('Access reason'),'Synthetic pending request');await user.click(screen.getByRole('button',{name:'Review evidence access'}));await user.click(screen.getByRole('button',{name:'Confirm evidence access'}));await screen.findByRole('button',{name:'Opening evidence…'})
 view.unmount();await act(async()=>{finish(reply({error:'expired'},401))})
 expect(expired).not.toHaveBeenCalled();expect(created).not.toHaveBeenCalled()
})

test('unconfigured storage explains unavailable preservation and access without fetching bytes',async()=>{
 fixture(path=>path.endsWith('/capabilities')?reply({preservationAvailable:false,accessAvailable:false,maxPreservationBytes:524288,maxAccessBytes:524288}):undefined)
 window.location.hash='#/incidents/'+id
 const user=userEvent.setup(),view=render(<SafetyWorkspace area="incidents" operator={admin} onLocked={()=>{}} onSessionExpired={()=>{}}/>)
 await user.click(await screen.findByText('Preserve restricted evidence'))
 expect(screen.getByText('Evidence preservation is unavailable because restricted storage or the upload allowance is not configured.')).toBeTruthy();expect(screen.queryByLabelText('Evidence file')).toBeNull()
 view.unmount();window.location.hash='#/evidence/'+evidenceId
 render(<SafetyWorkspace area="evidence" operator={admin} onLocked={()=>{}} onSessionExpired={()=>{}}/>)
 expect(await screen.findByText('Image access is unavailable because restricted storage is not configured.')).toBeTruthy();expect(screen.queryByRole('button',{name:'Review evidence access'})).toBeNull()
 expect(vi.mocked(fetch).mock.calls.some(([,init])=>init?.method==='POST')).toBe(false)
})

test('releasing a legal hold reviews its current revision and keeps committed feedback',async()=>{
 const hold='65000000-0000-4000-8000-000000000005';window.location.hash='#/holds/'+hold
 let released=false;const writes:unknown[]=[]
 fixture((path,init)=>{if(init?.method==='POST'){released=true;writes.push(JSON.parse(String(init.body)));return reply({id:hold,revision:9})}if(path.endsWith('/holds/'+hold))return reply({id:hold,incidentId:id,revision:released?9:8,expiresAt:'2031-01-01T00:00:00Z',releasedAt:released?'2030-01-01T00:00:00Z':null})})
 const user=userEvent.setup();render(<SafetyWorkspace area="holds" operator={admin} onLocked={()=>{}} onSessionExpired={()=>{}}/>)
 await user.click(await screen.findByText('Release legal hold'));await user.type(screen.getByLabelText('Release reason'),'Synthetic approval to release')
 await user.click(screen.getByRole('button',{name:'Review hold release'}));expect(writes).toHaveLength(0);await user.click(screen.getByRole('button',{name:'Confirm hold release'}));await screen.findByText('Command committed. Hold data refreshed.')
 expect(writes).toEqual([{expectedRevision:8,reason:'Synthetic approval to release'}]);expect(screen.queryByText('Release legal hold')).toBeNull()
})
