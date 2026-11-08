import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
const { mount, goto } = vi.hoisted(() => ({
	mount: vi.fn(() => ({})),
	goto: vi.fn()
}));
vi.mock('$app/navigation', () => ({ goto }));
vi.mock('svelte', () => ({ mount, unmount: vi.fn(), onDestroy: vi.fn() }));
vi.mock('./PromptDialog.svelte', () => ({ default: {} }));
import { confirmExternalLink } from './prompt';

class Anchor {
	target = '_blank';
	constructor(public href: string) {}
	removeAttribute(name: string) {
		if (name === 'target') this.target = '';
	}
}
function click(href: string, modifiers = {}) {
	const anchor = new Anchor(href);
	const event = {
		currentTarget: anchor,
		preventDefault: vi.fn(),
		...modifiers
	};
	confirmExternalLink(event as unknown as MouseEvent);
	return { anchor, event };
}

beforeEach(() => {
	vi.clearAllMocks();
	vi.stubGlobal('HTMLAnchorElement', Anchor);
	vi.stubGlobal('location', { origin: 'http://localhost:5173' });
	vi.stubGlobal('document', { body: {} });
});
afterEach(() => vi.unstubAllGlobals());

describe('link navigation', () => {
	it('uses a local CraftSky route without confirmation, even on a preview host', () => {
		const { anchor, event } = click(
			'https://app.craftsky.social/posts/alice/post?focus=reply#top'
		);
		expect(anchor.href).toBe('/posts/alice/post?focus=reply#top');
		expect(anchor.target).toBe('');
		expect(event.preventDefault).toHaveBeenCalledOnce();
		expect(goto).toHaveBeenCalledExactlyOnceWith(
			'/posts/alice/post?focus=reply#top'
		);
		expect(mount).not.toHaveBeenCalled();
	});
	it('preserves native modifier clicks and encoded paths', () => {
		const { anchor, event } = click(
			'https://app.craftsky.social/profiles/a%20b',
			{ ctrlKey: true }
		);
		expect(anchor.href).toBe('/profiles/a%20b');
		expect(goto).not.toHaveBeenCalled();
		expect(event.preventDefault).not.toHaveBeenCalled();
		expect(mount).not.toHaveBeenCalled();
	});
	it('allows same-origin rich-text links without confirmation', () => {
		const { event } = click('http://localhost:5173/feed');
		expect(event.preventDefault).not.toHaveBeenCalled();
		expect(mount).not.toHaveBeenCalled();
	});
	it.each([
		'https://example.com/shop',
		'https://craftsky.social/terms',
		'https://app.craftsky.social.evil.example/feed',
		'https://user@app.craftsky.social/feed',
		'https://app.craftsky.social:8443/feed',
		'https://app.craftsky.social//evil.example',
		'mailto:hello@example.com'
	])('still confirms external destination %s', (href) => {
		const { anchor, event } = click(href);
		expect(anchor.href).toBe(href);
		expect(event.preventDefault).toHaveBeenCalledOnce();
		expect(mount).toHaveBeenCalledWith(
			expect.anything(),
			expect.objectContaining({
				props: expect.objectContaining({ href, target: '_blank' })
			})
		);
	});
});
