<script lang="ts">
	import type { Author } from '#lib/api/types.ts';
	import { name } from './model';
	import { identityCrafts, fittingIdentityCrafts } from './compact-identity';
	let { author }: { author: Author } = $props();
	const crafts = $derived(identityCrafts(author.crafts));
	let visible = $state(5);
	function fit(node: HTMLElement, entries: typeof crafts) {
		let dead = false;
		const context = document.createElement('canvas').getContext('2d');
		function measure() {
			if (dead) return;
			const text = node.querySelector('strong');
			const count = node.querySelector<HTMLElement>('.measure-count');
			if (!text || !count) return;
			if (context) context.font = getComputedStyle(count).font;
			visible = fittingIdentityCrafts(
				node.clientWidth,
				text.scrollWidth,
				entries,
				(value) =>
					context?.measureText('+' + value).width ||
					count.getBoundingClientRect().width
			);
		}
		const observer = new ResizeObserver(measure);
		observer.observe(node);
		void document.fonts.ready.then(measure);
		measure();
		return {
			update(next: typeof crafts) {
				entries = next;
				measure();
			},
			destroy() {
				dead = true;
				observer.disconnect();
			}
		};
	}
</script>

<span class="compact-identity">
	<span class="identity-name" use:fit={crafts}>
		<strong>{name(author)}</strong>{#if crafts.length}<span
				class="inline-crafts"
				role="img"
				aria-label={'Crafts: ' + crafts.map((craft) => craft.name).join(', ')}
				title={crafts.map((craft) => craft.name).join(', ')}
			>
				{#each crafts
					.filter((craft) => craft.icon)
					.slice(0, visible) as craft}<span
						class="inline-craft"
						style:--craft-icon={`url('/craft_icons/${craft.id}.svg')`}
						aria-hidden="true"
					></span>{/each}
				{#if crafts.length > visible}<span
						class="craft-count"
						aria-hidden="true">+{crafts.length - visible}</span
					>{/if}
			</span>{/if}<span class="measure-count" aria-hidden="true"
			>+{crafts.length}</span
		>
	</span>
	<small>@{author.handle || 'Account unavailable'}</small>
</span>

<style>
	.compact-identity {
		display: block;
		min-width: 0;
		flex: 1;
	}
	.identity-name {
		display: flex;
		align-items: center;
		min-width: 0;
	}
	strong {
		flex: 0 1 auto;
		min-width: 0;
		font-size: 14px;
		font-weight: 600;
		line-height: 1.3;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	small {
		display: block;
		color: var(--ink-2);
		font-size: 13px;
		line-height: 1.45;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.inline-crafts {
		display: inline-flex;
		align-items: center;
		flex-shrink: 0;
		gap: 4px;
		padding-left: 8px;
	}
	.inline-craft {
		width: 16px;
		height: 16px;
		flex-shrink: 0;
		background: currentColor;
		mask: var(--craft-icon) center / contain no-repeat;
	}
	.craft-count,
	.measure-count {
		font-size: 13px;
		line-height: 1.45;
		white-space: nowrap;
	}
	.measure-count {
		position: absolute;
		visibility: hidden;
		pointer-events: none;
	}
</style>
