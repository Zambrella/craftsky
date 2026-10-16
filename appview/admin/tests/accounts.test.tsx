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

test('account lookup displays the current personal and business profile with product media',async()=>{
 vi.stubGlobal('fetch',vi.fn(async(path:string)=>{
  if(path.endsWith('/preview'))return reply({state:'available',subjectType:'account',subjectDid:'did:example:studio',mediaAvailable:true,title:'Mira Studio',text:'Hand woven goods',author:{did:'did:example:studio',displayName:'Mira Studio'},businessState:'available',business:{tagline:'Woven with care',hoursNote:'Weekdays',serviceArea:'UK',businessTypes:[{value:'maker',known:false}],offerings:[],location:{country:'GB',locality:'York'},primaryAction:{type:'visit-website',destination:'https://example.com/studio'},products:[{title:'Woven scarf',uri:'https://example.com/scarf',price:{amount:'25',currency:'GBP'},image:{fullsize:'https://cdn.bsky.app/scarf',alt:'A blue woven scarf'}}]}})
  if(path.includes('/history'))return reply({items:[]})
  if(path.includes('/eligibility'))return reply({state:'eligible',revision:1})
  return reply({activeStrikeCount:0,strikeThreshold:3})
 }))
 const user=userEvent.setup();render(<AccountWorkspace operator={op} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.type(screen.getByLabelText('Account DID'),'did:example:studio');await user.click(screen.getByRole('button',{name:'Review account'}))
 expect(await screen.findByText('Mira Studio')).toBeTruthy()
 expect(screen.getByRole('heading',{name:'Business profile'})).toBeTruthy()
 expect(screen.getByText('Woven with care')).toBeTruthy()
 expect(screen.getByText('25 GBP')).toBeTruthy()
 expect(screen.getByRole('img',{name:'A blue woven scarf'})).toBeTruthy()
 expect(screen.getByRole('link',{name:'Visit website'}).getAttribute('rel')).toContain('noreferrer')
})

test('account cases without a captured version explicitly identify the current profile',async()=>{
 vi.stubGlobal('fetch',vi.fn(async(path:string)=>{
  if(path.endsWith('/preview'))return reply({state:'available',subjectType:'account',subjectDid:'did:example:studio',title:'Current studio profile',mediaAvailable:false,version:{basis:'current',status:'current',currentCid:'profile-cid',records:[{collection:'app.bsky.actor.profile',cid:'profile-cid'},{collection:'social.craftsky.business.profile',cid:'business-cid'}]}})
  if(path.includes('/history'))return reply({items:[]})
  if(path.includes('/eligibility'))return reply({state:'eligible',revision:1})
  return reply({activeStrikeCount:0,strikeThreshold:3})
 }))
 const user=userEvent.setup();render(<AccountWorkspace operator={op} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.type(screen.getByLabelText('Account DID'),'did:example:studio');await user.click(screen.getByRole('button',{name:'Review account'}))
 expect(await screen.findByText('Current indexed content.')).toBeTruthy()
 expect(screen.getByText('business-cid')).toBeTruthy()
})
