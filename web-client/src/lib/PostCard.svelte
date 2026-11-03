<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { Facet } from './api/types';
	import Avatar from './Avatar.svelte';
	import CraftBadge from './CraftBadge.svelte';
	import ImageCarousel from './ImageCarousel.svelte';
	import Icon from './features/shell/Icon.svelte';
	import {
		authorName,
		gaugeText,
		compactDate,
		fullTimestamp,
		label,
		postPath,
		profilePath,
		safeURL,
		postTitle,
		type Post
	} from './public';
	let {
		post,
		detail = false,
		content,
		media,
		field,
		member = false,
		onImageChange,
		initialImageIndex = 0
	}: {
		post: Post;
		detail?: boolean;
		content?: Snippet<[string]>;
		media?: Snippet;
		field?: Snippet<[string, Facet[] | undefined]>;
		member?: boolean;
		onImageChange?: (index: number) => void;
		initialImageIndex?: number;
	} = $props();
	const editDescriptionId = $props.id();
	const profileHref = (did: string) =>
		member ? '/app/profile/' + encodeURIComponent(did) : profilePath(did);
	const postHref = () =>
		member
			? '/app/posts/' +
				encodeURIComponent(post.author.did) +
				'/' +
				encodeURIComponent(post.rkey)
			: postPath(post);
	const authorCrafts = $derived.by(() => {
		const entries = new Map<string, string>();
		for (const value of post.author.crafts || []) {
			const id = value.split('#').at(-1)?.trim().toLowerCase() || '';
			if (id) entries.set(id, label(id));
		}
		return [...entries].map(([id, name]) => ({
			id,
			name,
			icon: [
				'knitting',
				'crochet',
				'sewing',
				'embroidery',
				'quilting'
			].includes(id)
		}));
	});
	let visibleCraftCount = $state(5);
	const pattern = $derived(post.project?.common.pattern);
	const hasPattern = $derived(
		!!pattern &&
			!!(
				pattern.name?.trim() ||
				pattern.designer?.trim() ||
				pattern.publisher?.trim() ||
				pattern.selfDrafted
			)
	);
	const sizeSummary = $derived.by(() => {
		const craft = post.project?.common.craftType.split('#').at(-1);
		const details = post.project?.details;
		const key =
			craft === 'sewing'
				? 'sizeMade'
				: craft === 'knitting' || craft === 'crochet'
					? 'finishedSize'
					: craft === 'quilting'
						? 'size'
						: undefined;
		const value = key && details?.[key];
		return typeof value === 'string' && value.trim()
			? {
					label: key === 'finishedSize' ? 'Finished size' : 'Size',
					value: value.trim()
				}
			: undefined;
	});
	function fitCrafts(node: HTMLElement, entries: typeof authorCrafts) {
		let disposed = false;
		const canvas = document.createElement('canvas');
		const context = canvas.getContext('2d');
		function fit() {
			if (disposed) return;
			const name = node.querySelector('strong');
			const measure = node.querySelector<HTMLElement>('.craft-count-measure');
			if (!name || !measure) return;
			if (context) context.font = getComputedStyle(measure).font;
			const countWidth = (count: number) =>
				context?.measureText('+' + count).width ||
				measure.getBoundingClientRect().width;
			const budget = Math.min(
				node.clientWidth,
				Math.max(
					node.clientWidth - name.scrollWidth,
					8 + countWidth(entries.length)
				)
			);
			let visible = entries.filter((entry) => entry.icon).length;
			while (visible > 0) {
				const hidden = entries.length - visible;
				const width =
					8 +
					visible * 16 +
					(visible - 1) * 4 +
					(hidden > 0 ? 4 + countWidth(hidden) : 0);
				if (width <= budget) break;
				visible--;
			}
			visibleCraftCount = visible;
		}
		const observer = new ResizeObserver(fit);
		observer.observe(node);
		void document.fonts.ready.then(fit);
		fit();
		return {
			update(next: typeof authorCrafts) {
				entries = next;
				fit();
			},
			destroy() {
				disposed = true;
				observer.disconnect();
			}
		};
	}
	const detailFields: Record<string, string> = {
		projectType: 'Project type',
		size: 'Size',
		yarnWeight: 'Yarn weight',
		needleSizeMm: 'Needle size',
		hookSizeMm: 'Hook size',
		finishedSize: 'Finished size',
		sizeMade: 'Size made',
		fitNotes: 'Fit notes',
		projectSubtype: 'Project subtype',
		gauge: 'Gauge',
		piecingTechnique: 'Piecing technique',
		quiltingMethod: 'Quilting method',
		fabric: 'Fabric',
		techniques: 'Techniques',
		modifications: 'Modifications'
	};
</script>

<article
	class="post-card"
	class:project-card={!!post.project}
	class:post-detail={detail}
>
	<div class="author">
		<a href={profileHref(post.author.did)}
			><Avatar author={post.author} /><span
				>{#if member}<span
						class="member-author-line"
						use:fitCrafts={authorCrafts}
						><strong>{authorName(post.author)}</strong
						>{#if authorCrafts.length}<span
								class="author-crafts"
								role="img"
								aria-label={'Crafts: ' +
									authorCrafts.map((entry) => entry.name).join(', ')}
								title={authorCrafts.map((entry) => entry.name).join(', ')}
								>{#each authorCrafts
									.filter((entry) => entry.icon)
									.slice(0, visibleCraftCount) as entry}<span
										class="author-craft-icon"
										data-craft={entry.id}
										style:--craft-icon={`url('/craft_icons/${entry.id}.svg')`}
										aria-hidden="true"

									></span>{/each}{#if authorCrafts.length > visibleCraftCount}<span
										class="craft-count"
										aria-hidden="true"
										>+{authorCrafts.length - visibleCraftCount}</span
									>{/if}</span
							>{/if}<span class="craft-count-measure" aria-hidden="true"
							>+{authorCrafts.length}</span
						></span
					>{:else}<strong>{authorName(post.author)}</strong
					>{/if}{#if post.author.business}<span class="business-label"
						>Business account</span
					>{/if}<small>@{post.author.handle}</small></span
			></a
		><a
			class="date"
			href={postHref()}
			aria-label={member ? 'Open post' : undefined}
			aria-describedby={post.editedAt ? editDescriptionId : undefined}
			title={fullTimestamp(post.createdAt) +
				(post.editedAt ? '\nEdited ' + fullTimestamp(post.editedAt) : '')}
			>{#if post.editedAt}<span class="edited-indicator" aria-hidden="true"
					><Icon name="edit" size={13} /></span
				><span id={editDescriptionId} class="visually-hidden"
					>Edited {fullTimestamp(post.editedAt)}</span
				>{/if}<time
				datetime={post.createdAt}
				title={fullTimestamp(post.createdAt)}
				aria-label={fullTimestamp(post.createdAt)}
				>{compactDate(post.createdAt)}</time
			></a
		>
	</div>
	{#if post.sponsored}<p class="badge sponsored">
			Sponsored / commercial relationship
		</p>{/if}
	{#if !post.project}
		{#if detail}<h1 class="visually-hidden">{postTitle(post)}</h1>{/if}
		{#if !detail}<h2 class="visually-hidden">
				{postTitle(post)}
			</h2>{/if}
		{#if detail || content}<p class="prose">
				{#if content}{@render content(post.text)}{:else}{post.text}{/if}
			</p>{:else}<a class="post-body-link" href={postHref()}
				><p class="prose">{post.text}</p></a
			>{/if}
	{/if}

	{#if post.images.length}<ImageCarousel
			images={post.images}
			{detail}
			interactive={member}
			initialIndex={initialImageIndex}
			onChange={onImageChange}
		/>{/if}
	{#if media}{@render media()}{:else if post.video && safeURL(post.video.playlist)}
		<video
			controls
			preload="none"
			poster={safeURL(post.video.thumbnail)}
			aria-label={post.video.alt || 'Post video'}
			><source
				src={safeURL(post.video.playlist)}
				type="application/vnd.apple.mpegurl"
			/></video
		>
		<p class="video-fallback">
			<a href={safeURL(post.video.playlist)}>Open video</a> · Playback requires a
			browser that supports HLS video.
		</p>
	{/if}
	{#if post.project}
		<div class="project-summary">
			{#if detail}<h1>{postTitle(post)}</h1>{:else}<h2>
					<a href={postHref()}>{postTitle(post)}</a>
				</h2>{/if}
			<div class="project-chips">
				{#if post.project.common.status}<span
						class="status-badge"
						class:finished={post.project.common.status.split('#').at(-1) ===
							'finished'}
						class:wip={post.project.common.status.split('#').at(-1) === 'wip'}
						>{label(post.project.common.status)}</span
					>{/if}
				<CraftBadge craft={post.project.common.craftType} />
				{#each post.project.common.additionalCrafts || [] as craft}<CraftBadge
						craft={craft.craftType}
					/>{/each}
			</div>
		</div>
	{/if}
	{#if post.project && !detail && (hasPattern || sizeSummary)}
		<dl class="project-preview-metadata">
			{#if hasPattern && pattern}<dt>Pattern</dt>
				<dd>
					{#if pattern.name?.trim()}{#if field}{@render field(
								pattern.name,
								pattern.nameFacets
							)}{:else}{pattern.name}{/if}{/if}
					{#if pattern.name?.trim() && (pattern.designer?.trim() || pattern.publisher?.trim())}
						{' by '}
					{/if}
					{#if pattern.designer?.trim()}{#if field}{@render field(
								pattern.designer,
								pattern.designerFacets
							)}{:else}{pattern.designer}{/if}{/if}{#if pattern.designer?.trim() && pattern.publisher?.trim()}{', '}
					{/if}
					{#if pattern.publisher?.trim()}{#if field}{@render field(
								pattern.publisher,
								pattern.publisherFacets
							)}{:else}{pattern.publisher}{/if}{/if}
					{#if pattern.selfDrafted}{#if pattern.name?.trim() || pattern.designer?.trim() || pattern.publisher?.trim()}
							{' · '}
						{/if}Self-drafted{/if}
				</dd>{/if}
			{#if sizeSummary}<dt>{sizeSummary.label}</dt>
				<dd>{sizeSummary.value}</dd>{/if}
		</dl>
	{/if}

	{#if post.project && detail}{@const common = post.project.common}
		<section class="project-details" aria-label="Project details">
			<h2>The project details</h2>
			<dl>
				<dt>Craft</dt>
				<dd>{label(common.craftType)}</dd>
				{#if common.additionalCrafts?.length}<dt>Other crafts</dt>
					<dd>
						{common.additionalCrafts.map((c) => label(c.craftType)).join(', ')}
					</dd>{/if}
				{#if common.status}<dt>Status</dt>
					<dd>{label(common.status)}</dd>{/if}
				{#if common.duration}<dt>Time spent</dt>
					<dd>{common.duration}</dd>{/if}
				{#if common.pattern}{@const pattern = common.pattern}
					<dt>Pattern</dt>
					<dd>
						{#if field && pattern.name}{@render field(
								pattern.name,
								pattern.nameFacets
							)}{#if safeURL(pattern.url)}
								<a href={safeURL(pattern.url)} rel="ugc nofollow noreferrer"
									>Pattern website ↗</a
								>{/if}{:else if safeURL(pattern.url)}<a
								href={safeURL(pattern.url)}
								rel="ugc nofollow noreferrer"
								>{pattern.name || 'View pattern'}</a
							>{:else}{pattern.name ||
								(pattern.selfDrafted ? 'Self drafted' : 'Not specified')}{/if}
					</dd>
					{#if pattern.designer}<dt>Designer</dt>
						<dd>
							{#if field}{@render field(
									pattern.designer,
									pattern.designerFacets
								)}{:else}{pattern.designer}{/if}
						</dd>{/if}{#if pattern.publisher}<dt>Publisher</dt>
						<dd>
							{#if field}{@render field(
									pattern.publisher,
									pattern.publisherFacets
								)}{:else}{pattern.publisher}{/if}
						</dd>{/if}{#if pattern.difficulty}<dt>Difficulty</dt>
						<dd>{label(pattern.difficulty)}</dd>{/if}{/if}
				{#if common.materials?.length}<dt>Materials</dt>
					<dd>
						<ul>
							{#each common.materials as material}<li>
									{#if field}{@render field(
											material.text,
											material.facets
										)}{:else}{material.text}{/if}
								</li>{/each}
						</ul>
					</dd>{/if}
				{#if common.colors?.length}<dt>Colours</dt>
					<dd>{common.colors.map(label).join(', ')}</dd>{/if}
				{#if gaugeText(post.project.details?.gauge)}<dt>Gauge</dt>
					<dd>{gaugeText(post.project.details?.gauge)}</dd>{/if}
				{#each Object.entries(post.project.details || {}) as [key, value]}{#if detailFields[key] && typeof value === 'string'}<dt
						>
							{detailFields[key]}
						</dt>
						<dd>{value.includes('#') ? label(value) : value}</dd>{/if}{/each}
				{#if common.designTags?.length}<dt>Design details</dt>
					<dd>{common.designTags.map(label).join(', ')}</dd>{/if}
				{#if common.tags?.length}<dt>Tags</dt>
					<dd>{common.tags.map((t) => '#' + t).join(' ')}</dd>{/if}
			</dl>
		</section>{/if}
	{#if post.project}<p class="prose project-notes">
			{#if content}{@render content(post.text)}{:else}{post.text}{/if}
		</p>{/if}
</article>

<style>
	.edited-indicator {
		display: inline-flex;
		align-items: center;
		margin-right: 4px;
	}
	.project-preview-metadata {
		display: grid;
		grid-template-columns: 112px minmax(0, 1fr);
		gap: 4px 0;
		border-top: 1px solid var(--hairline);
		padding-top: 8px;
		margin: 12px 0 0;
	}
	.project-preview-metadata dt {
		font-size: 12px;
		letter-spacing: 1.68px;
		text-transform: uppercase;
		color: var(--ink-3);
	}
	.project-preview-metadata dd {
		margin: 0;
		font-size: 13px;
		font-weight: 600;
		line-height: 1.45;
		overflow-wrap: anywhere;
	}
	.member-author-line {
		display: flex;
		align-items: center;
		min-width: 0;
	}
	.member-author-line strong {
		flex: 0 1 auto;
		min-width: 0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.author-crafts {
		display: inline-flex;
		align-items: center;
		flex-shrink: 0;
		gap: 4px;
		padding-left: 8px;
	}
	.author-craft-icon {
		width: 16px;
		height: 16px;
		flex-shrink: 0;
		background: currentColor;
		mask: var(--craft-icon) center / contain no-repeat;
	}
	.craft-count,
	.craft-count-measure {
		font-size: 13px;
		line-height: 1.45;
		white-space: nowrap;
	}
	.craft-count-measure {
		position: absolute;
		visibility: hidden;
		pointer-events: none;
	}
</style>
