/** Browser paths shared with the mobile router; composer/editor paths are web pages for mobile modals. */
export function authenticatedRoute(pathname: string): boolean {
	return /^(?:\/feed(?:\/filters)?|\/projects|\/search(?:\/tags)?|\/notifications(?:\/settings)?|\/messages(?:\/(?:new|requests)|\/conversations\/[^/]+(?:\/details|\/report(?:\/[^/]+)?)?)?|\/profile(?:\/edit)?|\/profiles\/[^/]+(?:\/mutual-followers)?|\/posts\/[^/]+\/[^/]+(?:\/(?:likes|reposts|quotes))?|\/events\/[^/]+\/[^/]+|\/saved(?:\/folder)?|\/scheduled|\/drafts|\/settings(?:\/[^/]+)*|\/compose(?:\/[^/]+)*)$/.test(
		pathname.replace(/\/$/, '')
	);
}

/** Public reading destinations; all other app routes retain authentication. */
export function guestRoute(pathname: string): boolean {
	return /^(?:\/feed(?:\/filters)?|\/projects|\/search(?:\/tags)?|\/profiles\/(?!me(?:\/|$)|recent(?:\/|$))[^/]+|\/posts\/[^/]+\/[^/]+(?:\/(?:likes|reposts|quotes))?|\/events\/[^/]+\/[^/]+)$/.test(
		pathname.replace(/\/$/, '')
	);
}
