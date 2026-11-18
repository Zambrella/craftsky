import { afterEach, describe, expect, it, vi } from 'vitest';
import {
	clearInstagramPostSession,
	importedPostPath,
	useInstagramPostSession
} from './instagram-post-session';
import type { InstagramArchive } from '../instagram-import';

afterEach(() => {
	clearInstagramPostSession();
	vi.restoreAllMocks();
});
describe('Instagram export session', () => {
	it('retains its archive for a return from the composer and closes it on an account change', () => {
		const close = vi.fn();
		const session = useInstagramPostSession('session-one:did:plc:alice');
		session.archive = { close } as unknown as InstagramArchive;
		session.fileName = 'export.zip';
		expect(useInstagramPostSession('session-one:did:plc:alice')).toBe(session);
		expect(close).not.toHaveBeenCalled();
		const next = useInstagramPostSession('session-two:did:plc:bob');
		expect(close).toHaveBeenCalledOnce();
		expect(next.archive).toBeUndefined();
		expect(next.fileName).toBe('');
	});
	it('revokes thumbnails and releases local ZIP resources on explicit clearing', () => {
		const revoke = vi.spyOn(URL, 'revokeObjectURL');
		const close = vi.fn();
		const session = useInstagramPostSession('alice');
		session.archive = { close } as unknown as InstagramArchive;
		session.previews = { source: 'blob:local-thumbnail' };
		clearInstagramPostSession();
		expect(close).toHaveBeenCalledOnce();
		expect(revoke).toHaveBeenCalledWith('blob:local-thumbnail');
		expect(useInstagramPostSession('alice').previews).toEqual({});
	});
	it('builds post links only for CraftSky post records', () => {
		expect(
			importedPostPath('at://did:plc:alice/social.craftsky.feed.post/abc')
		).toBe('/posts/did%3Aplc%3Aalice/abc');
		expect(importedPostPath('https://example.org/tracking')).toBeUndefined();
		expect(
			importedPostPath('at://did:plc:alice/app.bsky.feed.post/abc')
		).toBeUndefined();
	});
});
