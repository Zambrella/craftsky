import { describe, expect, it } from 'vitest';
import { draftPreview, draftURL } from './presentation';
import { newDraft } from './store';

describe('native draft row presentation', () => {
	it('prefers a project title and keeps its distinct story as secondary text', () => {
		const draft = {
			...newDraft('did:plc:alice', 'project'),
			text: 'A soft winter knit.',
			project: { common: { title: '  Winter cardigan  ' } }
		};
		expect(draftPreview(draft)).toEqual({
			title: 'Winter cardigan',
			body: 'A soft winter knit.',
			kind: 'Project'
		});
	});
	it('truncates previews at Unicode code points without breaking emoji', () => {
		const result = draftPreview({
			...newDraft('did:plc:alice'),
			text: '🧶'.repeat(121)
		});
		expect(result.title).toBe('🧶'.repeat(119) + '…');
	});
	it('keeps unresolved operations visible and encodes draft references', () => {
		const draft = {
			...newDraft('did:plc:alice'),
			id: 'draft&other=one',
			pending: { key: 'frozen', body: {} }
		};
		expect(draftPreview(draft).kind).toBe('Awaiting confirmation');
		expect(draftURL(draft)).toBe('/compose?draft=draft%26other%3Done');
	});
});
