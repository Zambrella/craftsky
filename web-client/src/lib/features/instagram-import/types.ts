export type ExportVariant = 'legacy' | 'detailed';
export type SourceMediaKind = 'image' | 'video' | 'unsupported';

export interface AdapterMedia {
	readonly path: string;
	readonly kind: SourceMediaKind;
}

export interface AdapterPost {
	readonly variant: ExportVariant;
	readonly timestamp: number;
	readonly caption: string;
	readonly media: readonly AdapterMedia[];
}

export interface AdapterSkipOutcome {
	readonly code: 'invalidTimestamp' | 'unpublished' | 'unsupportedPost';
	readonly ordinal: number;
	readonly timestamp?: number;
}

export interface AdapterParseResult {
	readonly posts: readonly AdapterPost[];
	readonly skipped: readonly AdapterSkipOutcome[];
	readonly recognizedPostRecords: number;
}

export interface NormalizedAdapterResult {
	readonly posts: readonly AdapterPost[];
	readonly skipped: readonly {
		readonly code: 'ambiguousDuplicate';
		readonly timestamp: number;
	}[];
}

export interface InstagramPost {
	readonly sourceKey: string;
	readonly createdAt: string;
	readonly caption: string;
	readonly imageCount: number;
	readonly videoCount: number;
	readonly warnings: readonly string[];
}
export interface MediaSelection {
	readonly kind: 'images' | 'video';
	readonly indices?: readonly number[];
}
export interface InstagramMedia {
	images: File[];
	videos: File[];
}
export interface InstagramArchive {
	readonly posts: readonly InstagramPost[];
	readonly skipped: number;
	media(
		post: InstagramPost,
		signal?: AbortSignal,
		selection?: MediaSelection
	): Promise<InstagramMedia>;
	preview(post: InstagramPost, signal?: AbortSignal): Promise<Blob | undefined>;
	close(): void;
}
