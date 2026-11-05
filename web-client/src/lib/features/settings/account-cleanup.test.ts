import { it, expect } from 'vitest';
import { clearAccountChat } from './account-cleanup.ts';
it('deletion removes only the captured activation chat data and preserves other accounts and device preferences', () => {
	const own =
			'craftsky.chat.v1:' + encodeURIComponent('alice-id:did:plc:alice') + ':',
		other =
			'craftsky.chat.v1:' + encodeURIComponent('bob-id:did:plc:bob') + ':';
	const values = new Map([
		[own + 'draft', 'private'],
		[own + 'receipt', 'pending'],
		[other + 'draft', 'other-private'],
		['craftsky.theme', 'dark'],
		['craftsky.diagnostics-consent', 'true']
	]);
	clearAccountChat(
		{
			get length() {
				return values.size;
			},
			key: (index) => [...values.keys()][index] ?? null,
			removeItem: (key) => {
				values.delete(key);
			}
		},
		'alice-id',
		'did:plc:alice'
	);
	expect([...values.keys()]).toEqual([
		other + 'draft',
		'craftsky.theme',
		'craftsky.diagnostics-consent'
	]);
});
