export const signInContext = 'craftsky-request-sign-in';
export type RequestSignIn = (trigger?: HTMLElement) => void;
/** Keep member controls intact; guests may read public links and copy URLs. */
export function protectAccountAction(
	guest: boolean,
	request: RequestSignIn | undefined,
	event: MouseEvent
) {
	if (!guest || !(event.target instanceof Element)) return;
	const target = event.target.closest<HTMLElement>('button,a,input');
	if (!target || target.matches('[data-public-action],.menu-backdrop')) return;
	if (target instanceof HTMLAnchorElement) {
		const url = new URL(target.href);
		// Comment navigation is a read. Likes/reposts, replies and other account actions are gated.
		if (
			/^\/posts\/[^/]+\/[^/]+$/.test(url.pathname) &&
			url.searchParams.has('show-comments')
		)
			return;
	}
	event.preventDefault();
	event.stopPropagation();
	const menu = target.closest('details');
	const trigger = menu?.querySelector<HTMLElement>('summary') ?? target;
	if (menu) menu.open = false;
	request?.(trigger);
}
