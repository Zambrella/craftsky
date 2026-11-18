import guide from './guide-data.json';
// Optional local signposting only; no classification, persistence or telemetry.
export function shouldOfferSearchSupport(query: string): boolean {
	const normalized = query
		.toLowerCase()
		.replace(/[-_\u2010-\u2015]/g, ' ')
		.replace(/\s+/g, ' ')
		.trim();
	return guide.searchTerms.some((term) =>
		new RegExp(`(^|[^a-z0-9])${term}([^a-z0-9]|$)`).test(normalized)
	);
}
