/** Flutter's ChatTimeHeader groups authoritative messages by local day or a five-minute gap. */
export function startsTimeSection(current: string, previous?: string): boolean {
	const next = new Date(current),
		prior = previous ? new Date(previous) : undefined;
	if (!Number.isFinite(next.getTime())) return false;
	return (
		!prior ||
		!Number.isFinite(prior.getTime()) ||
		next.toDateString() !== prior.toDateString() ||
		next.getTime() - prior.getTime() >= 5 * 60_000
	);
}

export function timeSectionLabel(
	value: string,
	now = new Date(),
	locale?: string
): string {
	const date = new Date(value);
	if (!Number.isFinite(date.getTime())) return 'Message time unavailable';
	const yesterday = new Date(
		now.getFullYear(),
		now.getMonth(),
		now.getDate() - 1
	);
	const day =
		date.toDateString() === now.toDateString()
			? 'Today'
			: date.toDateString() === yesterday.toDateString()
				? 'Yesterday'
				: new Intl.DateTimeFormat(locale, {
						weekday: 'short',
						day: 'numeric',
						month: 'long',
						...(date.getFullYear() !== now.getFullYear()
							? { year: 'numeric' as const }
							: {})
					}).format(date);
	return (
		day +
		' at ' +
		new Intl.DateTimeFormat(locale, {
			hour: 'numeric',
			minute: '2-digit'
		}).format(date)
	);
}

export function inboxTime(
	value: string,
	now = new Date(),
	locale?: string
): string {
	const date = new Date(value),
		seconds = Math.max(0, (now.getTime() - date.getTime()) / 1000);
	if (!Number.isFinite(seconds)) return '';
	if (seconds < 60) return 'now';
	if (seconds < 3600) return Math.floor(seconds / 60) + 'm';
	if (seconds < 86400) return Math.floor(seconds / 3600) + 'h';
	if (seconds < 7 * 86400) return Math.floor(seconds / 86400) + 'd';
	const parts = new Intl.DateTimeFormat(locale, {
		day: 'numeric',
		month: 'short',
		year: '2-digit'
	}).formatToParts(date);
	return ['day', 'month', 'year']
		.map((type) => parts.find((part) => part.type === type)?.value)
		.join(' ');
}

export function historyScrollMode(
	previous: { id: string; first?: string; last?: string } | undefined,
	next: { id: string; first?: string; last?: string },
	nearEnd: boolean
): 'end' | 'preserve' | 'none' {
	if (!previous || previous.id !== next.id) return 'end';
	if (previous.first !== next.first && previous.last === next.last)
		return 'preserve';
	return nearEnd && previous.last !== next.last ? 'end' : 'none';
}

export function messageBackURL(view: string, id?: string): string {
	return id && (view === 'details' || view === 'report')
		? '/app/messages/' + encodeURIComponent(id)
		: '/app/messages';
}
