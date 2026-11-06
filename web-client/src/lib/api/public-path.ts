const filters = new Set([
	'craftType',
	'status',
	'projectType',
	'projectSubtype',
	'patternDifficulty',
	'color',
	'designTag',
	'yarnWeight',
	'piecingTechnique',
	'quiltingMethod',
	'selfDrafted'
]);
const invalid = (): never => {
	throw new TypeError('Invalid public read request');
};
const identifier = (value: string) => {
	const id = value.replace(/^@/, '');
	if (['me', 'recent'].includes(id)) return false;
	return (
		/^did:[a-z0-9]+:[a-zA-Z0-9._:%-]+$/.test(id) ||
		/^[a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z0-9-]+$/.test(id)
	);
};
const key = (value: string) =>
	!['.', '..'].includes(value) && /^[a-zA-Z0-9._~:-]{1,512}$/.test(value);

/** The browser and proxy use exactly this closed public read contract. */
export function publicReadPath(
	path: string,
	query = new URLSearchParams()
): string {
	let parts: string[];
	try {
		parts = path.split('/').map(decodeURIComponent);
	} catch {
		return invalid();
	}
	if (parts.some((part) => !part || /[/?#\\\u0000-\u0020\u007f]/.test(part)))
		return invalid();
	const [area, id, record, section] = parts;
	const feed = path === 'projects';
	const search =
		parts.length === 2 &&
		area === 'search' &&
		['profiles', 'posts', 'projects', 'hashtags'].includes(id);
	const tag =
		parts.length === 4 &&
		area === 'search' &&
		id === 'hashtags' &&
		section === 'posts' &&
		Array.from(record).length <= 128;
	const profile =
		area === 'profiles' &&
		identifier(id || '') &&
		(parts.length === 2 ||
			(parts.length === 3 &&
				['posts', 'projects', 'comments', 'reposts', 'events'].includes(
					record
				)));
	const caption =
		parts.length === 5 &&
		area === 'posts' &&
		identifier(id || '') &&
		id.startsWith('did:') &&
		key(record || '') &&
		section === 'video-captions' &&
		key(parts[4]);
	const post =
		area === 'posts' &&
		/^did:/.test(id || '') &&
		identifier(id) &&
		key(record || '') &&
		(parts.length === 3 ||
			(parts.length === 4 &&
				['thread', 'likes', 'reposts', 'quotes'].includes(section)));
	const event =
		area === 'events' &&
		parts.length === 3 &&
		/^did:/.test(id || '') &&
		identifier(id) &&
		key(record || '');
	if (!feed && !search && !tag && !profile && !post && !event && !caption)
		return invalid();
	const paged =
		feed ||
		search ||
		tag ||
		(profile && parts.length === 3) ||
		(post && parts.length === 4);
	const allowed = new Set(paged ? ['cursor', 'limit'] : []);
	if (search) allowed.add('q');
	if (feed) {
		allowed.add('sort');
		for (const filter of filters) allowed.add(filter);
	}
	let filterCount = 0;
	for (const name of new Set(query.keys())) {
		const values = query.getAll(name);
		if (
			!allowed.has(name) ||
			values.length > 10 ||
			(!filters.has(name) && values.length !== 1)
		)
			return invalid();
		if (filters.has(name)) {
			filterCount += values.length;
			if (values.some((value) => !value || Array.from(value).length > 256))
				return invalid();
		}
	}
	if (filterCount > 50) return invalid();
	if (
		query.has('limit') &&
		(!/^[0-9]+$/.test(query.get('limit')!) ||
			Number(query.get('limit')) < 1 ||
			Number(query.get('limit')) > 20)
	)
		return invalid();
	if ((query.get('cursor') || '').length > 2048) return invalid();
	if (
		search &&
		(!query.get('q')?.trim() || Array.from(query.get('q')!.trim()).length > 256)
	)
		return invalid();
	if (
		feed &&
		query.has('sort') &&
		!['chronological', 'popular'].includes(query.get('sort')!)
	)
		return invalid();
	const canonical = parts.map(encodeURIComponent).join('/');
	return canonical + (query.size ? '?' + query.toString() : '');
}
