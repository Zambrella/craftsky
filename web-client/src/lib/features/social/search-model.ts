import type { RecentSearch } from '#lib/api/types.ts';
export function recentDestination(item: RecentSearch): string {
	const p = item.payload;
	if (item.type === 'profile' && typeof p.did === 'string')
		return '/app/profile/' + encodeURIComponent(p.did);
	if (item.type === 'hashtag' && typeof p.tag === 'string')
		return '/app/tags/' + encodeURIComponent(p.tag);
	const query = new URLSearchParams({
		q: typeof p.q === 'string' ? p.q : '',
		tab:
			item.type === 'post'
				? 'posts'
				: item.type === 'project'
					? 'projects'
					: 'projects'
	});
	if (p.sort === 'chronological' || p.sort === 'popular')
		query.set('sort', p.sort);
	if (p.filters && typeof p.filters === 'object' && !Array.isArray(p.filters))
		for (const [key, value] of Object.entries(p.filters)) {
			if (typeof value === 'string' || typeof value === 'boolean')
				query.set(key, String(value));
			else if (Array.isArray(value))
				for (const entry of value)
					if (typeof entry === 'string') query.append(key, entry);
		}
	return '/app/search?' + query;
}
