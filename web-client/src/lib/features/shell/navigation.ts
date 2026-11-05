/** Recognize the account's profile for both canonical DID links and current handle links. */
export function ownProfileRoute(
	pathname: string,
	account: { did: string; handle: string },
	currentHandle = account.handle
): boolean {
	if (pathname === '/profile') return true;
	const match = /^\/profiles\/([^/]+)$/.exec(pathname);
	if (!match) return false;
	try {
		const address = decodeURIComponent(match[1]);
		return (
			address === account.did ||
			address.replace(/^@/, '').toLowerCase() === currentHandle.toLowerCase() ||
			address.replace(/^@/, '').toLowerCase() === account.handle.toLowerCase()
		);
	} catch {
		return false;
	}
}
