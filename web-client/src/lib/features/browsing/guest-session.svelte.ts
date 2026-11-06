const exploredKey = 'craftsky.guest.explored';
type TabStorage = Pick<Storage, 'getItem' | 'setItem'>;

/** Guest exploration lasts for this tab. Storage failure falls back to this
 * runtime's lifetime; it never prevents viewing public content. */
export class GuestSession {
	explored = $state(false);
	promptOpen = $state(false);
	constructor(private readonly storage?: TabStorage) {
		try {
			this.explored = storage?.getItem(exploredKey) === '1';
		} catch {}
	}
	enter(path: string) {
		if (!this.explored && /^\/(?:feed(?:\/filters)?)?\/?$/.test(path))
			this.promptOpen = true;
	}
	signIn() {
		this.promptOpen = true;
	}
	explore() {
		this.explored = true;
		this.promptOpen = false;
		try {
			this.storage?.setItem(exploredKey, '1');
		} catch {}
	}
}
