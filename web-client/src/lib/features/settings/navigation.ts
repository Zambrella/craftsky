import { record, text, validDid } from './model.ts';

/** Store-owned management pages; never navigate to an arbitrary response URL. */
export function storeManagementURL(store: string): string | undefined {
	if (store === 'app_store')
		return 'https://apps.apple.com/account/subscriptions';
	if (store === 'play_store')
		return 'https://play.google.com/store/account/subscriptions';
}

export function moderationSubject(value: unknown): {
	reference: string;
	href?: string;
	label?: string;
} {
	const snapshot = record(value),
		did = text(snapshot.did),
		uri = text(snapshot.uri);
	const reference = uri || text(snapshot.submittedHandle) || did;
	if (!validDid(did)) return { reference };
	if (snapshot.type === 'account' || snapshot.type === 'profileLink')
		return {
			reference,
			href: '/profiles/' + encodeURIComponent(did),
			label: 'View profile'
		};
	const match =
		/^at:\/\/([^/]+)\/(social\.craftsky\.(?:feed\.post|business\.event))\/([A-Za-z0-9._~:-]+)$/.exec(
			uri
		);
	if (!match || match[1] !== did || match[3] === '.' || match[3] === '..')
		return { reference };
	if (snapshot.type === 'post' && match[2] === 'social.craftsky.feed.post')
		return {
			reference,
			href: `/posts/${encodeURIComponent(did)}/${encodeURIComponent(match[3])}`,
			label: 'View post'
		};
	if (
		snapshot.type === 'event' &&
		match[2] === 'social.craftsky.business.event'
	)
		return {
			reference,
			href: `/events/${encodeURIComponent(did)}/${encodeURIComponent(match[3])}`,
			label: 'View event'
		};
	return { reference };
}
