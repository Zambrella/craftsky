import { describe, expect, it } from 'vitest';
import { excerpt, graphemeBoundary, graphemeCount } from './text';

describe('shared Unicode text handling', () => {
	it('counts and truncates whole emoji, flags and combining characters', () => {
		const clusters = ['a', '👨‍👩‍👧‍👦', '🇬🇧', 'é'];
		const text = clusters.join('');
		expect(graphemeCount(text)).toBe(4);
		for (let limit = 0; limit <= 4; limit++) {
			const prefix = clusters.slice(0, limit).join('');
			expect(graphemeBoundary(text, limit)).toBe(prefix.length);
			expect(excerpt(text, limit)).toBe(prefix + (limit < 4 ? '…' : ''));
		}
	});
	it('leaves empty and shorter inputs untouched', () => {
		expect(graphemeCount('')).toBe(0);
		expect(graphemeBoundary('', 0)).toBe(0);
		expect(excerpt('', 0)).toBe('');
		expect(excerpt('hello', 160)).toBe('hello');
	});
});
