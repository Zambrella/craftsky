import { expect, test, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { OperatorWorkspace } from '../src/workspaces/OperatorWorkspace'
const op={did:'did:example:mod',actorId:'operator:fixture',permissionSet:'mod' as const,permissions:{'moderation.read':true,'moderation.manage':true,'safety.read':true,'intake.manage':true,'incident.confirm':true},assignedIncidentReferences:[]}
test('navigation exposes permitted workspaces and blocks navigation while a new case is under review',async()=>{
 vi.stubGlobal('fetch',vi.fn(async(path:string)=>new Response(JSON.stringify(path.endsWith('/status')?{imageScans:{queued:0,leased:0,deadLettered:0},work:[]}:{items:[]}),{status:200,headers:{'Content-Type':'application/json'}})))
 const user=userEvent.setup();render(<OperatorWorkspace operator={op} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 expect(screen.getByRole('link',{name:'External intake'})).toBeTruthy();expect(screen.queryByRole('link',{name:'Operators'})).toBeNull()
 const tabs=screen.getByRole('navigation',{name:'Moderation workspaces'})
 const overview=screen.getByText('Safety overview').closest('details')!
 expect(overview.compareDocumentPosition(tabs)&Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
 expect(tabs.nextElementSibling?.textContent).toBe('Review reports and moderation cases, record decisions, and manage appeals and consequences.')
 await user.click(screen.getByRole('link',{name:'Create case'}));await user.click(await screen.findByText('Create moderation case'))
 expect(tabs.nextElementSibling?.textContent).toContain('Creating a case does not apply consequences.')
 await user.selectOptions(screen.getByLabelText('Subject type'),'account');await user.type(screen.getByLabelText('Subject account DID'),'did:example:subject');await user.type(screen.getByLabelText('Reason for opening the case'),'Local review fixture')
 await user.click(screen.getByRole('button',{name:'Review moderation case'}))
 const before=window.location.hash
 await user.click(screen.getByRole('link',{name:'External intake'}))
 expect(window.location.hash).toBe(before);expect(screen.getByRole('dialog')).toBeTruthy()
 await user.click(screen.getByRole('button',{name:'Back to review'}))
 await user.click(screen.getByRole('link',{name:'External intake'}))
 expect(await screen.findByRole('complementary',{name:'External intake queue'})).toBeTruthy()
 expect(tabs.nextElementSibling?.textContent).toContain('Record reports, complaints and appeals received outside the app')
})

test('new case confirmation reads back the recorded case before reporting refreshed data',async()=>{
 window.location.hash='#/new-case'
 const reference='MOD-64000000-0000-4000-8000-000000000001'
 vi.stubGlobal('fetch',vi.fn(async(path:string,init?:RequestInit)=>new Response(JSON.stringify(path.includes('/csrf')?{csrfToken:'fixture'}:init?.method==='POST'?{caseReference:reference,revision:0}:path.endsWith('/status')?{imageScans:{queued:0,leased:0,deadLettered:0},work:[]}:{items:[]}),{status:200,headers:{'Content-Type':'application/json'}})))
 const user=userEvent.setup();render(<OperatorWorkspace operator={op} onSessionExpired={()=>{}} onLocked={()=>{}}/>)
 await user.click(screen.getByText('Create moderation case'));await user.selectOptions(screen.getByLabelText('Subject type'),'account');await user.type(screen.getByLabelText('Subject account DID'),'did:example:subject');await user.type(screen.getByLabelText('Reason for opening the case'),'Synthetic case')
 await user.click(screen.getByRole('button',{name:'Review moderation case'}));await user.click(screen.getByRole('button',{name:'Confirm moderation case'}))
 await screen.findByText('Command committed. Case data refreshed.')
 expect(vi.mocked(fetch).mock.calls.some(([path])=>String(path)==='/v1/admin/moderation/cases/'+reference)).toBe(true)
})
