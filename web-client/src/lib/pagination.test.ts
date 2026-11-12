import { expect, it } from 'vitest';
import { appendPage, reloadPages } from './pagination';

it('preserves server order and existing objects across overlapping pages without mutating inputs', () => {
	const first = { id: 'a', text: 'original' };
	const current = Object.freeze([first]);
	const next = Object.freeze([
		{ id: 'b', text: 'next' },
		{ id: 'a', text: 'changed' },
		{ id: 'b', text: 'duplicate' }
	]);
	const result = appendPage(current, next, (item) => item.id);
	expect(result).toEqual([first, next[0]]);
	expect(result[0]).toBe(first);
	expect(current).toHaveLength(1);
	expect(next).toHaveLength(3);
	expect(appendPage(current, [], (item) => item.id)).toEqual(current);
});

it('reloads the complete loaded range before resolving and deduplicates overlap', async () => {
	let finish!: (page: { items: string[]; cursor: null }) => void;
	const cursors: (string | null | undefined)[] = [];
	const result = reloadPages(
		async (cursor) => {
			cursors.push(cursor);
			if (!cursor) return { items: ['a', 'b'], cursor: 'second' };
			return new Promise<{ items: string[]; cursor: null }>((resolve) => {
				finish = resolve;
			});
		},
		2,
		(item) => item
	);
	let resolved = false;
	void result.then(() => {
		resolved = true;
	});
	await Promise.resolve();
	await Promise.resolve();
	expect(cursors).toEqual([undefined, 'second']);
	expect(resolved).toBe(false);
	finish({ items: ['b', 'c'], cursor: null });
	expect(await result).toEqual({
		items: ['a', 'b', 'c'],
		cursor: null,
		pages: 2
	});
});

it('stops refresh at the end of the feed or a repeated cursor', async () => {
	expect(
		await reloadPages(
			async () => ({ items: ['a'], cursor: null }),
			3,
			(item) => item
		)
	).toEqual({ items: ['a'], cursor: null, pages: 1 });
	let calls = 0;
	const result = await reloadPages(
		async () => {
			calls++;
			return { items: ['a'], cursor: 'loop' };
		},
		5,
		(item) => item
	);
	expect(calls).toBe(2);
	expect(result.items).toEqual(['a']);
});

it('does not return a partial refreshed feed when a later page fails', async () => {
	await expect(
		reloadPages(
			async (cursor) => {
				if (cursor) throw new Error('offline');
				return { items: ['a'], cursor: 'next' };
			},
			2,
			(item) => item
		)
	).rejects.toThrow('offline');
});
