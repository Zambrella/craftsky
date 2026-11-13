// Explicit consumer API allowlist. Operator, dev, auth and public routes are excluded.
const routes: [string, RegExp][] = [
	['GET', /^\/v1\/preferences\/logged-out-visibility$/],
	['PUT', /^\/v1\/preferences\/logged-out-visibility$/],
	['GET', /^\/v1\/chat\/notification\-opens\/[^/]+$/],
	['POST', /^\/v1\/chat\/sync$/],
	['GET', /^\/v1\/chat\/incoming\-policy$/],
	['GET', /^\/v1\/chat\/push\-preferences$/],
	['PUT', /^\/v1\/chat\/push\-preferences$/],
	['PUT', /^\/v1\/chat\/connection$/],
	['GET', /^\/v1\/chat\/conversations$/],
	['GET', /^\/v1\/chat\/availability$/],
	['POST', /^\/v1\/chat\/conversations$/],
	['GET', /^\/v1\/chat\/conversations\/[^/]+$/],
	['GET', /^\/v1\/chat\/conversations\/[^/]+\/details$/],
	['GET', /^\/v1\/chat\/conversations\/[^/]+\/messages$/],
	['DELETE', /^\/v1\/chat\/conversations\/[^/]+\/messages\/[^/]+$/],
	['GET', /^\/v1\/chat\/operations\/[^/]+$/],
	['PUT', /^\/v1\/chat\/conversations\/[^/]+\/read$/],
	['POST', /^\/v1\/chat\/conversations\/[^/]+\/accept$/],
	['PUT', /^\/v1\/chat\/conversations\/[^/]+\/mute$/],
	['DELETE', /^\/v1\/chat\/conversations\/[^/]+\/mute$/],
	['POST', /^\/v1\/chat\/conversations\/[^/]+\/leave$/],
	['POST', /^\/v1\/chat\/reports$/],
	['GET', /^\/v1\/chat\/unread\-counts$/],
	['PUT', /^\/v1\/chat\/incoming\-policy$/],
	['POST', /^\/v1\/chat\/conversations\/[^/]+\/messages$/],
	['GET', /^\/v1\/chat\/status$/],
	['GET', /^\/v1\/moderation\/standing$/],
	['GET', /^\/v1\/moderation\/history$/],
	['GET', /^\/v1\/moderation\/history\/[^/]+$/],
	['GET', /^\/v1\/account\/eligibility$/],
	['GET', /^\/v1\/whoami$/],
	['GET', /^\/v1\/facets\/mentions$/],
	['GET', /^\/v1\/facets\/mentions\/resolve$/],
	['GET', /^\/v1\/facets\/hashtags$/],
	['GET', /^\/v1\/projects$/],
	['GET', /^\/v1\/search\/hashtags\/[^/]+\/posts$/],
	['GET', /^\/v1\/search\/suggestions$/],
	['GET', /^\/v1\/search\/hashtags$/],
	['GET', /^\/v1\/search\/profiles$/],
	['GET', /^\/v1\/search\/posts$/],
	['GET', /^\/v1\/search\/projects$/],
	['GET', /^\/v1\/search\/hashtags\/top$/],
	['GET', /^\/v1\/search\/recent$/],
	['POST', /^\/v1\/search\/recent$/],
	['DELETE', /^\/v1\/search\/recent\/[^/]+$/],
	['GET', /^\/v1\/subscriptions\/access$/],
	['PUT', /^\/v1\/billing\/account$/],
	['GET', /^\/v1\/billing\/account$/],
	['POST', /^\/v1\/billing\/reconciliation$/],
	['POST', /^\/v1\/billing\/web-checkout$/],
	['POST', /^\/v1\/billing\/web-management$/],
	['POST', /^\/v1\/blobs\/videos\/authorization$/],
	['GET', /^\/v1\/blobs\/videos\/limits$/],
	[
		'POST',
		/^\/v1\/blobs\/videos\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\/(?:status|cancel)$/
	],
	['POST', /^\/v1\/migrations\/instagram\/verifications$/],
	['GET', /^\/v1\/migrations\/instagram\/verifications\/current$/],
	['GET', /^\/v1\/migrations\/instagram\/verifications\/[^/]+$/],
	['DELETE', /^\/v1\/migrations\/instagram\/verifications\/[^/]+$/],
	['POST', /^\/v1\/migrations\/instagram\/verifications\/[^/]+\/confirm$/],
	['GET', /^\/v1\/migrations\/instagram\/account$/],
	['DELETE', /^\/v1\/migrations\/instagram\/account$/],
	['PATCH', /^\/v1\/migrations\/instagram\/settings$/],
	['POST', /^\/v1\/migrations\/instagram\/imports$/],
	['GET', /^\/v1\/migrations\/instagram\/imports$/],
	['GET', /^\/v1\/migrations\/instagram\/imports\/[^/]+$/],
	['PATCH', /^\/v1\/migrations\/instagram\/imports\/[^/]+$/],
	['DELETE', /^\/v1\/migrations\/instagram\/imports\/[^/]+$/],
	['GET', /^\/v1\/migrations\/instagram\/suggestions$/],
	['POST', /^\/v1\/migrations\/instagram\/suggestions\/[^/]+\/accept$/],
	['DELETE', /^\/v1\/migrations\/instagram\/suggestions\/[^/]+$/],
	['GET', /^\/v1\/onboarding\/status$/],
	['POST', /^\/v1\/onboarding\/completion$/],
	['GET', /^\/v1\/profiles\/[^/]+$/],
	['GET', /^\/v1\/profiles\/recent$/],
	['GET', /^\/v1\/profiles\/me$/],
	['GET', /^\/v1\/profiles\/me\/follower\-growth$/],
	['GET', /^\/v1\/profiles\/me\/pins$/],
	['GET', /^\/v1\/profiles\/me\/followers$/],
	['GET', /^\/v1\/profiles\/me\/following$/],
	['PUT', /^\/v1\/profiles\/me$/],
	['PUT', /^\/v1\/profiles\/me\/customisation$/],
	['PUT', /^\/v1\/profiles\/me\/business$/],
	['DELETE', /^\/v1\/profiles\/me\/business$/],
	['GET', /^\/v1\/profiles\/[^/]+\/followers$/],
	['GET', /^\/v1\/profiles\/[^/]+\/following$/],
	['GET', /^\/v1\/profiles\/[^/]+\/mutual\-followers$/],
	['GET', /^\/v1\/profiles\/[^/]+\/events$/],
	['POST', /^\/v1\/profiles\/[^/]+\/follows$/],
	['DELETE', /^\/v1\/profiles\/[^/]+\/follows$/],
	['POST', /^\/v1\/profiles\/[^/]+\/mutes$/],
	['DELETE', /^\/v1\/profiles\/[^/]+\/mutes$/],
	['POST', /^\/v1\/profiles\/[^/]+\/blocks$/],
	['DELETE', /^\/v1\/profiles\/[^/]+\/blocks$/],
	['GET', /^\/v1\/profiles\/me\/mutes$/],
	['GET', /^\/v1\/profiles\/me\/blocks$/],
	['POST', /^\/v1\/profile\-links\/[^/]+\/[^/]+\/reports$/],
	['POST', /^\/v1\/profiles\/[^/]+\/reports$/],
	['POST', /^\/v1\/events$/],
	['GET', /^\/v1\/events$/],
	['GET', /^\/v1\/events\/[^/]+\/[^/]+$/],
	['PUT', /^\/v1\/events\/[^/]+\/[^/]+$/],
	['DELETE', /^\/v1\/events\/[^/]+\/[^/]+$/],
	['POST', /^\/v1\/events\/[^/]+\/[^/]+\/reports$/],
	['GET', /^\/v1\/feed\/timeline$/],
	['GET', /^\/v1\/feed\/extended-network$/],
	['GET', /^\/v1\/notifications$/],
	['GET', /^\/v1\/notifications\/new\-count$/],
	['POST', /^\/v1\/notifications\/seen$/],
	['GET', /^\/v1\/notifications\/preferences$/],
	['PATCH', /^\/v1\/notifications\/preferences$/],
	['GET', /^\/v1\/languages\/preferences$/],
	['PUT', /^\/v1\/languages\/preferences$/],
	['POST', /^\/v1\/languages\/preferences\/initialize$/],
	['POST', /^\/v1\/notifications\/devices$/],
	['DELETE', /^\/v1\/notifications\/devices\/[^/]+$/],
	['POST', /^\/v1\/blobs\/images\/authorization$/],
	['POST', /^\/v1\/blobs\/images$/],
	['PUT', /^\/v1\/scheduled\-post\-media\/[^/]+$/],
	['GET', /^\/v1\/scheduled\-post\-media\/[^/]+$/],
	['DELETE', /^\/v1\/scheduled\-post\-media\/[^/]+$/],
	['POST', /^\/v1\/scheduled\-posts$/],
	['GET', /^\/v1\/scheduled\-posts$/],
	['GET', /^\/v1\/scheduled\-posts\/[^/]+$/],
	['PUT', /^\/v1\/scheduled\-posts\/[^/]+$/],
	['DELETE', /^\/v1\/scheduled\-posts\/[^/]+$/],
	['POST', /^\/v1\/scheduled\-posts\/[^/]+\/publication$/],
	['POST', /^\/v1\/posts$/],
	['POST', /^\/v1\/profile\-link\-previews$/],
	['POST', /^\/v1\/link\-previews$/],
	['GET', /^\/v1\/posts\/[^/]+\/[^/]+$/],
	['GET', /^\/v1\/posts\/[^/]+\/[^/]+\/edit$/],
	['PATCH', /^\/v1\/posts\/[^/]+\/[^/]+$/],
	['GET', /^\/v1\/posts\/[^/]+\/[^/]+\/video\-captions\/[^/]+$/],
	['POST', /^\/v1\/posts\/[^/]+\/[^/]+\/saves$/],
	['DELETE', /^\/v1\/posts\/[^/]+\/[^/]+\/saves$/],
	['PUT', /^\/v1\/posts\/[^/]+\/[^/]+\/pin$/],
	['DELETE', /^\/v1\/posts\/[^/]+\/[^/]+\/pin$/],
	['GET', /^\/v1\/saved\-posts$/],
	['GET', /^\/v1\/saved\-post\-folders$/],
	['POST', /^\/v1\/saved\-post\-folders$/],
	['PATCH', /^\/v1\/saved\-post\-folders\/[^/]+$/],
	['DELETE', /^\/v1\/saved\-post\-folders\/[^/]+$/],
	['GET', /^\/v1\/posts\/[^/]+\/[^/]+\/replies$/],
	['GET', /^\/v1\/posts\/[^/]+\/[^/]+\/comments$/],
	['GET', /^\/v1\/posts\/[^/]+\/[^/]+\/likes$/],
	['POST', /^\/v1\/posts\/[^/]+\/[^/]+\/likes$/],
	['DELETE', /^\/v1\/posts\/[^/]+\/[^/]+\/likes$/],
	['GET', /^\/v1\/posts\/[^/]+\/[^/]+\/reposts$/],
	['POST', /^\/v1\/posts\/[^/]+\/[^/]+\/reposts$/],
	['DELETE', /^\/v1\/posts\/[^/]+\/[^/]+\/reposts$/],
	['GET', /^\/v1\/posts\/[^/]+\/[^/]+\/quotes$/],
	['DELETE', /^\/v1\/posts\/[^/]+\/[^/]+$/],
	['POST', /^\/v1\/posts\/[^/]+\/[^/]+\/reports$/],
	['GET', /^\/v1\/profiles\/[^/]+\/posts$/],
	['GET', /^\/v1\/profiles\/[^/]+\/projects$/],
	['GET', /^\/v1\/profiles\/[^/]+\/comments$/],
	['GET', /^\/v1\/profiles\/[^/]+\/reposts$/]
];
export function allowedRoute(method: string, path: string) {
	return routes.some(
		([verb, pattern]) => verb === method && pattern.test(path)
	);
}

/** Params are already URL-decoded by SvelteKit; never decode them again. */
export function safeProxyPath(raw: unknown): string | undefined {
	if (
		typeof raw !== 'string' ||
		raw.length > 4096 ||
		/[\\?#\u0000-\u0020\u007f]/.test(raw)
	)
		return;
	const segments = raw.split('/');
	if (
		segments.some(
			(segment) =>
				!segment ||
				segment === '.' ||
				segment === '..' ||
				(segment.includes('%') && !encodedWebDIDPort(segment))
		)
	)
		return;
	return '/' + segments.map(encodeURIComponent).join('/');
}
// Encoded colons belong to did:web identifiers. Encode each segment exactly once.
function encodedWebDIDPort(segment: string): boolean {
	const match =
		/^@?did:web:([A-Za-z0-9.-]+)%3[aA]([0-9]{1,5})(?::[A-Za-z0-9._~!$&'()*+,;=-]+)*$/.exec(
			segment
		);
	if (
		!match ||
		Number(match[2]) < 1 ||
		Number(match[2]) > 65535 ||
		match[1].length > 253
	)
		return false;
	return match[1]
		.split('.')
		.every((label) =>
			/^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$/.test(label)
		);
}
