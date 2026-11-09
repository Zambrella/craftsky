<script lang="ts">
	import guide from './guide-data.json';
</script>

<svelte:head
	><title>Safety and support — CraftSky</title><meta
		name="referrer"
		content="no-referrer"
	/></svelte:head
>
<article class="safety-guide">
	<h1>{guide.title}</h1>
	<p>{guide.intro}</p>
	<p><strong>{guide.urgent}</strong></p>
	<h2 id="support">{guide.supportTitle}</h2>
	<p>{guide.supportIntro}</p>
	{#each guide.resources as resource}<section>
			<h3>
				<a
					href={resource.url}
					target="_blank"
					rel="noopener noreferrer"
					referrerpolicy="no-referrer">{resource.title} ↗</a
				>
			</h3>
			<p><strong>{resource.scope}</strong> — {resource.body}</p>
		</section>{/each}
	<p>{guide.outsideUK}</p>
	<h2>{guide.youngTitle}</h2>
	<p>{guide.youngBody}</p>
	<h2>{guide.stepsTitle}</h2>
	<ol class="steps">
		{#each guide.steps as step}<li>
				<span class="symbol" aria-hidden="true">{step.symbol}</span>
				<h3>{step.title}</h3>
				<p>{step.body}</p>
			</li>{/each}
	</ol>
	<h2>{guide.messagesTitle}</h2>
	<p>{guide.messagesBody}</p>
	<h2>{guide.parentsTitle}</h2>
	<p>{guide.parentsBody}</p>
	<p>{guide.limits}</p>
	<p>Resources reviewed: {guide.reviewed}</p>
</article>

<style>
	.safety-guide {
		max-width: 52rem;
		margin: auto;
		padding: 24px;
		overflow-wrap: anywhere;
	}
	p {
		line-height: 1.65;
	}
	h2 {
		margin-top: 2rem;
	}
	a {
		text-decoration: underline;
		text-underline-offset: 0.2em;
	}
	.steps {
		display: grid;
		gap: 16px;
		list-style: none;
		padding: 0;
	}
	.steps li {
		padding: 16px;
		border: 1.5px solid var(--ink);
		border-radius: 14px;
	}
	.symbol {
		display: inline-grid;
		place-items: center;
		border: 2px solid currentColor;
		border-radius: 50%;
		width: 40px;
		height: 40px;
		font-size: 24px;
	}
	@media (min-width: 750px) {
		.steps {
			grid-template-columns: repeat(3, minmax(0, 1fr));
		}
	}
</style>
