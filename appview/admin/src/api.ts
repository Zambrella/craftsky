export type Operator = { did: string; actorId: string; permissionSet: 'admin' | 'mod' | 'helper'; permissions: Record<string, boolean>; assignedIncidentReferences: string[] }
export class ApiError extends Error {
  constructor(public status: number, public code: string, public requestId?: string) { super(code) }
}
export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(path, { ...init, signal: init.signal ?? AbortSignal.timeout(20_000), credentials: 'same-origin', cache: 'no-store', redirect: 'error' })
  if (!response.ok) {
    const body = await response.json().catch(() => ({}))
    throw new ApiError(response.status, typeof body.error === 'string' ? body.error : 'request_failed', typeof body.requestId === 'string' ? body.requestId : undefined)
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}
export const identity = () => request<Operator>('/v1/admin/identity')
export const browserToken = async () => (await request<{ csrfToken: string }>('/v1/admin/auth/csrf')).csrfToken
export const login = (identifier: string, csrf: string) => request<{ authorizationUrl: string }>('/v1/admin/auth/login', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify({ identifier }) })
export const logout = (csrf: string) => request<void>('/v1/admin/auth/logout', { method: 'POST', headers: { 'X-CSRF-Token': csrf } })
