import { describe, expect, it } from 'vitest';
import { contextMenu } from './context-menu';

function fixture() {
	const document = new EventTarget(),
		focus = { called: false };
	const node = Object.assign(new EventTarget(), {
		open: true,
		ownerDocument: document,
		contains: () => false,
		querySelector: () => ({
			focus: () => {
				focus.called = true;
			}
		})
	});
	const action = contextMenu(node as unknown as HTMLDetailsElement);
	return { document, node, focus, action };
}
describe('social context menu interactions', () => {
	it('dismisses with Escape and restores keyboard focus without closing after cleanup', () => {
		const { document, node, focus, action } = fixture();
		document.dispatchEvent(
			Object.assign(new Event('keydown', { cancelable: true }), {
				key: 'Escape'
			})
		);
		expect(node.open).toBe(false);
		expect(focus.called).toBe(true);
		action.destroy();
		node.open = true;
		document.dispatchEvent(
			Object.assign(new Event('keydown'), { key: 'Escape' })
		);
		expect(node.open).toBe(true);
	});
	it('closes on outside interaction without preventing the destination action', () => {
		const { document, node, action } = fixture(),
			click = new Event('pointerdown', { cancelable: true });
		expect(document.dispatchEvent(click)).toBe(true);
		expect(node.open).toBe(false);
		expect(click.defaultPrevented).toBe(false);
		action.destroy();
	});
});
