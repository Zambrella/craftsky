/** Recognize the account's profile for both canonical DID links and current handle links. */
export function ownProfileRoute(
	pathname: string,
	account: { did: string; handle: string },
	currentHandle = account.handle
): boolean {
	const match = /^\/app\/profile\/([^/]+)$/.exec(pathname);
	if (!match) return false;
	try {
		const address = decodeURIComponent(match[1]);
		return (
			address === account.did ||
			address.toLowerCase() === currentHandle.toLowerCase() ||
			address.toLowerCase() === account.handle.toLowerCase()
		);
	} catch {
		return false;
	}
}
