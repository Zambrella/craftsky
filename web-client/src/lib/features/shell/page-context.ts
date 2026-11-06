/** Both standalone authenticated routes and signed-in public URLs use this context. */
export function authenticatedContext<
	T extends { session?: unknown; authenticated?: unknown }
>(data: T) {
	return (data.authenticated ?? (data.session ? data : undefined)) as
		| {
				session: { id: string; did: string; handle: string };
				accounts: { id: string; did: string; handle: string }[];
				subscriptionsEnabled?: boolean;
		  }
		| undefined;
}

const titles: Record<string, string> = {
	feed: 'Feed',
	projects: 'Projects',
	search: 'Search',
	notifications: 'Notifications',
	messages: 'Messages',
	profile: 'Your profile',
	profiles: 'Profile',
	posts: 'Post',
	events: 'Event',
	saved: 'Saved posts',
	scheduled: 'Scheduled posts',
	drafts: 'Drafts',
	settings: 'Settings',
	compose: 'New post'
};
const settings: Record<string, string> = {
	preferences: 'Preferences',
	appearance: 'Appearance',
	customisation: 'Profile customisation',
	messaging: 'Messaging settings',
	followers: 'Followers',
	following: 'Following',
	growth: 'Follower growth',
	about: 'About CraftSky',
	products: 'Products',
	events: 'Events',
	privacy: 'Privacy settings',
	account: 'Account settings',
	business: 'Business settings',
	notifications: 'Notification settings',
	languages: 'Content languages',
	instagram: 'Instagram import',
	subscriptions: 'Subscriptions',
	billing: 'Billing',
	moderation: 'Moderation',
	blocked: 'Blocked accounts',
	muted: 'Muted accounts',
	delete: 'Delete account'
};
/** Titles use static route labels, never private text, recipient routing, or query values. */
export function routeTitle(pathname: string): string {
	const [section, child] = pathname.split('/').filter(Boolean);
	let title = titles[section] ?? 'CraftSky';
	if (section === 'settings' && child) title = settings[child] ?? 'Settings';
	if (section === 'compose')
		title =
			(
				{
					project: 'Project post',
					edit: 'Edit post',
					reply: 'Reply',
					quote: 'Quote',
					draft: 'Edit draft',
					schedule: 'Edit scheduled post'
				} as Record<string, string>
			)[child] ?? title;
	if (section === 'profile' && child === 'edit') title = 'Edit profile';
	if (section === 'feed' && child === 'filters') title = 'Feed filters';
	if (section === 'notifications' && child === 'settings')
		title = 'Notification settings';
	if (section === 'messages')
		title =
			child === 'new'
				? 'New message'
				: child === 'requests'
					? 'Message requests'
					: child === 'conversations'
						? 'Conversation'
						: title;
	return title === 'CraftSky' ? title : `${title} · CraftSky`;
}
