import { expect, test, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { SafetyStatus } from '../src/workspaces/SafetyStatus'
test('safety overview exposes queue alerts and links overdue work without rendering unknown payload fields',async()=>{
 vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify({imageScans:{queued:3,leased:1,deadLettered:2,oldestDueSeconds:69735,maxAttempts:4,alert:true},work:[{reference:'64000000-0000-4000-8000-000000000001',kind:'credibleThreat',state:'open',priority:2,ageSeconds:600,deadline:'2026-10-15T09:00:00Z',owner:'operator:fixture',alert:true,objectKey:'DO-NOT-RENDER'}]}),{status:200})))
 render(<SafetyStatus onSessionExpired={()=>{}} locked={false}/>)
 expect(await screen.findByText('2 failed scans')).toBeTruthy()
 expect(screen.getByText(/oldest due/).textContent).toContain('oldest due 19h 22m')
 expect(screen.getByRole('link',{name:'Review work needing attention'}).getAttribute('href')).toBe('#/workflows/64000000-0000-4000-8000-000000000001')
 expect(screen.queryByText('DO-NOT-RENDER')).toBeNull()
})
