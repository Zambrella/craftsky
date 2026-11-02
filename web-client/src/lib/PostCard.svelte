<script lang="ts">
	import Avatar from './Avatar.svelte';
	import CraftBadge from './CraftBadge.svelte';
	import {
		authorName,
		gaugeText,
		date,
		compactDate,
		fullTimestamp,
		label,
		postPath,
		profilePath,
		safeURL,
		postTitle,
		type Post
	} from './public';
	let { post, detail = false }: { post: Post; detail?: boolean } = $props();
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
		<a href={profilePath(post.author.did)}
			><Avatar author={post.author} /><span
				><strong>{authorName(post.author)}</strong
				>{#if post.author.business}<span class="business-label"
						>Business account</span
					>{/if}<small>@{post.author.handle}</small></span
			></a
		><a class="date" href={postPath(post)}
			><time
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
		{#if detail}<p class="prose">{post.text}</p>{:else}<a
				class="post-body-link"
				href={postPath(post)}><p class="prose">{post.text}</p></a
			>{/if}
	{/if}

	{#if post.images.length}<div
			class="gallery"
			role="region"
			aria-label="Post images"
		>
			{#each post.images as image, i}{#if safeURL(image.fullsize)}<a
						href={safeURL(image.fullsize)}
						aria-label={image.alt || `View image ${i + 1}`}
						><img
							src={safeURL(
								detail ? image.fullsize : image.thumb || image.fullsize
							)}
							alt={image.alt}
							width={image.aspectRatio?.width}
							height={image.aspectRatio?.height}
							loading={detail && i === 0 ? 'eager' : 'lazy'}
							fetchpriority={detail && i === 0 ? 'high' : 'auto'}
						/></a
					>{/if}{/each}
		</div>{/if}
	{#if post.video && safeURL(post.video.playlist)}
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
		<p>
			<a href={safeURL(post.video.playlist)}>Open video</a> · Playback requires a
			browser that supports HLS video.
		</p>
	{/if}
	{#if post.project}
		<div class="project-summary">
			{#if detail}<h1>{postTitle(post)}</h1>{:else}<h2>
					<a href={postPath(post)}>{postTitle(post)}</a>
				</h2>{/if}
			<div class="project-chips">
				{#if post.project.common.status}<span
						class="status-badge"
						class:finished={post.project.common.status.endsWith('#finished')}
						>{label(post.project.common.status)}</span
					>{/if}
				<CraftBadge craft={post.project.common.craftType} />
				{#each post.project.common.additionalCrafts || [] as craft}<CraftBadge
						craft={craft.craftType}
					/>{/each}
			</div>
		</div>
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
						{#if safeURL(pattern.url)}<a
								href={safeURL(pattern.url)}
								rel="ugc nofollow noreferrer"
								>{pattern.name || 'View pattern'}</a
							>{:else}{pattern.name ||
								(pattern.selfDrafted ? 'Self drafted' : 'Not specified')}{/if}
					</dd>
					{#if pattern.designer}<dt>Designer</dt>
						<dd>{pattern.designer}</dd>{/if}{#if pattern.publisher}<dt>
							Publisher
						</dt>
						<dd>{pattern.publisher}</dd>{/if}{#if pattern.difficulty}<dt>
							Difficulty
						</dt>
						<dd>{label(pattern.difficulty)}</dd>{/if}{/if}
				{#if common.materials?.length}<dt>Materials</dt>
					<dd>
						<ul>
							{#each common.materials as material}<li>
									{material.text}
								</li>{/each}
						</ul>
					</dd>{/if}
				{#if common.colors?.length}<dt>Colours</dt>
					<dd>{common.colors.join(', ')}</dd>{/if}
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
	{#if post.project}<p class="prose project-notes">{post.text}</p>{/if}
	{#if post.editedAt}<p class="muted">
			Edited <time datetime={post.editedAt}>{date(post.editedAt)}</time>
		</p>{/if}
</article>
