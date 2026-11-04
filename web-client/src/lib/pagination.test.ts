import { expect, it } from 'vitest';
import { appendPage } from './pagination';

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
