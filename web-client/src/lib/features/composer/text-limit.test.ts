import { describe, expect, it } from 'vitest';
import { limitText } from './text-limit';

describe('native record text limits', () => {
	it('counts whole family emoji and combining marks as one character', () => {
		expect(limitText('👨‍👩‍👧‍👦e\u0301z', { graphemes: 2, bytes: 100 })).toBe(
			'👨‍👩‍👧‍👦e\u0301'
		);
	});
	it('honours the independent byte budget without splitting a cluster', () => {
		expect(limitText('éé👩🏽‍💻tail', { graphemes: 200, bytes: 10 })).toBe('éé');
		expect(limitText('🧶'.repeat(200), { graphemes: 200, bytes: 500 })).toBe(
			'🧶'.repeat(125)
		);
	});
	it('preserves exact-limit and empty values', () => {
		expect(limitText('e\u0301', { graphemes: 1, bytes: 3 })).toBe('e\u0301');
		expect(limitText('', { graphemes: 1, bytes: 3 })).toBe('');
	});
});
