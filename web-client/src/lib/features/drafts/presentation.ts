import type { Draft } from './store';

export function draftPreview(draft: Draft): {
	title: string;
	body?: string;
	kind: string;
} {
	const common = draft.project?.common;
	const projectTitle =
		common &&
		typeof common === 'object' &&
		!Array.isArray(common) &&
		typeof common.title === 'string'
			? common.title.trim()
			: '';
	const body = draft.text.trim();
	const title = projectTitle || body || 'Untitled draft';
	const points = Array.from(title);
	return {
		title: points.length > 120 ? points.slice(0, 119).join('') + '…' : title,
		...(projectTitle && body && projectTitle !== body ? { body } : {}),
		kind: draft.pending
			? 'Awaiting confirmation'
			: draft.edit
				? 'Post edit'
				: draft.kind === 'project'
					? 'Project'
					: 'Standard'
	};
}

export function draftURL(draft: Pick<Draft, 'id'>): string {
	return '/compose?draft=' + encodeURIComponent(draft.id);
}
