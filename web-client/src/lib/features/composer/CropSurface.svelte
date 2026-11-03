<script lang="ts">
	let {
		url,
		rotation,
		x,
		y,
		width,
		height,
		disabled = false,
		onchange
	}: {
		url: string;
		rotation: number;
		x: number;
		y: number;
		width: number;
		height: number;
		disabled?: boolean;
		onchange: (crop: {
			x: number;
			y: number;
			width: number;
			height: number;
		}) => void;
	} = $props();
	let availableWidth = $state(0),
		availableHeight = $state(0),
		naturalWidth = $state(1),
		naturalHeight = $state(1);
	const scale = $derived(
		Math.min(
			Math.max(1, availableWidth - 40) /
				(rotation % 2 ? naturalHeight : naturalWidth),
			Math.max(1, availableHeight - 40) /
				(rotation % 2 ? naturalWidth : naturalHeight)
		)
	);
	const imageWidth = $derived(naturalWidth * scale),
		imageHeight = $derived(naturalHeight * scale);
	let drag:
		| {
				id: number;
				handle: string;
				clientX: number;
				clientY: number;
				x: number;
				y: number;
				width: number;
				height: number;
		  }
		| undefined;
	function adjust(
		handle: string,
		dx: number,
		dy: number,
		original = { x, y, width, height }
	) {
		const { x: left, y: top, width: w, height: h } = original;
		let l = left,
			t = top,
			r = left + w,
			b = top + h;
		if (handle === 'move') {
			l = Math.max(0, Math.min(100 - w, left + dx));
			t = Math.max(0, Math.min(100 - h, top + dy));
			r = l + w;
			b = t + h;
		} else {
			if (handle.includes('left')) l = Math.max(0, Math.min(r - 5, left + dx));
			else r = Math.max(l + 5, Math.min(100, r + dx));
			if (handle.includes('top')) t = Math.max(0, Math.min(b - 5, top + dy));
			else b = Math.max(t + 5, Math.min(100, b + dy));
		}
		onchange({ x: l, y: t, width: r - l, height: b - t });
	}
	function down(event: PointerEvent, handle: string) {
		if (disabled || event.button !== 0) return;
		event.preventDefault();
		event.stopPropagation();
		(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
		drag = {
			id: event.pointerId,
			handle,
			clientX: event.clientX,
			clientY: event.clientY,
			x,
			y,
			width,
			height
		};
	}
	function move(event: PointerEvent) {
		if (!drag || disabled || drag.id !== event.pointerId) return;
		const dx = event.clientX - drag.clientX,
			dy = event.clientY - drag.clientY,
			angle = (rotation * Math.PI) / 2;
		adjust(
			drag.handle,
			((dx * Math.cos(angle) + dy * Math.sin(angle)) / imageWidth) * 100,
			((-dx * Math.sin(angle) + dy * Math.cos(angle)) / imageHeight) * 100,
			drag
		);
	}
	function keyboard(event: KeyboardEvent, handle: string) {
		if (
			disabled ||
			!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown'].includes(event.key)
		)
			return;
		event.preventDefault();
		const step = event.shiftKey ? 10 : 1;
		const dx =
			event.key === 'ArrowLeft' ? -step : event.key === 'ArrowRight' ? step : 0;
		const dy =
			event.key === 'ArrowUp' ? -step : event.key === 'ArrowDown' ? step : 0;
		const angle = (rotation * Math.PI) / 2;
		adjust(
			handle,
			dx * Math.cos(angle) + dy * Math.sin(angle),
			-dx * Math.sin(angle) + dy * Math.cos(angle)
		);
	}
</script>

<div
	class="crop-stage"
	bind:clientWidth={availableWidth}
	bind:clientHeight={availableHeight}
>
	<div
		class="crop-frame"
		style:width={`${imageWidth}px`}
		style:height={`${imageHeight}px`}
		style:transform={`rotate(${rotation * 90}deg)`}
	>
		<img
			src={url}
			alt="Crop preview"
			draggable="false"
			onload={(event) => {
				const image = event.currentTarget as HTMLImageElement;
				naturalWidth = image.naturalWidth;
				naturalHeight = image.naturalHeight;
			}}
		/>
		<div
			class="crop-selection"
			style:left={`${x}%`}
			style:top={`${y}%`}
			style:width={`${width}%`}
			style:height={`${height}%`}
		>
			<button
				class="crop-move"
				type="button"
				aria-label="Move crop"
				{disabled}
				onpointerdown={(event) => down(event, 'move')}
				onpointermove={move}
				onpointerup={() => (drag = undefined)}
				onpointercancel={() => (drag = undefined)}
				onkeydown={(event) => keyboard(event, 'move')}
			></button>
			{#each ['top left', 'top right', 'bottom left', 'bottom right'] as corner}<button
					type="button"
					class="crop-corner"
					class:left={corner.includes('left')}
					class:right={corner.includes('right')}
					class:top={corner.includes('top')}
					class:bottom={corner.includes('bottom')}
					aria-label={`Crop ${corner} corner`}
					{disabled}
					onpointerdown={(event) => down(event, corner)}
					onpointermove={move}
					onpointerup={() => (drag = undefined)}
					onpointercancel={() => (drag = undefined)}
					onkeydown={(event) => keyboard(event, corner)}
				></button>{/each}
		</div>
	</div>
</div>

<style>
	.crop-stage {
		flex: 1;
		min-height: 200px;
		display: grid;
		place-items: center;
		overflow: hidden;
	}
	.crop-frame {
		position: relative;
		flex: none;
	}
	.crop-frame img {
		display: block;
		width: 100%;
		height: 100%;
		object-fit: contain;
		pointer-events: none;
	}
	.crop-selection {
		position: absolute;
		border: 1px solid #f5efe4;
		box-shadow: 0 0 0 200vmax #0009;
	}
	.crop-selection .crop-move {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		min-height: 0;
		border: 0;
		border-radius: 0;
		box-shadow: none;
		padding: 0;
		background:
			linear-gradient(
				to right,
				transparent 33%,
				#fff6 33%,
				#fff6 calc(33% + 1px),
				transparent calc(33% + 1px),
				transparent 66%,
				#fff6 66%,
				#fff6 calc(66% + 1px),
				transparent calc(66% + 1px)
			),
			linear-gradient(
				to bottom,
				transparent 33%,
				#fff6 33%,
				#fff6 calc(33% + 1px),
				transparent calc(33% + 1px),
				transparent 66%,
				#fff6 66%,
				#fff6 calc(66% + 1px),
				transparent calc(66% + 1px)
			);
		touch-action: none;
		cursor: move;
	}
	.crop-selection .crop-corner {
		position: absolute;
		width: 28px;
		height: 28px;
		min-height: 0;
		padding: 0;
		border: 0;
		border-radius: 0;
		box-shadow: none;
		background: transparent;
		touch-action: none;
	}
	.crop-corner::after {
		content: '';
		position: absolute;
		inset: 6px;
		border-color: #7890ff;
		border-style: solid;
		border-width: 0;
	}
	.crop-corner.left {
		left: -14px;
	}
	.crop-corner.right {
		right: -14px;
	}
	.crop-corner.top {
		top: -14px;
	}
	.crop-corner.bottom {
		bottom: -14px;
	}
	.crop-corner.left::after {
		border-left-width: 3px;
	}
	.crop-corner.right::after {
		border-right-width: 3px;
	}
	.crop-corner.top::after {
		border-top-width: 3px;
	}
	.crop-corner.bottom::after {
		border-bottom-width: 3px;
	}
	.crop-corner:focus-visible {
		outline: 2px solid #7890ff;
		outline-offset: 2px;
	}
</style>
