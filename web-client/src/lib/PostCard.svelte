<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { Facet } from './api/types';
	import Avatar from './Avatar.svelte';
	import CraftBadge from './CraftBadge.svelte';
	import ImageCarousel from './ImageCarousel.svelte';
	import Icon from './features/shell/Icon.svelte';
	import { confirmExternalLink } from './features/social/prompt';
	import catalogue from './features/composer/catalogue.json';
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
		member ? '/profiles/' + encodeURIComponent(did) : profilePath(did);
	const postHref = () =>
		member
			? '/posts/' +
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
	const detailRows = $derived.by(() => {
		const common = post.project?.common;
		if (!common) return [];
		const details = post.project?.details || {};
		const rows: { label: string; value: string; href?: string }[] = [];
		const option = (value: string) =>
			catalogue.find((entry) => entry.id === value)?.label || label(value);
		const add = (name: string, value: unknown, optionValue = false) => {
			if (typeof value === 'string' && value.trim())
				rows.push({
					label: name,
					value: optionValue ? option(value) : value.trim()
				});
		};
		add('Duration', common.duration);
		add('Difficulty', common.pattern?.difficulty, true);
		const href = safeURL(common.pattern?.url);
		if (href)
			rows.push({
				label: 'Link',
				value: href.replace(/^https?:\/\//, '').replace(/\/$/, ''),
				href
			});
		const type = [details.projectType, details.projectSubtype]
			.filter(
				(value): value is string => typeof value === 'string' && !!value.trim()
			)
			.map(option)
			.join(' > ');
		add('Project type', type);
		const craft = common.craftType.split('#').at(-1);
		if (craft === 'sewing') {
			add('Size made', details.sizeMade);
			add('Fit notes', details.fitNotes);
		} else if (craft === 'knitting' || craft === 'crochet') {
			add('Yarn weight', details.yarnWeight, true);
			add(
				craft === 'knitting' ? 'Needle size' : 'Hook size',
				craft === 'knitting' ? details.needleSizeMm : details.hookSizeMm
			);
			add('Gauge', gaugeText(details.gauge));
			add('Finished size', details.finishedSize);
		} else if (craft === 'quilting') {
			add('Size', details.size);
			add('Piecing technique', details.piecingTechnique, true);
			add('Quilting method', details.quiltingMethod, true);
		} else
			for (const [key, value] of Object.entries(details))
				if (!['$type', 'projectType', 'projectSubtype'].includes(key))
					add(label(key), value);
		return rows;
	});
	const chipSections = $derived(
		[
			{ label: 'Colours', values: post.project?.common.colors || [] },
			{ label: 'Design', values: post.project?.common.designTags || [] },
			{ label: 'Tags', values: post.project?.common.tags || [] }
		].filter((section) => section.values.some((value) => value.trim()))
	);
	const materialValues = $derived(
		post.project?.common.materials?.filter((material) =>
			material.text.trim()
		) || []
	);
	const optionLabel = (value: string) =>
		catalogue.find((entry) => entry.id === value)?.label || label(value);
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
	{#if post.project && (hasPattern || (detail ? detailRows.length : sizeSummary))}
		<dl class="project-preview-metadata" class:project-detail-metadata={detail}>
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
			{#if !detail && sizeSummary}<dt>{sizeSummary.label}</dt>
				<dd>{sizeSummary.value}</dd>{/if}
			{#if detail}{#each detailRows as row}<dt>{row.label}</dt>
					<dd>
						{#if row.href}<a
								href={row.href}
								rel="ugc nofollow noreferrer"
								onclick={confirmExternalLink}>{row.value}</a
							>{:else}{row.value}{/if}
					</dd>{/each}{/if}
		</dl>
	{/if}

	{#if post.project && detail && (materialValues.length || chipSections.length)}
		<section
			class="project-chip-sections"
			aria-label="Project materials and style"
		>
			{#if materialValues.length}<div>
					<h3>Materials</h3>
					<div class="detail-chips">
						{#each materialValues as material}<span
								>{#if field}{@render field(
										material.text,
										material.facets
									)}{:else}{material.text}{/if}</span
							>{/each}
					</div>
				</div>{/if}
			{#each chipSections as section}<div>
					<h3>{section.label}</h3>
					<div class="detail-chips">
						{#each section.values.filter((value) => value.trim()) as value}<span
								>{section.label === 'Tags' ? value : optionLabel(value)}</span
							>{/each}
					</div>
				</div>{/each}
		</section>
	{/if}
	{#if post.project}<p class="prose project-notes">
			{#if content}{@render content(post.text)}{:else}{post.text}{/if}
		</p>{/if}
</article>

<style>
	.project-detail-metadata a {
		text-decoration: none;
	}
	.project-detail-metadata a:hover {
		text-decoration: underline;
	}
	.project-chip-sections {
		margin-top: 12px;
		padding-top: 8px;
		border-top: 1px solid var(--hairline);
		display: grid;
		gap: 8px;
	}
	.project-chip-sections h3 {
		margin: 0 0 4px;
		font-family: inherit;
		font-size: 12px;
		font-weight: 600;
		line-height: 1.2;
		letter-spacing: 1.68px;
		text-transform: uppercase;
		color: var(--ink-3);
	}
	.detail-chips {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
	}
	.detail-chips > span {
		padding: 4px 8px;
		border: 1.2px solid var(--ink);
		border-radius: 999px;
		font-size: 12px;
		font-weight: 600;
		line-height: 1.2;
	}
	.project-detail-metadata {
		align-items: center;
	}

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
