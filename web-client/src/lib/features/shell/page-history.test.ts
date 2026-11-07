import { describe, expect, it, vi } from 'vitest';
import { PageHistory } from './page-history';

vi.mock('$app/state', () => ({ page: { state: {} } }));

describe('feed history after policy changes', () => {
	it('rejects an obsolete load but retains refreshed pages from the mounted view', () => {
		const history = new PageHistory();
		const view = history.view<string[]>('feed');
		view.save(['old page']);
		const obsolete = view.beginLoad();
		history.invalidate('feed');
		expect(obsolete.isCurrent()).toBe(false);
		obsolete.save(['stale response']);
		view.save(['stale unmount']);
		expect(history.view('feed').value).toBeUndefined();

		const refreshed = view.beginLoad();
		refreshed.save(['refreshed page']);
		view.save(['refreshed page', 'next page']);
		expect(history.view('feed').value).toEqual(['refreshed page', 'next page']);
	});

	it('notifies active feed views and detaches their policy listener', () => {
		const history = new PageHistory();
		const view = history.view('feed');
		const refresh = vi.fn();
		const unsubscribe = view.onInvalidate(refresh);
		history.invalidate('feed');
		expect(refresh).toHaveBeenCalledTimes(1);
		unsubscribe();
		history.invalidate('feed');
		expect(refresh).toHaveBeenCalledTimes(1);
	});
});
