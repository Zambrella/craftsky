/** AppView wire contracts. IDs and cursors remain opaque strings. Dates are ISO strings. */
export type Json =
	| null
	| boolean
	| number
	| string
	| Json[]
	| { [key: string]: Json | undefined };
export type JsonObject = { [key: string]: Json | undefined };
export type DID = string;
export type RecordKey = string;
export type CID = string;
export type ATURI = string;
export interface PageQuery {
	cursor?: string | null;
	limit?: number;
	signal?: AbortSignal;
}
export interface Page<T> {
	items: T[];
	cursor?: string | null;
}
export interface ProfileCustomisation {
	colour: string;
	profileBackground: string;
}
export interface BlobReference {
	$type?: 'blob';
	ref: { $link: string };
	mimeType: string;
	size: number;
}
export interface StrongRef {
	uri: ATURI;
	cid: CID;
}
export interface Facet {
	index: { byteStart: number; byteEnd: number };
	features: JsonObject[];
}
export interface ProfileLink extends StrongRef {
	url: string;
	title?: string;
	icon?: string;
	iconBlob?: BlobReference;
	createdAt?: string;
}
export interface ProfileLinkInput {
	rkey: string;
	url: string;
	title?: string;
	cid?: CID;
	createdAt: string;
	icon?: BlobReference;
}
export interface Author {
	did: DID;
	handle: string;
	displayName?: string | null;
	avatar?: string | null;
	avatarCid?: string | null;
	crafts?: string[];
	customisation?: ProfileCustomisation;
	muted?: boolean;
	blocking?: boolean;
	blockedBy?: boolean;
	accountType?: 'regular' | 'business';
	description?: string | null;
	isCraftskyProfile?: boolean;
	viewerIsFollowing?: boolean;
}
export interface Profile extends Author {
	description?: string | null;
	pronouns?: string | null;
	banner?: string | null;
	crafts: string[];
	links: ProfileLink[];
	editableLinks?: ProfileLink[];
	createdAt?: string;
	viewerIsFollowing: boolean;
	isCraftskyProfile: boolean;
	followerCount?: number;
	followingCount?: number;
	mutualFollowerCount?: number;
	postCount?: number;
	projectCount?: number;
	postsLast7Days?: number;
	moderation?: { warningKind: string };
	business?: BusinessProfile;
	hasUpcomingEvents?: boolean;
}
export interface ProfilePage extends Page<Author> {
	totalCount: number;
}
export interface ProfileUpdate {
	displayName?: string;
	description?: string;
	pronouns?: string;
	crafts?: string[];
	links?: ProfileLinkInput[];
	removedLinks?: StrongRef[];
	avatar?: BlobReference | null;
	banner?: BlobReference | null;
}
export interface Image {
	cid?: CID;
	mime?: string;
	size?: number;
	alt: string;
	thumb?: string;
	fullsize?: string;
	aspectRatio?: { width: number; height: number };
}
export interface Video {
	cid: CID;
	mime: string;
	size: number;
	alt: string;
	playlist: string;
	embedUrl?: string;
	thumbnail?: string;
	aspectRatio?: { width: number; height: number };
	captions?: { cid: CID; lang: string; uri: string }[];
}
export interface ProjectPattern {
	url?: string;
	name?: string;
	difficulty?: string;
	designer?: string;
	publisher?: string;
	selfDrafted?: boolean;
	nameFacets?: Facet[];
	designerFacets?: Facet[];
	publisherFacets?: Facet[];
}
export interface Project {
	common: {
		craftType: string;
		title?: string;
		status?: string;
		duration?: string;
		additionalCrafts?: { craftType: string }[];
		pattern?: ProjectPattern;
		materials?: { text: string; facets?: Facet[] }[];
		colors?: string[];
		designTags?: string[];
		tags?: string[];
	};
	details?: JsonObject;
}
export interface Post {
	uri: ATURI;
	cid: CID;
	rkey: RecordKey;
	text: string;
	author: Author;
	createdAt: string;
	indexedAt: string;
	tags: string[];
	langs: string[];
	sponsored: boolean;
	images?: Image[];
	video?: Video;
	project?: Project;
	facets?: Facet[] | null;
	reply?: { root: StrongRef; parent: StrongRef } | null;
	quote?: StrongRef | null;
	quoteView?: { state: string; revealable: boolean; post?: QuotePreviewPost };
	likeCount: number;
	repostCount: number;
	quoteCount: number;
	replyCount: number;
	viewerHasLiked: boolean;
	viewerHasReposted: boolean;
	viewerHasReplied: boolean;
	viewerHasSaved: boolean;
	viewerSavedFolderId?: string | null;
	editedAt?: string;
	recordRevision?: string | null;
	moderation?: { warningKind: string };
	availability?: 'muted' | 'blocked';
	relationship?: { state: string; revealable: boolean };
	externalImport?: { source: string };
	external?: {
		uri: string;
		title: string;
		description: string;
		thumb?: { url: string; cid: string; mime: string; size: number };
	};
}
/** Compact one-level quote hydration deliberately omits a record key and engagement. */
export type QuotePreviewPost = Pick<
	Post,
	'uri' | 'cid' | 'text' | 'author' | 'createdAt'
> &
	Partial<
		Pick<
			Post,
			| 'images'
			| 'project'
			| 'sponsored'
			| 'editedAt'
			| 'recordRevision'
			| 'externalImport'
			| 'external'
			| 'moderation'
			| 'availability'
			| 'relationship'
		>
	>;
export interface ProtectedPost {
	uri?: string;
	availability: 'muted' | 'blocked';
	relationship: { state: string; revealable: boolean };
}
export type PostPresentation = Post | ProtectedPost;
export interface TimelineItem {
	itemKey: string;
	post: PostPresentation;
	reason?: {
		type: 'repost';
		by: Author;
		uri: ATURI;
		cid?: CID;
		createdAt: string;
		indexedAt: string;
	};
}
export interface CreatePostImage {
	image: BlobReference;
	alt: string;
	aspectRatio?: { width: number; height: number };
}
export interface PostPayload {
	text: string;
	langs: string[];
	sponsored: boolean;
	images?: CreatePostImage[];
	facets?: Facet[];
	project?: Project;
	reply?: { root: StrongRef; parent: StrongRef };
	embed?:
		| { quote: StrongRef; imageCid?: CID }
		| {
				external: {
					uri: string;
					title: string;
					description: string;
					thumb?: BlobReference;
				};
		  }
		| {
				video: {
					jobId: string;
					blob: BlobReference;
					alt?: string;
					aspectRatio?: { width: number; height: number };
				};
		  };
}
export type PostEmbedEdit =
	| {
			$type?: 'app.bsky.embed.video';
			video?: BlobReference;
			jobId?: string;
			alt?: string | null;
			aspectRatio?: { width?: number; height?: number } | null;
	  }
	| {
			$type?: 'app.bsky.embed.external';
			external?: Partial<{
				uri: string;
				title: string;
				description: string;
				thumb: BlobReference | null;
			}>;
	  };
export type PostEditChanges = Partial<{
	text: string;
	sponsored: boolean;
	langs: string[];
	facets: Facet[] | null;
	images:
		| (
				| CreatePostImage
				| {
						existingIndex: number;
						alt?: string;
						aspectRatio?: { width?: number; height?: number };
				  }
		  )[]
		| null;
	embed: PostEmbedEdit | null;
	project: { common?: Partial<Project['common']>; details?: JsonObject | null };
}>;
export interface PostEditSeed {
	post: Post;
	baseCid: CID;
	recordRevision?: string | null;
	editable: JsonObject;
}
export interface PostEditResult {
	acceptedCid: CID;
	changed: boolean;
	editedAt?: string;
	recordRevision?: string;
	observationCid?: CID;
	observationRevision?: string;
}
export interface InteractionWrite {
	uri: ATURI;
	cid: CID;
}
export interface RecordMutation {
	uri?: ATURI;
	cid: CID;
	did?: DID;
	rkey?: RecordKey;
}
export interface ReplyItem {
	post: PostPresentation;
	replyTo?: Author;
}
export interface ReplyPage extends Page<ReplyItem> {
	loaded: boolean;
}
export interface CommentItem {
	post: PostPresentation;
	replies: ReplyPage;
	placement: 'focused' | 'viewerAuthored' | 'normal';
}
export interface CommentSection {
	post: PostPresentation;
	comments: Page<CommentItem>;
	sort: 'oldest' | 'newest' | 'follows';
	focus?: {
		status: 'included' | 'notFound' | 'mismatchedRoot';
		kind?: 'comment' | 'reply';
		uri?: ATURI;
	};
}
export interface ProfilePins {
	standardPostUri?: ATURI | null;
	projectPostUri?: ATURI | null;
}
export interface ReportSubmission {
	reasonType: string;
	details?: string;
}
export interface ReportResult {
	id?: string;
	reportId?: string;
	status?: string;
	[key: string]: Json | undefined;
}
export interface Relationship {
	did: DID;
	muted: boolean;
	blocking: boolean;
	blockedBy: boolean;
}
export interface FollowerGrowth {
	period: string;
	[key: string]: Json | undefined;
}
export type SearchSort = 'chronological' | 'popular';
export type QueryValue =
	string | number | boolean | null | undefined | readonly string[];
export interface ProjectFilters {
	[key: string]: QueryValue;
}
export interface ProjectQuery extends PageQuery {
	craftType?: string[];
	sort?: SearchSort;
	filters?: ProjectFilters;
}
export interface SearchQuery extends PageQuery {
	q: string;
	sort?: SearchSort;
	filters?: ProjectFilters;
}
export interface Hashtag {
	tag: string;
	postCount: number;
}
export interface SearchSuggestions {
	profiles: { items: Author[]; hasMore: boolean };
	hashtags: { items: Hashtag[]; hasMore: boolean };
}
export interface TopHashtags {
	groups: { craftType: string; items: { tag: string; count: number }[] }[];
}
export interface RecentSearch {
	displayLabel: string;
	id: string;
	type: 'query' | 'hashtag' | 'profile' | 'post' | 'project';
	payload: JsonObject;
	createdAt?: string;
	updatedAt?: string;
}
export interface SaveRecentSearch {
	displayLabel: string;
	type: RecentSearch['type'];
	payload: JsonObject;
}
export interface LinkPreview {
	url?: string;
	uri?: string;
	title: string;
	description: string;
	image?: string;
	thumb?: Image;
	[key: string]: Json | Image | undefined;
}
export interface SavedState {
	savedAt: string;
	folderId?: string | null;
}
export interface SavedItem extends SavedState {
	post: PostPresentation;
}
export interface SavedFolder {
	id: string;
	name: string;
	createdAt: string;
	updatedAt: string;
}
export interface SavedQuery extends PageQuery {
	sort?: 'newest' | 'oldest';
	folderId?: string;
	unfiled?: boolean;
}
export interface ScheduledPostSummary {
	id: string;
	kind: 'standard' | 'project';
	status: string;
	textPreview: string;
	projectTitle?: string;
	scheduledAt: string;
	firstMediaId?: string;
	needsAttentionExpiresAt?: string;
	lastErrorCode?: string;
	subscriptionRequired?: boolean;
}
export interface ScheduledPost {
	id: string;
	operationId: string;
	status: string;
	scheduledAt: string;
	payload: JsonObject;
	subscriptionRequired?: boolean;
	lastErrorCode?: string;
}
export interface BusinessProfile {
	cid: CID;
	businessTypes?: { value: string; known: boolean }[];
	offerings?: { value: string; known: boolean }[];
	tagline?: string;
	hoursNote?: string;
	serviceArea?: string;
	location?: JsonObject;
	primaryAction?: JsonObject;
	products?: {
		title: string;
		uri?: string;
		image?: Image;
		price?: JsonObject;
	}[];
}
export interface BusinessEvent {
	uri: ATURI;
	cid: CID;
	rkey: RecordKey;
	did: DID;
	name: string;
	startsAt: string;
	endsAt: string;
	roles: { value: string; known: boolean }[];
	status: { value: string; known: boolean };
	isAllDay: boolean;
	createdAt: string;
	past: boolean;
	publicSuppressionReasons: string[];
	upcomingExclusionReasons: string[];
	mode?: { value: string; known: boolean };
	timeZone?: string;
	summary?: string;
	venueName?: string;
	eventUri?: string;
	registrationUri?: string;
	image?: Image;
}
export interface AppNotification {
	id: string;
	type: string;
	createdAt: string;
	indexedAt: string;
	isNew: boolean;
	contentAvailable?: boolean;
	actor?: Author & { available?: boolean };
	subjectPost?: PostPresentation;
	preview?: { text: string; projectTitle?: string; role: string };
	quotedPreview?: { text: string; projectTitle?: string; role?: string };
	uri?: ATURI;
	cid?: CID;
	rkey?: string;
	reply?: StrongRef & { rkey: string };
	caseReference?: string;
}
export type NotificationCategory =
	| 'follow'
	| 'like'
	| 'repost'
	| 'reply'
	| 'mention'
	| 'quote'
	| 'instagramMatch'
	| 'everythingElse'
	| 'moderation';
export interface NotificationPreferences {
	preferences: Record<
		string,
		{ scope: 'everyone' | 'peopleIFollow'; pushEnabled: boolean }
	>;
}
export interface NotificationPreferencePatch {
	preferences: Record<
		string,
		Partial<{ scope: 'everyone' | 'peopleIFollow'; pushEnabled: boolean }>
	>;
}
export interface LanguagePreferences {
	primaryLanguage: string;
	contentLanguages: string[];
}
export interface OnboardingStatus {
	completed: boolean;
	completedAt?: string;
	requiredPolicyVersion: string;
	acceptedPolicyVersion?: string;
	acceptedAt?: string;
}
export interface EligibilityStatus {
	state: 'eligible' | 'restricted';
	appealable: boolean;
	appealGuidance?: string;
}
export interface WhoAmI {
	did: DID;
	handle: string;
	[key: string]: Json | undefined;
}
export interface LoginResponse {
	authUrl: string;
}
export interface AccountDeletionIntent {
	jobId: string;
	authUrl: string;
	confirmationDid: DID;
	expiresAt: string;
}
export interface SubscriptionAccess {
	did: DID;
	effectiveTier: 'free' | 'plus' | 'business';
	givesAccess: boolean;
	accessEndsAt?: string;
	assignedTier?: 'plus' | 'business';
}
export interface BillingSubscription {
	id: string;
	productId: string;
	store: string;
	status: string;
	givesAccess: boolean;
	pendingPayment: boolean;
	autoRenewalStatus?: string;
	currentPeriodStartsAt?: string;
	currentPeriodEndsAt?: string;
	endsAt?: string;
	anomaly: string;
}
export interface BillingLicense {
	id: string;
	subscriptionId: string;
	tier: 'plus' | 'business';
	assignedDid?: DID;
	assignedAt?: string;
	assignable: boolean;
	anomaly: string;
}
export interface BillingState {
	billingAccountId: string;
	revenueCatAppUserId: string;
	requestedGeneration: number;
	reconciledGeneration: number;
	reconciliationRequestedAt?: string;
	reconciledAt?: string;
	reconciliationStale: boolean;
	subscriptions: BillingSubscription[];
	licenses: BillingLicense[];
}
export interface BillingAssignment {
	licenseId: string;
	targetDid: DID;
	assignedAt: string;
}
export interface ChatActor {
	did: DID;
	handle?: string;
	displayName?: string;
	avatar?: string;
}
export interface ChatMessage {
	id: string;
	kind: 'text' | 'deleted' | 'unsupported';
	revision: string;
	sentAt: string;
	senderDid?: DID;
	text?: string;
}
export interface ChatConversation {
	id: string;
	revision: string;
	status: 'accepted' | 'request';
	muted: boolean;
	unreadCount: number;
	canSend: boolean;
	peer: ChatActor;
	lastMessage?: ChatMessage;
}
export interface ChatDetails {
	conversation: ChatConversation;
	messages: Page<ChatMessage>;
}
export interface ChatStatus {
	enabled: boolean;
	generation: number;
	pushEnabled: boolean;
	state: string;
	capabilities: {
		canRead: boolean;
		canSend: boolean;
		canReport: boolean;
		canSetPolicy: boolean;
	};
	syncState?: string;
	recoveryCode?: string;
	lastSuccessAt?: string;
}
export interface ChatReceipt {
	operationId: string;
	kind: string;
	state: 'confirmed' | 'failed' | 'uncertain';
	messageId?: string;
	failureCode?: string;
}
export interface ChatSendResult extends ChatReceipt {
	message?: ChatMessage;
}
export interface ChatSync {
	revision: number;
	state: string;
	lastSuccessAt?: string;
}
export interface ChatIncomingSetting {
	allowIncoming: 'everyone' | 'following' | 'nobody';
	persisted: boolean;
}
export interface VideoUploadLimits {
	canUpload: boolean;
	message?: string;
	error?: string;
	[key: string]: Json | undefined;
}
export interface ImageUploadAuthorization {
	[key: string]: Json | undefined;
}
export interface InstagramAccountLink extends JsonObject {
	state: string;
	username: string;
	discoverable: boolean;
	conflictPending: boolean;
	reactivationRequired: boolean;
	verifiedAt: string;
}
export interface InstagramAccount {
	integrationAvailable: boolean;
	account: InstagramAccountLink | null;
	[key: string]: Json | undefined;
}
export interface InstagramVerificationConfirmation {
	state: string;
	account: InstagramAccountLink;
}
export interface InstagramVerification {
	verificationId: string;
	state: string;
	expiresAt: string;
	challenge?: string;
	dmUrl?: string;
	candidateUsername?: string;
	retryCode?: string;
	[key: string]: Json | undefined;
}
export interface InstagramImport {
	importId: string;
	state: string;
	sourceType: string;
	followingCount: number;
	createdAt: string;
	[key: string]: Json | undefined;
}
export interface InstagramImportCreate {
	import: InstagramImport;
	counts: { followingCount: number };
}
export interface InstagramSuggestion {
	suggestionId: string;
	target: Author;
	createdAt: string;
	[key: string]: unknown;
}
export interface ModerationStanding {
	[key: string]: Json | undefined;
}
export interface ModerationHistoryItem {
	[key: string]: Json | undefined;
}
