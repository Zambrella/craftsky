import { expect, test, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AccountWorkspace } from '../src/workspaces/AccountWorkspace'
const op={did:'did:example:admin',actorId:'operator:fixture',permissionSet:'admin' as const,permissions:{'moderation.read':true,'eligibility.review':true},assignedIncidentReferences:[]}
const reply=(v:unknown)=>new Response(JSON.stringify(v),{status:200,headers:{'Content-Type':'application/json'}})
test('eligibility restriction requires evidence and appeal guidance and uses the eligibility revision',async()=>{
 const writes:{path:string;body:unknown}[]=[]
 vi.stubGlobal('fetch',vi.fn(async(path:string,init?:RequestInit)=>{if(init?.method==='POST'){writes.push({path,body:JSON.parse(String(init.body))});return reply({})}if(path.includes('/csrf'))return reply({csrfToken:'fixture'});if(path.includes('/eligibility'))return reply({state:'eligible',revision:4});if(path.includes('/history'))return reply({items:[]});return reply({activeStrikeCount:1,strikeThreshold:3,suspended:false})}))
 const user=userEvent.setup();render(<AccountWorkspace operator={op} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.type(screen.getByLabelText('Account DID'),'did:example:subject');await user.click(screen.getByRole('button',{name:'Review account'}))
 expect(await screen.findByText('1 of 3')).toBeTruthy()
 await user.click(screen.getByText('Restrict age eligibility'))
 const form=screen.getByRole('form',{name:'Eligibility restriction'})
 await user.type(within(form).getByLabelText('Evidence reference'),'verified-record-fixture')
 await user.type(within(form).getByLabelText('Reason'),'Eligibility assessment for local test')
 await user.type(within(form).getByLabelText('Appeal guidance'),'Contact the approved support channel')
 await user.click(within(form).getByRole('button',{name:'Review eligibility restriction'}))
 expect(writes).toHaveLength(0)
 await user.click(screen.getByRole('button',{name:'Confirm eligibility restriction'}))
 await screen.findByText('Command committed. Eligibility data refreshed.')
 expect(writes).toEqual([{path:'/v1/admin/eligibility/reviews',body:expect.objectContaining({accountDid:'did:example:subject',state:'restricted',expectedRevision:4,evidenceKind:'self_disclosure',evidenceReference:'verified-record-fixture',appealGuidance:'Contact the approved support channel'})}])
})
