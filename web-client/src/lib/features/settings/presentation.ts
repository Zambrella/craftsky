/** Order and labels mirror Flutter SettingsPage and NotificationCategory.preferenceValues. */
export const settingsGroups = [
	{
		title: 'Preferences',
		items: [
			['appearance', 'Appearance'],
			['customisation', 'Customisation'],
			['languages', 'Languages'],
			['notifications', 'Notifications'],
			['messaging', 'Direct messages'],
			['privacy', 'Privacy']
		]
	},
	{
		title: 'Connections',
		items: [
			['growth', 'Growth'],
			['followers', 'Followers'],
			['following', 'Following'],
			['muted', 'Muted accounts'],
			['blocked', 'Blocked accounts']
		]
	},
	{ title: 'Discovery', items: [['instagram', 'Find people from Instagram']] },
	{
		title: 'Business',
		items: [
			['events', 'Events'],
			['products', 'Products']
		]
	},
	{
		title: 'General',
		items: [
			['safety', 'Safety and support'],
			['moderation', 'Account standing'],
			['account', 'Account'],
			['about', 'About'],
			['feedback', 'Feedback']
		]
	}
] as const;

export const notificationCategories = [
	'like',
	'follow',
	'reply',
	'mention',
	'quote',
	'repost',
	'instagramMatch',
	'moderation',
	'everythingElse'
] as const;
export const notificationLabels: Record<string, string> = {
	like: 'Likes',
	follow: 'Follows',
	reply: 'Comments & replies',
	mention: 'Mentions',
	quote: 'Quotes',
	repost: 'Reposts',
	instagramMatch: 'Instagram matches',
	moderation: 'Moderation',
	everythingElse: 'Everything else'
};
export const backgroundLabels: Record<string, string> = {
	none: 'None',
	'animated-flow': 'Aurora',
	'animated-waves': 'Colour pools',
	'animated-weave': 'Sunset',
	'animated-grain': 'Pigment',
	'animated-marble': 'Marble',
	'animated-textile': 'Woven',
	bayerdark: 'Dither',
	cubedark: 'Grid',
	dotcrossdark: 'Cross stitch',
	scallopdark: 'Scallops',
	skewdark: 'Diagonal weave',
	x2: 'Crosshatch',
	'craft-sewing': 'Sewing',
	'craft-knitting': 'Knitting',
	'craft-crochet': 'Crochet',
	'craft-quilting': 'Quilting',
	'craft-embroidery': 'Embroidery'
};
export function visibleSettingsGroups(restricted: boolean, business: boolean) {
	return settingsGroups
		.map((group) => ({
			...group,
			items: group.items.filter(
				([slug]) =>
					(!restricted ||
						[
							'safety',
							'moderation',
							'account',
							'muted',
							'blocked',
							'about',
							'feedback'
						].includes(slug)) &&
					(group.title !== 'Business' || business)
			)
		}))
		.filter((group) => group.items.length);
}
