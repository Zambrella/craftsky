import type { Draft, DraftMedia } from '../drafts/store';
export interface PreviewCandidate {
	identity: string;
	fragment: string;
}
export interface ComposerPreview {
	external: NonNullable<Draft['external']>;
	image?: DraftMedia;
	sourceIdentity?: string;
	resolvedURI?: string;
}
export interface PreviewState {
	selected?: ComposerPreview;
	index: number;
	total: number;
	loading: boolean;
	dismissed: boolean;
	suppressed: boolean;
}
const linkPattern =
	/(^|[\s([{])((?:https?:\/\/)?(?:[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?\.)+[A-Za-z]{2,}(?::[0-9]+)?(?:\/[^\s]*)?)/gi;
/** Flutter facet completion rules avoid fetching a URL while it is still being typed. */
export function previewCandidates(
	text: string,
	retainedIdentity?: string
): PreviewCandidate[] {
	const result: PreviewCandidate[] = [],
		seen = new Set<string>();
	for (const match of text.matchAll(linkPattern)) {
		let visible = match[2].replace(/[.,!?;:]+$/g, '');
		for (const [closing, opening] of [
			[')', '('],
			[']', '['],
			['}', '{']
		])
			while (
				visible.endsWith(closing) &&
				visible.split(closing).length > visible.split(opening).length
			)
				visible = visible.slice(0, -1);
		try {
			const url = new URL(
				/^https?:\/\//i.test(visible) ? visible : 'https://' + visible
			);
			if (
				!['http:', 'https:'].includes(url.protocol) ||
				!url.hostname ||
				url.username ||
				url.password
			)
				continue;
			const fragment = url.hash;
			url.hash = '';
			const identity = url.href;
			const end = match.index! + match[1].length + visible.length;
			if (
				identity !== retainedIdentity &&
				(end >= text.length || !/[\s.,!?;:)\]}]/.test(text[end]))
			)
				continue;
			if (seen.has(identity)) continue;
			seen.add(identity);
			result.push({ identity, fragment });
			if (result.length === 4) break;
		} catch {
			/* Unsupported tokens remain ordinary prose. */
		}
	}
	return result;
}
export function previewNavigation(
	candidate: PreviewCandidate,
	finalURL: string
): string {
	const url = new URL(finalURL);
	if (
		!['http:', 'https:'].includes(url.protocol) ||
		url.username ||
		url.password
	)
		throw new Error('Invalid link preview URL');
	if (!url.hash && candidate.fragment) url.hash = candidate.fragment;
	return url.href;
}
/** Per editor activation: bounded candidates/cache, cancellation, debounce, and no browser URL fetches. */
export class ComposerLinkPreviews {
	private text = '';
	private seedText?: string;
	private seedURI?: string;
	private retained?: string;
	private candidates: PreviewCandidate[] = [];
	private successes = new Map<string, ComposerPreview>();
	private failures = new Set<string>();
	private selected?: string;
	private active?: AbortController;
	private timer?: ReturnType<typeof setTimeout>;
	private generation = 0;
	private disposed = false;
	private stopped = false;
	private suppressed = false;
	private dismissed = false;
	constructor(
		private fetcher: (
			candidate: PreviewCandidate,
			signal: AbortSignal
		) => Promise<ComposerPreview>,
		private publish: (state: PreviewState) => void,
		private debounce = 300
	) {}
	seed(text: string, preview: ComposerPreview) {
		try {
			const url = new URL(preview.sourceIdentity || preview.external.uri);
			if (
				!['http:', 'https:'].includes(url.protocol) ||
				url.username ||
				url.password
			)
				return;
			url.hash = '';
			this.retained = url.href;
			this.seedText = text;
			this.seedURI = preview.external.uri;
			this.successes.set(url.href, {
				...preview,
				external: {
					...preview.external,
					uri: preview.resolvedURI || preview.external.uri
				}
			});
			this.selected = url.href;
		} catch {}
	}
	update(text: string, suppressed: boolean, dismissed: boolean) {
		if (this.disposed) return;
		const changed = text !== this.text;
		this.text = text;
		if (changed) this.stopped = false;
		this.suppressed = suppressed;
		this.dismissed = dismissed;
		this.candidates = previewCandidates(text, this.retained);
		if (!this.candidates.length && this.seedText === text && this.retained)
			this.candidates = [{ identity: this.retained, fragment: '' }];
		if (
			this.selected &&
			!this.candidates.some((value) => value.identity === this.selected)
		)
			this.selected = undefined;
		this.cancel();
		this.selectEarliest();
		this.emit();
		this.schedule();
	}
	move(delta: number) {
		const available = this.available();
		if (!available.length) return;
		const index = available.findIndex(
			(value) => value.identity === this.selected
		);
		this.selected =
			available[
				(Math.max(0, index) + delta + available.length) % available.length
			].identity;
		this.emit();
	}
	pause() {
		this.stopped = true;
		this.cancel();
		this.emit();
	}
	dispose() {
		this.disposed = true;
		this.cancel();
		this.successes.clear();
		this.failures.clear();
		this.candidates = [];
	}
	private cancel() {
		this.generation++;
		this.active?.abort();
		this.active = undefined;
		if (this.timer) clearTimeout(this.timer);
		this.timer = undefined;
	}
	private available() {
		return this.candidates.filter((value) =>
			this.successes.has(value.identity)
		);
	}
	private selectEarliest() {
		if (!this.selected) this.selected = this.available()[0]?.identity;
	}
	private emit() {
		if (this.disposed) return;
		const available = this.available(),
			candidate = this.candidates.find(
				(value) => value.identity === this.selected
			),
			stored = candidate && this.successes.get(candidate.identity);
		const selected =
			stored && candidate && !this.suppressed && !this.dismissed
				? {
						...stored,
						sourceIdentity: candidate.identity,
						resolvedURI: stored.external.uri,
						external: {
							...stored.external,
							uri:
								this.seedText === this.text &&
								candidate.identity === this.retained &&
								this.seedURI
									? this.seedURI
									: previewNavigation(candidate, stored.external.uri)
						}
					}
				: undefined;
		this.publish({
			selected,
			index: selected
				? available.findIndex((value) => value.identity === this.selected) + 1
				: 0,
			total: available.length,
			loading: !!this.active,
			dismissed: this.dismissed,
			suppressed: this.suppressed
		});
	}
	private schedule() {
		if (
			this.disposed ||
			this.stopped ||
			this.suppressed ||
			this.dismissed ||
			this.active ||
			this.timer
		)
			return;
		const candidate = this.candidates.find(
			(value) =>
				!this.successes.has(value.identity) &&
				!this.failures.has(value.identity)
		);
		if (!candidate) return;
		const revision = this.generation;
		this.timer = setTimeout(() => {
			this.timer = undefined;
			if (this.disposed || revision !== this.generation) return;
			const request = new AbortController();
			this.active = request;
			this.emit();
			void this.fetcher(candidate, request.signal)
				.then((value) => {
					if (
						this.disposed ||
						request.signal.aborted ||
						revision !== this.generation
					)
						return;
					this.successes.set(candidate.identity, value);
				})
				.catch(() => {
					if (
						!this.disposed &&
						!request.signal.aborted &&
						revision === this.generation
					)
						this.failures.add(candidate.identity);
				})
				.finally(() => {
					if (
						this.disposed ||
						request.signal.aborted ||
						revision !== this.generation
					)
						return;
					this.active = undefined;
					this.selectEarliest();
					this.emit();
					this.schedule();
				});
		}, this.debounce);
	}
}
