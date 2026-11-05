import { describe, expect, it, vi } from 'vitest';
import { ToastHub, type Toast } from './toasts.svelte.ts';
function toast(message: string, warning = false): Toast {
	return { source: Symbol(message), message, warning, onDismiss: vi.fn() };
}
describe('action toast queue', () => {
	it('updates a source in place and deduplicates feedback rendered twice', () => {
		const hub = new ToastHub(),
			first = toast('Saved.');
		hub.show(first);
		hub.show(toast('Saved.'));
		expect(hub.items).toHaveLength(1);
		hub.show({ ...first, message: 'Updated.' });
		expect(hub.items.map((item) => item.message)).toEqual(['Updated.']);
	});
	it('prioritises action failures ahead of confirmations', () => {
		const hub = new ToastHub(),
			success = toast('Saved.'),
			error = toast('Try again.', true);
		hub.show(success);
		hub.show(error);
		expect(hub.items[0].source).toBe(error.source);
		hub.dismiss(error.source);
		expect(error.onDismiss).toHaveBeenCalledOnce();
		expect(hub.items[0].source).toBe(success.source);
	});
	it('bounds pending confirmations while retaining the visible one', () => {
		const hub = new ToastHub();
		for (let i = 0; i < 10; i++) hub.show(toast(String(i)));
		expect(hub.items.map((item) => item.message)).toEqual([
			'0',
			'6',
			'7',
			'8',
			'9'
		]);
	});
	it('keeps same-account navigation feedback and clears it on account changes', () => {
		const hub = new ToastHub();
		hub.scope('alice');
		hub.show(toast('Private account action.'));
		hub.scope('alice');
		expect(hub.items).toHaveLength(1);
		hub.scope('bob');
		expect(hub.items).toHaveLength(0);
	});
	it('removes an unmounted recovery control without invoking its action', () => {
		const hub = new ToastHub(),
			error = toast('Try again.', true);
		hub.show(error);
		hub.remove(error.source);
		hub.dismiss(error.source);
		expect(error.onDismiss).not.toHaveBeenCalled();
		expect(hub.items).toHaveLength(0);
	});
});
