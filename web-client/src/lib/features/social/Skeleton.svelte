<script lang="ts">
	let {
		kind = 'posts'
	}: {
		kind?:
			'posts' | 'people' | 'profile' | 'activity' | 'management' | 'comments';
	} = $props();
</script>

<div
	class="skeletons"
	role="status"
	aria-label="Loading content"
	aria-live="polite"
>
	{#each Array(kind === 'profile' ? 1 : 6) as _, index}
		<div
			class="skeleton"
			class:profile={kind === 'profile'}
			class:post={kind === 'posts'}
			aria-hidden="true"
		>
			{#if kind === 'profile'}<div class="band bone"></div>{/if}
			<div class="identity">
				{#if kind === 'activity'}<span class="activity-icon bone"></span>{/if}
				<span
					class="skeleton-avatar bone"
					class:person={kind === 'people'}
					class:thumbnail={kind === 'management'}
					class:comment={kind === 'comments'}
				></span>
				<div class="identity-lines">
					<i class="bone name"></i><i class="bone metadata"
					></i>{#if kind === 'management'}<i class="bone short"></i>{/if}
				</div>
				{#if kind === 'posts' || kind === 'management'}<span class="more bone"
					></span>{:else if kind === 'people'}<span class="trailing bone"
					></span>{/if}
			</div>
			{#if kind === 'posts' || kind === 'profile' || kind === 'comments'}
				<div class="lines">
					<i class="bone"></i><i class="bone"></i><i class="bone short"></i>
				</div>
			{/if}
			{#if kind === 'posts' && index === 0}<div class="media bone"></div>{/if}
			{#if kind === 'posts'}<div class="reactions">
					<i class="bone"></i><i class="bone"></i><i class="bone"></i><i
						class="bone"
					></i>
				</div>{/if}
		</div>
	{/each}
</div>

<style>
	.skeletons {
		--skeleton-base: color-mix(in srgb, var(--ink) 10%, var(--surface));
		--skeleton-highlight: color-mix(in srgb, var(--ink) 3%, var(--surface));
		pointer-events: none;
		overflow: hidden;
	}
	.skeleton {
		padding: 12px 16px;
		overflow: hidden;
	}
	.skeleton.post {
		padding-bottom: 24px;
	}
	.bone {
		background: linear-gradient(
			107deg,
			var(--skeleton-base) 10%,
			var(--skeleton-highlight) 30%,
			var(--skeleton-base) 40%
		);
		background-size: 200vw 100%;
		background-attachment: fixed;
		border-radius: 999px;
		animation: shimmer 1400ms linear infinite;
	}
	.identity {
		display: flex;
		gap: 12px;
		align-items: center;
	}
	.skeleton-avatar {
		border: 0;
		box-shadow: none;
		width: 40px;
		height: 40px;
		flex-shrink: 0;
	}
	.skeleton-avatar.person {
		width: 48px;
		height: 48px;
	}
	.skeleton-avatar.thumbnail {
		width: 72px;
		height: 72px;
		border-radius: 6px;
	}
	.skeleton-avatar.comment {
		width: 36px;
		height: 36px;
	}
	.activity-icon {
		width: 24px;
		height: 24px;
		flex-shrink: 0;
	}
	.identity-lines {
		display: grid;
		gap: 4px;
		flex: 1;
		min-width: 0;
	}
	i {
		display: block;
		height: 14px;
		width: 100%;
	}
	.name {
		width: min(132px, 100%);
	}
	.metadata {
		width: min(88px, 100%);
		height: 13px;
	}
	.short {
		width: 65%;
	}
	.more {
		width: 32px;
		height: 32px;
		flex-shrink: 0;
	}
	.trailing {
		width: 72px;
		height: 36px;
		flex-shrink: 0;
	}
	.lines {
		display: grid;
		gap: 7px;
		margin-top: 12px;
		padding-block: 3.5px;
	}
	.media {
		aspect-ratio: 16/9;
		border-radius: 6px;
		margin-top: 12px;
	}
	.reactions {
		display: flex;
		justify-content: space-between;
		margin-top: 12px;
	}
	.reactions i {
		width: 54px;
		height: 22px;
	}
	.reactions i:last-child {
		width: 32px;
	}
	.band {
		height: 128px;
		margin: -12px -16px -38px;
		border-radius: 0;
	}
	.profile .identity {
		position: relative;
		display: grid;
		justify-items: center;
		text-align: center;
	}
	.profile .skeleton-avatar {
		width: 106px;
		height: 106px;
		border: 5px solid var(--paper);
	}
	@keyframes shimmer {
		from {
			background-position: 100vw 0;
		}
		to {
			background-position: -100vw 0;
		}
	}
	:global([dir='rtl']) .bone {
		animation-direction: reverse;
	}
	@media (prefers-reduced-motion: reduce) {
		.bone {
			animation: none;
			background: var(--skeleton-base);
		}
	}
</style>
