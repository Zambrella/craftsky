<script lang="ts">
	let { kind = 'posts' }: { kind?: 'posts' | 'people' | 'profile' } = $props();
</script>

<div
	class="skeletons"
	role="status"
	aria-label="Loading content"
	aria-live="polite"
>
	{#each Array(kind === 'profile' ? 1 : 3) as _, index}<div
			class="skeleton"
			class:card={kind === 'posts'}
			class:profile={kind === 'profile'}
			aria-hidden="true"
		>
			{#if kind === 'profile'}<div class="band shimmer"></div>{/if}
			<div class="identity">
				<span class="avatar shimmer"></span><span class="identity-lines"
					><i class="shimmer"></i><i class="shimmer short"></i></span
				>
			</div>
			{#if kind !== 'people'}<div class="lines">
					<i class="shimmer"></i><i class="shimmer"></i><i class="shimmer short"
					></i>
				</div>{/if}
			{#if kind === 'posts' && index === 0}<div
					class="media shimmer"
				></div>{/if}
		</div>{/each}
</div>

<style>
	.skeletons {
		padding: 8px 16px;
		display: grid;
		gap: 16px;
	}
	.skeleton {
		padding: 12px 0;
		overflow: hidden;
	}
	.skeleton.card {
		background: var(--surface);
		border: 1.5px solid var(--hairline);
		border-radius: 16px;
		padding: 16px;
	}
	.shimmer {
		background: var(--paper-2);
		border-radius: 6px;
		animation: breathe 1.5s ease-in-out infinite alternate;
	}
	.identity {
		display: flex;
		gap: 12px;
		align-items: center;
	}
	.avatar {
		width: 44px;
		height: 44px;
		border-radius: 50%;
		flex-shrink: 0;
	}
	.identity-lines {
		display: grid;
		gap: 8px;
		width: 52%;
	}
	i {
		display: block;
		height: 12px;
		width: 100%;
	}
	.short {
		width: 65%;
	}
	.lines {
		display: grid;
		gap: 8px;
		margin-top: 20px;
	}
	.media {
		height: 220px;
		margin-top: 20px;
	}
	.band {
		height: 128px;
		margin: -12px 0 -38px;
	}
	.profile .identity {
		position: relative;
		display: grid;
		justify-items: center;
		text-align: center;
	}
	.profile .avatar {
		width: 106px;
		height: 106px;
		border: 5px solid var(--paper);
	}
	@keyframes breathe {
		to {
			opacity: 0.45;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.shimmer {
			animation: none;
		}
	}
</style>
