import { describe, expect, it } from 'vitest';
import {
	historyScrollMode,
	inboxTime,
	messageBackURL,
	startsTimeSection,
	timeSectionLabel
} from './presentation';
describe('mobile message presentation', () => {
	it('starts sections for the first message, local date changes and five-minute gaps', () => {
		expect(startsTimeSection('2026-11-02T10:00:00')).toBe(true);
		expect(
			startsTimeSection('2026-11-02T10:04:59', '2026-11-02T10:00:00')
		).toBe(false);
		expect(
			startsTimeSection('2026-11-02T10:05:00', '2026-11-02T10:00:00')
		).toBe(true);
		expect(
			startsTimeSection('2026-11-03T00:01:00', '2026-11-02T23:59:00')
		).toBe(true);
		expect(startsTimeSection('invalid')).toBe(false);
	});
	it('uses relative inbox times and understandable authoritative section labels', () => {
		const now = new Date('2026-11-02T12:00:00');
		expect(inboxTime('2026-11-02T11:58:00', now)).toBe('2m');
		expect(inboxTime('2026-11-01T10:00:00', now)).toBe('1d');
		expect(timeSectionLabel('2026-11-02T10:00:00', now, 'en-GB')).toBe(
			'Today at 10:00'
		);
		expect(timeSectionLabel('2026-11-01T10:00:00', now, 'en-GB')).toBe(
			'Yesterday at 10:00'
		);
	});
	it('keeps older-history position and never drags a reader back down on new messages', () => {
		const previous = { id: 'one', first: 'b', last: 'd' };
		expect(
			historyScrollMode(previous, { ...previous, first: 'a' }, false)
		).toBe('preserve');
		expect(historyScrollMode(previous, { ...previous, last: 'e' }, false)).toBe(
			'none'
		);
		expect(historyScrollMode(previous, { ...previous, last: 'e' }, true)).toBe(
			'end'
		);
		expect(
			historyScrollMode(previous, { id: 'two', first: 'a', last: 'z' }, false)
		).toBe('end');
	});
	it('returns from a conversation to the inbox and from its details to the conversation', () => {
		expect(messageBackURL('conversation', 'one')).toBe('/app/messages');
		expect(messageBackURL('details', 'one/two')).toBe(
			'/app/messages/one%2Ftwo'
		);
		expect(messageBackURL('inbox')).toBe('/app/messages');
	});
});
