import type { AppNotification, Profile } from '#lib/api/types.ts';
import { visible } from './model';

export const notificationCategories = [
	['', 'All categories', 'activity'],
	['follow', 'Follows', 'profile'],
	['like', 'Likes', 'heart'],
	['reply', 'Comments and replies', 'comment'],
	['mention', 'Mentions', 'comment'],
	['quote', 'Quotes', 'share'],
	['repost', 'Reposts', 'repost'],
	['instagramMatch', 'Instagram matches', 'profile'],
	['moderation', 'Moderation', 'shield']
] as const;

export function notificationCopy(item: AppNotification) {
	const type = item.type.replaceAll('_', '').toLowerCase();
	const sourceUnavailable =
		item.actor?.available === false ||
		item.contentAvailable === false ||
		!!(item.subjectPost && !visible(item.subjectPost));
	const roleSource =
		type === 'quote' ? item.quotedPreview?.role : item.preview?.role;
	const role =
		roleSource === 'reply'
			? 'reply'
			: roleSource === 'comment'
				? 'comment'
				: 'post';
	const actions: Record<string, [string, string]> = {
		follow: ['followed you', 'profile'],
		like: [`liked your ${role}`, 'heart'],
		repost: [`reposted your ${role}`, 'repost'],
		quote: [`quoted your ${role}`, 'share'],
		reply: [
			role === 'post' ? 'commented on your post' : `replied to your ${role}`,
			'comment'
		],
		mention: [`mentioned you in a ${role}`, 'comment'],
		instagrammatch: ['was found through your Instagram following', 'profile'],
		moderation: ['Account moderation update', 'shield']
	};
	const [action, icon] = actions[type] || ['New activity', 'activity'];
	const subject =
		item.subjectPost && visible(item.subjectPost)
			? item.subjectPost
			: undefined;
	const preview = sourceUnavailable
		? undefined
		: item.preview ||
			(type !== 'reply' && subject
				? { text: subject.text, projectTitle: subject.project?.common.title }
				: undefined);
	const contextText =
		type === 'reply' && !sourceUnavailable && subject
			? (subject.project?.common.title || subject.text)
					.replace(/\s+/g, ' ')
					.trim()
			: '';
	return {
		action: sourceUnavailable ? 'Activity unavailable' : action,
		icon,
		available: !sourceUnavailable,
		canFollow:
			!sourceUnavailable &&
			['follow', 'instagrammatch'].includes(type) &&
			!!item.actor &&
			!item.actor.blocking &&
			!item.actor.blockedBy,
		preview: preview?.projectTitle || preview?.text || '',
		previewTitle: preview?.projectTitle || '',
		previewText: preview?.text || '',
		quotedPreview:
			!sourceUnavailable && type === 'quote' ? item.quotedPreview : undefined,
		context: contextText
			? subject?.reply
				? `↳ You: “${contextText}”`
				: `↳ Your post: ${contextText}`
			: ''
	};
}

/** Matches ProfileTabPolicy in the Flutter client. Explicit old deep links remain addressable. */
export function profileTabs(profile: Profile, own: boolean): string[] {
	const ordinary = ['projects', 'posts', 'comments', 'reposts'];
	if (
		profile.accountType !== 'business' ||
		profile.blocking ||
		profile.blockedBy
	)
		return ordinary;
	return [
		...(own || profile.business?.products?.length ? ['products'] : []),
		...(own || profile.hasUpcomingEvents ? ['events'] : []),
		'about',
		...ordinary
	];
}

export const tabLabel = (tab: string) =>
	tab === 'events'
		? 'Upcoming events'
		: tab === 'comments'
			? 'Comments & replies'
			: tab[0].toUpperCase() + tab.slice(1);

export function joinedAge(createdAt: string, now = Date.now()): string {
	const days = Math.max(
		0,
		Math.floor((now - Date.parse(createdAt)) / 86400000)
	);
	if (!Number.isFinite(days)) return '';
	return days === 0
		? 'today'
		: days < 31
			? `${days}d`
			: days < 365
				? `${Math.floor(days / 30)}mo`
				: `${Math.floor(days / 365)}y`;
}
