import {
	APITransport,
	pageQuery,
	profileSegment,
	segment,
	type Fetcher,
	type TransportOptions
} from './transport';
import type * as T from './types';
export * from './types';
export * from './transport';

const postPath = (did: string, rkey: string) =>
	`/v1/posts/${segment(did)}/${segment(rkey)}`;
const profilePath = (id: string) => `/v1/profiles/${profileSegment(id)}`;
const conversationPath = (id: string) =>
	`/v1/chat/conversations/${segment(id)}`;
const stringFields =
	(...names: string[]) =>
	(value: Record<string, unknown>) =>
		names.every(
			(name) => typeof value[name] === 'string' && value[name] !== ''
		);
const acceptedPost = (value: Record<string, unknown>) =>
	stringFields('uri', 'cid', 'rkey')(value) &&
	typeof value.text === 'string' &&
	!!value.author &&
	typeof value.author === 'object' &&
	stringFields('did', 'handle')(value.author as Record<string, unknown>);
const acceptedEdit = (value: Record<string, unknown>) =>
	stringFields('acceptedCid')(value) &&
	typeof value.changed === 'boolean' &&
	(!value.changed || typeof value.editedAt === 'string');

export interface CraftskyAPIOptions extends TransportOptions {
	onRelationshipChanged?: () => void;
}

/** Feature-specific methods mirror the Flutter clients and AppView camelCase wire contracts. */
export class CraftskyAPI {
	readonly transport: APITransport;
	constructor(
		fetcher: Fetcher = fetch,
		basePath = '/api/appview',
		private readonly options: CraftskyAPIOptions = {}
	) {
		this.transport = new APITransport(fetcher, basePath, options);
	}
	private async relationshipMutation<T>(request: Promise<T>): Promise<T> {
		const result = await request;
		this.options.onRelationshipChanged?.();
		return result;
	}
	readonly account = {
		whoami: () => this.transport.get<T.WhoAmI>('/v1/whoami'),
		logout: () =>
			this.transport.request<void>('POST', '/v1/auth/logout', {
				response: 'empty'
			}),
		eligibility: () =>
			this.transport.get<T.EligibilityStatus>('/v1/account/eligibility'),
		onboarding: () =>
			this.transport.get<T.OnboardingStatus>('/v1/onboarding/status'),
		completeOnboarding: (meetsMinimumAge: boolean, policyVersion: string) =>
			this.transport.request<T.OnboardingStatus>(
				'POST',
				'/v1/onboarding/completion',
				{ body: { meetsMinimumAge, policyVersion } }
			),
		deletionIntent: () =>
			this.transport.request<T.AccountDeletionIntent>(
				'POST',
				'/v1/account-deletion/intents'
			),
		cancelDeletion: (jobId: string) =>
			this.transport.request<void>(
				'DELETE',
				`/v1/account-deletion/intents/${segment(jobId)}`,
				{ response: 'empty' }
			),
		acceptDeletion: (
			jobId: string,
			reauthProof: string,
			confirmationDid: string
		) =>
			this.transport.request<void>(
				'POST',
				`/v1/account-deletions/${segment(jobId)}`,
				{ body: { reauthProof, confirmationDid }, response: 'empty' }
			)
	};
	readonly posts = {
		timeline: (options: T.PageQuery = {}) =>
			this.transport.page<T.TimelineItem>(
				'/v1/feed/timeline',
				pageQuery(options),
				options.signal
			),
		extendedNetwork: (options: T.PageQuery = {}) =>
			this.transport.page<T.TimelineItem>(
				'/v1/feed/extended-network',
				pageQuery(options),
				options.signal
			),
		get: (did: string, rkey: string) =>
			this.transport.get<T.PostPresentation>(postPath(did, rkey)),
		editSeed: (did: string, rkey: string) =>
			this.transport.get<T.PostEditSeed>(`${postPath(did, rkey)}/edit`),
		create: (
			payload: T.PostPayload,
			operationKey: string,
			signal?: AbortSignal
		) =>
			this.transport.request<T.Post>('POST', '/v1/posts', {
				body: payload,
				operationKey,
				signal,
				validate: acceptedPost
			}),
		edit: (
			did: string,
			rkey: string,
			changes: T.PostEditChanges,
			expectedCid: string,
			operationKey: string,
			signal?: AbortSignal
		) =>
			this.transport.request<T.PostEditResult>('PATCH', postPath(did, rkey), {
				body: { changes },
				expectedCid,
				operationKey,
				signal,
				validate: acceptedEdit
			}),
		delete: (
			did: string,
			rkey: string,
			expectedCid: string,
			operationKey: string
		) =>
			this.transport.request<void>('DELETE', postPath(did, rkey), {
				expectedCid,
				operationKey,
				response: 'empty'
			}),
		like: (did: string, rkey: string, operationKey: string) =>
			this.transport.request<T.InteractionWrite>(
				'POST',
				`${postPath(did, rkey)}/likes`,
				{ operationKey, validate: stringFields('uri', 'cid') }
			),
		unlike: (did: string, rkey: string, operationKey: string) =>
			this.transport.request<void>('DELETE', `${postPath(did, rkey)}/likes`, {
				operationKey,
				response: 'empty'
			}),
		repost: (did: string, rkey: string, operationKey: string) =>
			this.transport.request<T.InteractionWrite>(
				'POST',
				`${postPath(did, rkey)}/reposts`,
				{ operationKey, validate: stringFields('uri', 'cid') }
			),
		unrepost: (did: string, rkey: string, operationKey: string) =>
			this.transport.request<void>('DELETE', `${postPath(did, rkey)}/reposts`, {
				operationKey,
				response: 'empty'
			}),
		likes: (did: string, rkey: string, options: T.PageQuery = {}) =>
			this.transport.get<T.ProfilePage>(
				`${postPath(did, rkey)}/likes`,
				pageQuery(options),
				options.signal
			),
		reposts: (did: string, rkey: string, options: T.PageQuery = {}) =>
			this.transport.get<T.ProfilePage>(
				`${postPath(did, rkey)}/reposts`,
				pageQuery(options),
				options.signal
			),
		quotes: (did: string, rkey: string, options: T.PageQuery = {}) =>
			this.transport.page<T.PostPresentation>(
				`${postPath(did, rkey)}/quotes`,
				pageQuery(options),
				options.signal
			),
		comments: (
			did: string,
			rkey: string,
			options: T.PageQuery & {
				sort?: 'oldest' | 'newest' | 'follows';
				focus?: string;
			} = {}
		) =>
			this.transport.get<T.CommentSection>(
				`${postPath(did, rkey)}/comments`,
				{ ...pageQuery(options), sort: options.sort, focus: options.focus },
				options.signal
			),
		replies: (did: string, rkey: string, options: T.PageQuery = {}) =>
			this.transport.get<T.ReplyPage>(
				`${postPath(did, rkey)}/replies`,
				pageQuery(options),
				options.signal
			),
		report: (did: string, rkey: string, submission: T.ReportSubmission) =>
			this.transport.request<T.ReportResult>(
				'POST',
				`${postPath(did, rkey)}/reports`,
				{ body: submission }
			),
		pins: () => this.transport.get<T.ProfilePins>('/v1/profiles/me/pins'),
		pin: (did: string, rkey: string) =>
			this.transport.request<T.ProfilePins>(
				'PUT',
				`${postPath(did, rkey)}/pin`
			),
		unpin: (did: string, rkey: string) =>
			this.transport.request<T.ProfilePins>(
				'DELETE',
				`${postPath(did, rkey)}/pin`
			)
	};
	readonly profiles = {
		pins: () => this.transport.get<T.ProfilePins>('/v1/profiles/me/pins'),
		me: () => this.transport.get<T.Profile>('/v1/profiles/me'),
		get: (id: string) => this.transport.get<T.Profile>(profilePath(id)),
		recent: () => this.transport.page<T.Author>('/v1/profiles/recent'),
		update: (profile: T.ProfileUpdate, operationKey: string) =>
			this.transport.request<T.Profile>('PUT', '/v1/profiles/me', {
				body: profile,
				operationKey
			}),
		customisation: (value: T.ProfileCustomisation) =>
			this.transport.request<T.ProfileCustomisation>(
				'PUT',
				'/v1/profiles/me/customisation',
				{ body: value }
			),
		posts: (
			id: string,
			tab: 'posts' | 'projects' | 'comments' | 'reposts' = 'posts',
			options: T.PageQuery = {}
		) =>
			this.transport.page<T.PostPresentation>(
				`${profilePath(id)}/${tab}`,
				pageQuery(options),
				options.signal
			),
		follow: (id: string, operationKey: string) =>
			this.relationshipMutation(
				this.transport.request<T.Profile>(
					'POST',
					`${profilePath(id)}/follows`,
					{
						operationKey
					}
				)
			),
		unfollow: (id: string, operationKey: string) =>
			this.relationshipMutation(
				this.transport.request<T.Profile>(
					'DELETE',
					`${profilePath(id)}/follows`,
					{ operationKey }
				)
			),
		mute: (id: string) =>
			this.relationshipMutation(
				this.transport.request<T.Relationship>(
					'POST',
					`${profilePath(id)}/mutes`
				)
			),
		unmute: (id: string) =>
			this.relationshipMutation(
				this.transport.request<T.Relationship>(
					'DELETE',
					`${profilePath(id)}/mutes`
				)
			),
		block: (id: string, operationKey: string) =>
			this.relationshipMutation(
				this.transport.request<T.Relationship>(
					'POST',
					`${profilePath(id)}/blocks`,
					{ operationKey }
				)
			),
		unblock: (id: string, operationKey: string) =>
			this.relationshipMutation(
				this.transport.request<void>('DELETE', `${profilePath(id)}/blocks`, {
					operationKey,
					response: 'empty'
				})
			),
		relationships: (
			kind: 'followers' | 'following' | 'mutes' | 'blocks',
			options: T.PageQuery = {}
		) =>
			this.transport.get<T.ProfilePage>(
				`/v1/profiles/me/${kind}`,
				pageQuery(options),
				options.signal
			),
		mutualFollowers: (id: string, options: T.PageQuery = {}) =>
			this.transport.get<T.ProfilePage>(
				`${profilePath(id)}/mutual-followers`,
				pageQuery(options),
				options.signal
			),
		report: (id: string, submission: T.ReportSubmission) =>
			this.transport.request<T.ReportResult>(
				'POST',
				`${profilePath(id)}/reports`,
				{ body: submission }
			),
		reportLink: (did: string, rkey: string, submission: T.ReportSubmission) =>
			this.transport.request<T.ReportResult>(
				'POST',
				`/v1/profile-links/${segment(did)}/${segment(rkey)}/reports`,
				{ body: submission }
			),
		followerGrowth: (period: '7d' | '30d' | '1y') =>
			this.transport.get<T.FollowerGrowth>('/v1/profiles/me/follower-growth', {
				period
			}),
		linkPreview: (url: string, signal?: AbortSignal) =>
			this.transport.request<{
				reachability: 'reachable' | 'unreachable' | 'unknown';
				icon?: {
					bytes: string;
					mimeType: string;
					width: number;
					height: number;
				};
			}>('POST', '/v1/profile-link-previews', { body: { url }, signal })
	};
	readonly search = {
		suggestions: (
			q: string,
			options: {
				types?: ('profiles' | 'hashtags')[];
				profileLimit?: number;
				hashtagLimit?: number;
				signal?: AbortSignal;
			} = {}
		) =>
			this.transport.get<T.SearchSuggestions>(
				'/v1/search/suggestions',
				{
					q,
					types: options.types?.join(','),
					profileLimit: options.profileLimit,
					hashtagLimit: options.hashtagLimit
				},
				options.signal
			),
		profiles: (q: string, options: T.PageQuery = {}) =>
			this.transport.page<T.Author>(
				'/v1/search/profiles',
				{ q, ...pageQuery(options) },
				options.signal
			),
		posts: (options: T.SearchQuery) =>
			this.transport.page<T.PostPresentation>(
				'/v1/search/posts',
				{
					...options.filters,
					q: options.q,
					sort: options.sort,
					...pageQuery(options)
				},
				options.signal
			),
		projects: (options: T.PageQuery & { q: string }) =>
			this.transport.page<T.PostPresentation>(
				'/v1/search/projects',
				{
					q: options.q,
					...pageQuery(options)
				},
				options.signal
			),
		browseProjects: (options: T.ProjectQuery = {}) =>
			this.transport.page<T.PostPresentation>(
				'/v1/projects',
				{
					...options.filters,
					craftType: options.craftType,
					sort: options.sort,
					...pageQuery(options)
				},
				options.signal
			),
		hashtags: (q: string, options: T.PageQuery = {}) =>
			this.transport.page<T.Hashtag>(
				'/v1/search/hashtags',
				{ q, ...pageQuery(options) },
				options.signal
			),
		hashtagPosts: (
			tag: string,
			options: T.PageQuery & { sort?: T.SearchSort } = {}
		) =>
			this.transport.page<T.PostPresentation>(
				`/v1/search/hashtags/${segment(tag)}/posts`,
				{ sort: options.sort, ...pageQuery(options) },
				options.signal
			),
		topHashtags: (craftTypes?: string[], limit?: number) =>
			this.transport.get<T.TopHashtags>('/v1/search/hashtags/top', {
				craftTypes,
				limit
			}),
		recent: () => this.transport.page<T.RecentSearch>('/v1/search/recent'),
		saveRecent: (request: T.SaveRecentSearch) =>
			this.transport.request<T.RecentSearch>('POST', '/v1/search/recent', {
				body: request
			}),
		deleteRecent: (id: string) =>
			this.transport.request<void>(
				'DELETE',
				`/v1/search/recent/${segment(id)}`,
				{ response: 'empty' }
			),
		mentionSuggestions: (q: string, limit = 10, signal?: AbortSignal) =>
			this.transport.page<T.Author>(
				'/v1/facets/mentions',
				{ q, limit },
				signal
			),
		resolveMention: (handle: string, signal?: AbortSignal) =>
			this.transport.get<{ did: string }>(
				'/v1/facets/mentions/resolve',
				{ handle },
				signal
			),
		hashtagSuggestions: (q: string, limit = 10, signal?: AbortSignal) =>
			this.transport.page<{ tag: string; postCount: number }>(
				'/v1/facets/hashtags',
				{ q, limit },
				signal
			)
	};
	readonly saved = {
		list: (options: T.SavedQuery = {}) =>
			this.transport.page<T.SavedItem>(
				'/v1/saved-posts',
				{
					...pageQuery(options),
					sort: options.sort,
					folderId: options.folderId,
					unfiled: options.unfiled || undefined
				},
				options.signal
			),
		save: (did: string, rkey: string, folderId?: string | null) =>
			this.transport.request<T.SavedState>(
				'POST',
				`${postPath(did, rkey)}/saves`,
				{ body: folderId ? { folderId } : {} }
			),
		unsave: (did: string, rkey: string) =>
			this.transport.request<void>('DELETE', `${postPath(did, rkey)}/saves`, {
				response: 'empty'
			}),
		folders: (options: T.PageQuery = {}) =>
			this.transport.page<T.SavedFolder>(
				'/v1/saved-post-folders',
				pageQuery(options),
				options.signal
			),
		createFolder: (name: string) =>
			this.transport.request<T.SavedFolder>('POST', '/v1/saved-post-folders', {
				body: { name }
			}),
		renameFolder: (id: string, name: string) =>
			this.transport.request<T.SavedFolder>(
				'PATCH',
				`/v1/saved-post-folders/${segment(id)}`,
				{ body: { name } }
			),
		deleteFolder: (id: string, deleteSaves = false) =>
			this.transport.request<void>(
				'DELETE',
				`/v1/saved-post-folders/${segment(id)}`,
				{
					query: deleteSaves ? { deleteSaves: true } : undefined,
					response: 'empty'
				}
			)
	};
	readonly schedules = {
		list: () =>
			this.transport.page<T.ScheduledPostSummary>('/v1/scheduled-posts'),
		get: (id: string) =>
			this.transport.get<T.ScheduledPost>(`/v1/scheduled-posts/${segment(id)}`),
		create: (
			operationId: string,
			scheduledAt: string,
			payload: T.JsonObject,
			signal?: AbortSignal
		) =>
			this.transport.request<T.ScheduledPost>('POST', '/v1/scheduled-posts', {
				signal,
				body: { operationId, scheduledAt, payload }
			}),
		update: (
			id: string,
			scheduledAt: string,
			payload: T.JsonObject,
			signal?: AbortSignal
		) =>
			this.transport.request<T.ScheduledPost>(
				'PUT',
				`/v1/scheduled-posts/${segment(id)}`,
				{ signal, body: { scheduledAt, payload } }
			),
		delete: (id: string) =>
			this.transport.request<void>(
				'DELETE',
				`/v1/scheduled-posts/${segment(id)}`,
				{ response: 'empty' }
			),
		publish: (id: string, payload: T.JsonObject, signal?: AbortSignal) =>
			this.transport.request<{ status: 'published' | 'publishing' }>(
				'POST',
				`/v1/scheduled-posts/${segment(id)}/publication`,
				{ signal, body: { payload } }
			),
		stageMedia: (
			id: string,
			bytes: Blob | Uint8Array,
			mime: string,
			signal?: AbortSignal
		) =>
			this.transport.request<T.JsonObject>(
				'PUT',
				`/v1/scheduled-post-media/${segment(id)}`,
				{ body: bytes, headers: { 'Content-Type': mime }, signal }
			),
		media: (id: string, signal?: AbortSignal) =>
			this.transport.request<ArrayBuffer>(
				'GET',
				`/v1/scheduled-post-media/${segment(id)}`,
				{ response: 'bytes', signal }
			),
		deleteMedia: (id: string) =>
			this.transport.request<void>(
				'DELETE',
				`/v1/scheduled-post-media/${segment(id)}`,
				{ response: 'empty' }
			)
	};
	readonly notifications = {
		list: (options: T.PageQuery & { category?: T.NotificationCategory } = {}) =>
			this.transport.page<T.AppNotification>(
				'/v1/notifications',
				{ ...pageQuery(options), category: options.category },
				options.signal
			),
		count: () =>
			this.transport.get<{ newCount: number }>('/v1/notifications/new-count'),
		seen: () =>
			this.transport.request<void>('POST', '/v1/notifications/seen', {
				response: 'empty'
			}),
		preferences: () =>
			this.transport.get<T.NotificationPreferences>(
				'/v1/notifications/preferences'
			),
		patchPreferences: (patch: T.NotificationPreferencePatch) =>
			this.transport.request<T.NotificationPreferences>(
				'PATCH',
				'/v1/notifications/preferences',
				{ body: patch }
			),
		registerDevice: (platform: 'android' | 'ios' | 'web', token: string) =>
			this.transport.request<{ accountSubscriptionId: string }>(
				'POST',
				'/v1/notifications/devices',
				{ body: { platform, token } }
			),
		unregisterDevice: (id: string) =>
			this.transport.request<void>(
				'DELETE',
				`/v1/notifications/devices/${segment(id)}`,
				{ response: 'empty' }
			)
	};
	readonly languages = {
		get: () =>
			this.transport.get<T.LanguagePreferences>('/v1/languages/preferences'),
		replace: (preferences: T.LanguagePreferences) =>
			this.transport.request<T.LanguagePreferences>(
				'PUT',
				'/v1/languages/preferences',
				{ body: preferences }
			),
		initialize: (preferences: T.LanguagePreferences) =>
			this.transport.request<T.LanguagePreferences>(
				'POST',
				'/v1/languages/preferences/initialize',
				{ body: preferences }
			)
	};
	readonly business = {
		saveProfile: (
			body: T.JsonObject,
			expectedCid: string | null,
			operationKey: string
		) =>
			this.transport.request<T.RecordMutation>(
				'PUT',
				'/v1/profiles/me/business',
				{ body, expectedCid: expectedCid ?? '*', operationKey }
			),
		deleteProfile: (expectedCid: string, operationKey: string) =>
			this.transport.request<void>('DELETE', '/v1/profiles/me/business', {
				expectedCid,
				operationKey,
				response: 'empty'
			}),
		profileEvents: (id: string, options: T.PageQuery = {}) =>
			this.transport.page<T.BusinessEvent>(
				`/v1/profiles/${segment(id)}/events`,
				pageQuery(options),
				options.signal
			),
		ownerEvents: (
			filter: 'upcoming' | 'history' = 'upcoming',
			options: T.PageQuery = {}
		) =>
			this.transport.page<T.BusinessEvent>(
				'/v1/events',
				{ filter, ...pageQuery(options) },
				options.signal
			),
		event: (did: string, rkey: string) =>
			this.transport.get<T.BusinessEvent>(
				`/v1/events/${segment(did)}/${segment(rkey)}`
			),
		createEvent: (body: T.JsonObject, operationKey: string) =>
			this.transport.request<T.RecordMutation>('POST', '/v1/events', {
				body,
				operationKey
			}),
		updateEvent: (
			did: string,
			rkey: string,
			body: T.JsonObject,
			expectedCid: string,
			operationKey: string
		) =>
			this.transport.request<T.RecordMutation>(
				'PUT',
				`/v1/events/${segment(did)}/${segment(rkey)}`,
				{ body, expectedCid, operationKey }
			),
		deleteEvent: (
			did: string,
			rkey: string,
			expectedCid: string,
			operationKey: string
		) =>
			this.transport.request<void>(
				'DELETE',
				`/v1/events/${segment(did)}/${segment(rkey)}`,
				{ expectedCid, operationKey, response: 'empty' }
			),
		reportEvent: (did: string, rkey: string, submission: T.ReportSubmission) =>
			this.transport.request<T.ReportResult>(
				'POST',
				`/v1/events/${segment(did)}/${segment(rkey)}/reports`,
				{ body: submission }
			)
	};
	readonly billing = {
		access: () =>
			this.transport.get<T.SubscriptionAccess>('/v1/subscriptions/access'),
		get: () => this.transport.get<T.BillingState>('/v1/billing/account'),
		ensure: () =>
			this.transport.request<T.BillingState>('PUT', '/v1/billing/account'),
		reconcile: () =>
			this.transport.request<T.JsonObject>(
				'POST',
				'/v1/billing/reconciliation'
			),
		assign: (licenseId: string, targetDid: string) =>
			this.transport.request<T.BillingAssignment>(
				'PUT',
				`/v1/billing/licenses/${segment(licenseId)}/assignment`,
				{ body: { targetDid } }
			),
		unassign: (licenseId: string) =>
			this.transport.request<void>(
				'DELETE',
				`/v1/billing/licenses/${segment(licenseId)}/assignment`,
				{ response: 'empty' }
			)
	};
	readonly chat = {
		status: () => this.transport.get<T.ChatStatus>('/v1/chat/status'),
		setConnection: (enabled: boolean) =>
			this.transport.request<T.ChatStatus>('PUT', '/v1/chat/connection', {
				body: { enabled }
			}),
		authorize: (handoffMode: string, pushEnabled: boolean) =>
			this.transport.request<T.LoginResponse>(
				'POST',
				'/v1/chat/authorization',
				{ body: { handoffMode, pushEnabled } }
			),
		unread: () =>
			this.transport.get<{ accepted: number; requests: number }>(
				'/v1/chat/unread-counts'
			),
		conversations: (
			status: 'accepted' | 'request' = 'accepted',
			options: T.PageQuery = {}
		) =>
			this.transport.page<T.ChatConversation>(
				'/v1/chat/conversations',
				{ status, ...pageQuery(options) },
				options.signal
			),
		conversation: (id: string) =>
			this.transport.get<T.ChatConversation>(conversationPath(id)),
		details: (id: string) =>
			this.transport.get<T.ChatDetails>(`${conversationPath(id)}/details`),
		messages: (id: string, options: T.PageQuery = {}) =>
			this.transport.page<T.ChatMessage>(
				`${conversationPath(id)}/messages`,
				pageQuery(options),
				options.signal
			),
		send: (id: string, operationId: string, text: string) =>
			this.transport.request<T.ChatSendResult>(
				'POST',
				`${conversationPath(id)}/messages`,
				{ body: { operationId, text } }
			),
		receipt: (operationId: string) =>
			this.transport.get<T.ChatReceipt>(
				`/v1/chat/operations/${segment(operationId)}`
			),
		read: (id: string, messageId: string) =>
			this.transport.request<T.ChatConversation>(
				'PUT',
				`${conversationPath(id)}/read`,
				{ body: { messageId } }
			),
		accept: (id: string) =>
			this.transport.request<T.ChatConversation>(
				'POST',
				`${conversationPath(id)}/accept`
			),
		mute: (id: string, muted: boolean) =>
			this.transport.request<T.ChatConversation>(
				muted ? 'PUT' : 'DELETE',
				`${conversationPath(id)}/mute`
			),
		leave: (id: string) =>
			this.transport.request<{ left: boolean }>(
				'POST',
				`${conversationPath(id)}/leave`
			),
		deleteMessage: (id: string, messageId: string) =>
			this.transport.request<{ deleted: boolean }>(
				'DELETE',
				`${conversationPath(id)}/messages/${segment(messageId)}`
			),
		availability: (recipientDid: string) =>
			this.transport.get<{ canChat: boolean; conversationId?: string }>(
				'/v1/chat/availability',
				{ recipientDid }
			),
		start: (recipientDid: string) =>
			this.transport.request<T.ChatConversation>(
				'POST',
				'/v1/chat/conversations',
				{ body: { recipientDid } }
			),
		report: (
			operationId: string,
			conversationId: string,
			reasonType: string,
			reason?: string,
			messageId?: string
		) =>
			this.transport.request<T.ChatReceipt>('POST', '/v1/chat/reports', {
				body: {
					operationId,
					conversationId,
					reasonType,
					reason,
					...(messageId ? { messageId } : { subjectType: 'conversation' })
				}
			}),
		sync: (
			view: 'inbox' | 'conversation' | 'none',
			conversationId?: string,
			knownRevision?: number
		) =>
			this.transport.request<T.ChatSync>('POST', '/v1/chat/sync', {
				body: { view, conversationId, knownRevision }
			}),
		incoming: () =>
			this.transport.get<T.ChatIncomingSetting>('/v1/chat/incoming-policy'),
		setIncoming: (allowIncoming: 'everyone' | 'following' | 'nobody') =>
			this.transport.request<T.ChatIncomingSetting>(
				'PUT',
				'/v1/chat/incoming-policy',
				{ body: { allowIncoming } }
			),
		pushPreference: () =>
			this.transport.get<{ pushEnabled: boolean }>('/v1/chat/push-preferences'),
		setPushPreference: (pushEnabled: boolean) =>
			this.transport.request<{ pushEnabled: boolean }>(
				'PUT',
				'/v1/chat/push-preferences',
				{ body: { pushEnabled } }
			),
		notificationOpen: (reference: string) =>
			this.transport.get<{ type: 'conversation'; conversationId: string }>(
				`/v1/chat/notification-opens/${segment(reference)}`
			)
	};
	readonly media = {
		linkPreview: (url: string, signal?: AbortSignal) =>
			this.transport.request<T.LinkPreview>('POST', '/v1/link-previews', {
				body: { url },
				signal
			}),
		imageAuthorization: () =>
			this.transport.request<T.ImageUploadAuthorization>(
				'POST',
				'/v1/blobs/images/authorization'
			),
		image: (bytes: Blob, mime: string, signal?: AbortSignal) =>
			this.transport.request<{
				blob: T.BlobReference;
				cid: string;
				mime: string;
				size: number;
			}>('POST', '/v1/blobs/images', {
				body: bytes,
				headers: { 'Content-Type': mime },
				signal
			}),
		videoLimits: () =>
			this.transport.get<T.VideoUploadLimits>('/v1/blobs/videos/limits'),
		videoAuthorization: () =>
			this.transport.request<T.JsonObject>(
				'POST',
				'/v1/blobs/videos/authorization'
			),
		cancelVideo: (jobId: string, signal?: AbortSignal) =>
			this.transport.request<void>(
				'POST',
				`/v1/blobs/videos/${segment(jobId)}/cancel`,
				{ signal }
			),
		videoStatus: (jobId: string, signal?: AbortSignal) => {
			if (
				!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(
					jobId
				)
			)
				throw new TypeError('Invalid video job');
			return this.transport.request<{ jobStatus: T.JsonObject }>(
				'POST',
				`/v1/blobs/videos/${segment(jobId)}/status`,
				{ signal }
			);
		},
		caption: (did: string, rkey: string, cid: string, signal?: AbortSignal) =>
			this.transport.request<string>(
				'GET',
				`${postPath(did, rkey)}/video-captions/${segment(cid)}`,
				{ response: 'text', signal }
			)
	};
	readonly moderation = {
		standing: () =>
			this.transport.get<T.ModerationStanding>('/v1/moderation/standing'),
		history: (options: T.PageQuery = {}) =>
			this.transport.page<T.ModerationHistoryItem>(
				'/v1/moderation/history',
				pageQuery(options),
				options.signal
			),
		entry: (reference: string) =>
			this.transport.get<T.Page<T.ModerationHistoryItem>>(
				`/v1/moderation/history/${segment(reference)}`
			)
	};
	readonly instagram = {
		account: () =>
			this.transport.get<T.InstagramAccount>(
				'/v1/migrations/instagram/account'
			),
		settings: (patch: T.JsonObject) =>
			this.transport.request<T.InstagramAccount>(
				'PATCH',
				'/v1/migrations/instagram/settings',
				{ body: patch }
			),
		revoke: () =>
			this.transport.request<void>(
				'DELETE',
				'/v1/migrations/instagram/account',
				{ response: 'empty' }
			),
		verify: () =>
			this.transport.request<T.InstagramVerification>(
				'POST',
				'/v1/migrations/instagram/verifications',
				{ body: {} }
			),
		verification: (id: string) =>
			this.transport.get<T.InstagramVerification>(
				`/v1/migrations/instagram/verifications/${segment(id)}`
			),
		currentVerification: () =>
			this.transport.get<{ verification: T.InstagramVerification | null }>(
				'/v1/migrations/instagram/verifications/current'
			),
		cancelVerification: (id: string) =>
			this.transport.request<void>(
				'DELETE',
				`/v1/migrations/instagram/verifications/${segment(id)}`,
				{ response: 'empty' }
			),
		confirmVerification: (id: string, discoverable: boolean) =>
			this.transport.request<T.InstagramVerificationConfirmation>(
				'POST',
				`/v1/migrations/instagram/verifications/${segment(id)}/confirm`,
				{ body: { discoverable } }
			),
		imports: (options: T.PageQuery = {}) =>
			this.transport.page<T.InstagramImport>(
				'/v1/migrations/instagram/imports',
				pageQuery(options),
				options.signal
			),
		createImport: (body: T.JsonObject) =>
			this.transport.request<T.InstagramImportCreate>(
				'POST',
				'/v1/migrations/instagram/imports',
				{ body }
			),
		import: (id: string) =>
			this.transport.get<T.InstagramImport>(
				`/v1/migrations/instagram/imports/${segment(id)}`
			),
		updateImport: (id: string, patch: T.JsonObject) =>
			this.transport.request<T.InstagramImport>(
				'PATCH',
				`/v1/migrations/instagram/imports/${segment(id)}`,
				{ body: patch }
			),
		deleteImport: (id: string) =>
			this.transport.request<void>(
				'DELETE',
				`/v1/migrations/instagram/imports/${segment(id)}`,
				{ response: 'empty' }
			),
		suggestions: (options: T.PageQuery = {}) =>
			this.transport.page<T.InstagramSuggestion>(
				'/v1/migrations/instagram/suggestions',
				pageQuery(options),
				options.signal
			),
		acceptSuggestion: (id: string, operationKey: string) =>
			this.transport.request<{ suggestionId: string; state: string }>(
				'POST',
				`/v1/migrations/instagram/suggestions/${segment(id)}/accept`,
				{
					operationKey,
					validate: (value) =>
						Object.keys(value).length === 2 &&
						value.suggestionId === id &&
						['followed', 'alreadyFollowing'].includes(String(value.state))
				}
			),
		dismissSuggestion: (id: string) =>
			this.transport.request<void>(
				'DELETE',
				`/v1/migrations/instagram/suggestions/${segment(id)}`,
				{ response: 'empty' }
			)
	};
}
