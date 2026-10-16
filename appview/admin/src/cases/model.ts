export const base = '/v1/admin/moderation/cases'
export const effectLabels = {
  formalWarning: 'Formal warning', visibilityWarn: 'Content warning', visibilityHide: 'Hide content',
  visibilityTakedown: 'Take down in CraftSky', strike: 'Account strike', severeSuspension: 'Severe suspension',
} as const
export type Effect = keyof typeof effectLabels
export const reasons = ['harassment', 'hate', 'spam', 'misleading', 'suspected_ai_generated', 'adult_or_graphic', 'impersonation', 'off_topic', 'intellectual_property', 'other', 'child_safety', 'sexual_violation', 'threat_or_violence', 'self_harm', 'stalking_or_privacy', 'fraud_or_scam', 'misleading_commercial']
export function label(value: string) { return value.replace(/([a-z])([A-Z])/g, '$1 $2').replaceAll('_', ' ').replace(/^./, s => s.toUpperCase()) }
export function date(value?: string) { return value ? new Date(value).toLocaleString() : '—' }
export type Page<T> = { items: T[]; cursor?: string }
export type CaseItem = { caseReference: string; state: 'open' | 'resolved'; origin: string; subjectType: string; subjectDid: string; reportCount: number; revision: number; createdAt: string }
export type CaseDetail = CaseItem & {
  ownerDid: string; subjectUri?: string; subjectCidSnapshot?: string; incidentReference?: string; appealStatus?: string; appealRationale?: string;
  reports: { id: string; reporterDid: string; reason: string; details?: string; createdAt: string }[];
  events: { type: string; actorId: string; sourceSystem: string; revision: number; createdAt: string }[];
  decisions: { disposition: string; reason?: string; evidenceNotes?: string; userSafeDetail?: string; severityRationale?: string; legalClassification?: string; createdAt: string }[];
  effects: { effectId: string; type: Effect; action: string; rationale?: string; active: boolean; createdAt: string }[];
  standing: { activeStrikeCount: number; strikeThreshold: number; thresholdSuspended: boolean; severeSuspended: boolean; suspended: boolean };
  strikeContext?: { active: boolean; issuedAt: string; dueAt: string; expiredAt?: string; overturnedAt?: string };
}
export type PreviewImage = { fullsize?: string; thumb?: string; alt: string; aspectRatio?: { width: number; height: number } }
export type BusinessPreview = { cid?: string; tagline?: string; hoursNote?: string; serviceArea?: string; businessTypes?: { value: string; known: boolean }[]; offerings?: { value: string; known: boolean }[]; location?: { country: string; locality?: string }; primaryAction?: { type: string; destination: string }; products?: { title: string; uri?: string; price?: { amount: string; currency: string }; image?: PreviewImage }[] }
export type PreviewVersion = { basis: 'case' | 'current' | 'reference'; status: 'matches' | 'changed' | 'untracked' | 'missing' | 'current'; capturedCid?: string; currentCid?: string; caseOpenedAt?: string; records?: { collection: string; cid: string }[] }
export type Preview = { version?: PreviewVersion; current?: Preview; business?: BusinessPreview; businessState?: string; context?: { kind: 'quote' | 'parent' | 'root'; preview: Preview }[]; state: string; subjectType?: string; subjectDid?: string; title?: string; text?: string; mediaAvailable: boolean; author?: { did: string; displayName?: string | null; avatar?: string }; images?: PreviewImage[]; createdAt?: string; banner?: string; crafts?: string[];
 event?: { startsAt: string; endsAt: string; venueName?: string; timeZone?: string; eventUri?: string; registrationUri?: string; status?: { value: string } };
 video?: { playlist: string; thumbnail?: string; alt: string };
 external?: { uri: string; title: string; description: string; thumb?: { url: string } };
 project?: { common: { title?: string; craftType: string; status?: string; materials?: { text: string }[]; pattern?: { name?: string; url?: string } } };
}
export type Provenance = { id: string; actorId: string; sourceSystem: string; reason: string; createdAt: string }
export type Review = { title: string; endpoint: string; path?: string; expectedRevision?: number | null; body: Record<string, unknown>; summary: string[] }
export function selectedReference() {
  const match = /^#\/cases\/(MOD-[0-9a-f-]{36})$/i.exec(window.location.hash)
  return match?.[1] ?? ''
}
