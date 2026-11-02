<script lang="ts">
	import type { CraftskyAPI, Json, JsonObject } from '#lib/api/client.ts';
	import { label, options, record, str } from './model.ts';
	import FacetTextInput from './FacetTextInput.svelte';
	let {
		project,
		api,
		disabled = false,
		onchange
	}: {
		project: JsonObject;
		api: CraftskyAPI;
		disabled?: boolean;
		onchange: (project: JsonObject) => void;
	} = $props();
	const common = $derived(record(project.common));
	const pattern = $derived(record(common.pattern));
	const details = $derived(record(project.details));
	const craft = $derived(str(common.craftType));
	const shortCraft = $derived(craft.split('#').at(-1) || 'knitting');
	const textile = $derived(
		shortCraft === 'knitting' || shortCraft === 'crochet'
	);
	function updateCommon(key: string, value: Json | undefined) {
		onchange({ ...project, common: { ...common, [key]: value } });
	}
	function updatePattern(key: string, value: Json | undefined) {
		updateCommon('pattern', { ...pattern, [key]: value });
	}
	function updateDetail(key: string, value: Json | undefined) {
		onchange({
			...project,
			details: {
				...details,
				$type: `social.craftsky.project.${shortCraft}#details`,
				[key]: value
			}
		});
	}
	function projectType(value: string) {
		const subtype = str(details.projectSubtype);
		onchange({
			...project,
			details: {
				...details,
				$type: `social.craftsky.project.${shortCraft}#details`,
				projectType: value || undefined,
				projectSubtype: options('projectSubtype', craft, value).some(
					(option) => option.id === subtype
				)
					? subtype
					: undefined
			}
		});
	}
	function changeCraft(value: string) {
		onchange({
			...project,
			common: { ...common, craftType: value },
			details: undefined
		});
	}
	function toggle(key: string, value: string) {
		const values = Array.isArray(common[key]) ? (common[key] as string[]) : [];
		if (values.includes(value))
			updateCommon(
				key,
				values.filter((item) => item !== value)
			);
		else if (values.length < 10) updateCommon(key, [...values, value]);
	}
	function additional(value: string) {
		const values = Array.isArray(common.additionalCrafts)
			? common.additionalCrafts.map(record)
			: [];
		const exists = values.some((item) => item.craftType === value);
		updateCommon(
			'additionalCrafts',
			exists
				? values.filter((item) => item.craftType !== value)
				: [...values, { craftType: value }]
		);
	}
	function materials(text: string) {
		const previous = Array.isArray(common.materials)
			? common.materials.map(record)
			: [];
		const values = text.split('\n').filter((value) => value.trim());
		const used = new Set<number>();
		const next = values.map((value, index) => {
			let source = previous.findIndex(
				(item, position) => !used.has(position) && item.text === value
			);
			if (source < 0 && values.length === previous.length && !used.has(index))
				source = index;
			const old = source < 0 ? undefined : previous[source];
			if (source >= 0) used.add(source);
			const material: JsonObject = { text: value };
			if (old?.existingIndex !== undefined)
				material.existingIndex = old.existingIndex;
			if (old?.text === value && old.facets) material.facets = old.facets;
			return material;
		});
		updateCommon('materials', next);
	}

	function gauge(key: string, value: string) {
		const values = { ...record(details.gauge) };
		if (value === '') delete values[key];
		else values[key] = key === 'unit' ? value : Number(value);
		updateDetail('gauge', Object.keys(values).length ? values : undefined);
	}
</script>

<section class="composer-section project-editor">
	<h2>Project details</h2>
	<div class="fields">
		<label
			>Primary craft<select
				value={craft}
				onchange={(e) => changeCraft(e.currentTarget.value)}
				>{#each options('craft') as option}<option value={option.id}
						>{option.label}</option
					>{/each}</select
			></label
		><label
			>Status<select
				value={str(common.status)}
				onchange={(e) => updateCommon('status', e.currentTarget.value)}
				>{#each options('status') as option}<option value={option.id}
						>{option.label}</option
					>{/each}</select
			></label
		>
	</div>
	<label
		>Project title<input
			maxlength="200"
			value={str(common.title)}
			oninput={(e) => updateCommon('title', e.currentTarget.value || undefined)}
			placeholder="What have you been making?"
		/></label
	>
	<fieldset>
		<legend>Other crafts in this project</legend>
		<div class="chips">
			{#each options('craft').filter((item) => item.id !== craft) as item}<label
					class="check"
					><input
						type="checkbox"
						checked={(Array.isArray(common.additionalCrafts)
							? common.additionalCrafts
							: []
						).some((value) => record(value).craftType === item.id)}
						onchange={() => additional(item.id)}
					/>{item.label}</label
				>{/each}
		</div>
	</fieldset>
	<label
		>Time spent<input
			maxlength="100"
			value={str(common.duration)}
			oninput={(e) =>
				updateCommon('duration', e.currentTarget.value || undefined)}
			placeholder="For example, a lovely weekend"
		/></label
	>
	<details open={!!Object.keys(pattern).length}>
		<summary>Pattern</summary>
		<div class="stack">
			<label class="check"
				><input
					type="checkbox"
					checked={pattern.selfDrafted === true}
					onchange={(e) =>
						updatePattern('selfDrafted', e.currentTarget.checked)}
				/>Self drafted</label
			>
			<FacetTextInput
				{api}
				{disabled}
				label="Pattern name"
				value={str(pattern.name) || '#'}
				allowed={['hashtag']}
				onchange={(value) =>
					updatePattern(
						'name',
						value.trim() === '#' ? undefined : value || undefined
					)}
			/>
			<div class="fields">
				<FacetTextInput
					{api}
					{disabled}
					label="Designer"
					value={str(pattern.designer)}
					allowed={['mention']}
					onchange={(value) => updatePattern('designer', value || undefined)}
				/><FacetTextInput
					{api}
					{disabled}
					label="Publisher"
					value={str(pattern.publisher)}
					allowed={['mention']}
					onchange={(value) => updatePattern('publisher', value || undefined)}
				/>
			</div>
			<label
				>Pattern link<input
					type="url"
					value={str(pattern.url)}
					oninput={(e) =>
						updatePattern('url', e.currentTarget.value || undefined)}
					placeholder="https://"
				/></label
			><label
				>Difficulty<select
					value={str(pattern.difficulty)}
					onchange={(e) =>
						updatePattern('difficulty', e.currentTarget.value || undefined)}
					><option value="">Not specified</option
					>{#each options('difficulty') as option}<option value={option.id}
							>{option.label}</option
						>{/each}</select
				></label
			>
		</div>
	</details>
	<FacetTextInput
		{api}
		{disabled}
		label="Materials"
		hint="One material per line"
		rows={3}
		value={(Array.isArray(common.materials) ? common.materials : [])
			.map((item) => str(record(item).text))
			.join('\n')}
		onchange={materials}
		placeholder="Yarn, fabric, thread and other supplies"
	/>
	<details>
		<summary>Colours and design</summary>
		<fieldset>
			<legend>Colours</legend>
			<div class="chips">
				{#each options('color') as item}<label class="check"
						><input
							type="checkbox"
							checked={Array.isArray(common.colors) &&
								common.colors.includes(item.id)}
							onchange={() => toggle('colors', item.id)}
						/>{item.label}</label
					>{/each}
			</div>
		</fieldset>
		<fieldset>
			<legend>Design tags</legend>
			<div class="chips">
				{#each options('designTag') as item}<label class="check"
						><input
							type="checkbox"
							checked={Array.isArray(common.designTags) &&
								common.designTags.includes(item.id)}
							onchange={() => toggle('designTags', item.id)}
						/>{item.label}</label
					>{/each}
			</div>
		</fieldset>
		<p class="muted">Choose up to ten in each group.</p>
	</details>
	{#if ['sewing', 'knitting', 'crochet', 'quilting'].includes(shortCraft)}
		<details open={!!Object.keys(details).length}>
			<summary>{label(craft)} details</summary>
			<div class="stack">
				<div class="fields">
					<label
						>Project type<select
							value={str(details.projectType)}
							onchange={(e) => {
								projectType(e.currentTarget.value);
							}}
							><option value="">Not specified</option
							>{#each options('projectType', craft) as option}<option
									value={option.id}>{option.label}</option
								>{/each}</select
						></label
					><label
						>Project subtype<select
							value={str(details.projectSubtype)}
							onchange={(e) =>
								updateDetail(
									'projectSubtype',
									e.currentTarget.value || undefined
								)}
							><option value="">Not specified</option
							>{#each options('projectSubtype', craft, str(details.projectType)) as option}<option
									value={option.id}>{option.label}</option
								>{/each}</select
						></label
					>
				</div>
				{#if shortCraft === 'sewing'}<label
						>Size made<input
							maxlength="100"
							value={str(details.sizeMade)}
							oninput={(e) =>
								updateDetail('sizeMade', e.currentTarget.value || undefined)}
						/></label
					><label
						>Fit notes<textarea
							rows="3"
							maxlength="600"
							value={str(details.fitNotes)}
							oninput={(e) =>
								updateDetail('fitNotes', e.currentTarget.value || undefined)}
						></textarea></label
					>{/if}
				{#if textile}<div class="fields">
						<label
							>Yarn weight<select
								value={str(details.yarnWeight)}
								onchange={(e) =>
									updateDetail(
										'yarnWeight',
										e.currentTarget.value || undefined
									)}
								><option value="">Not specified</option
								>{#each options('yarnWeight') as option}<option
										value={option.id}>{option.label}</option
									>{/each}</select
							></label
						><label
							>{shortCraft === 'knitting' ? 'Needle' : 'Hook'} size (mm)<input
								inputmode="decimal"
								value={str(
									details[
										shortCraft === 'knitting' ? 'needleSizeMm' : 'hookSizeMm'
									]
								)}
								oninput={(e) =>
									updateDetail(
										shortCraft === 'knitting' ? 'needleSizeMm' : 'hookSizeMm',
										e.currentTarget.value || undefined
									)}
								placeholder="4.0mm"
							/></label
						>
					</div>
					<label
						>Finished size<input
							maxlength="100"
							value={str(details.finishedSize)}
							oninput={(e) =>
								updateDetail(
									'finishedSize',
									e.currentTarget.value || undefined
								)}
						/></label
					>
					<fieldset>
						<legend>Gauge</legend>
						<div class="fields gauge">
							{#each [['stitches', 'Stitches'], ['rows', 'Rows'], ['measurement', 'Over']] as [key, name]}<label
									>{name}<input
										type="number"
										min="1"
										step="1"
										value={typeof record(details.gauge)[key] === 'number'
											? Number(record(details.gauge)[key])
											: ''}
										oninput={(e) => gauge(key, e.currentTarget.value)}
									/></label
								>{/each}<label
								>Unit<select
									value={str(record(details.gauge).unit)}
									onchange={(e) => gauge('unit', e.currentTarget.value)}
									><option value="">Choose</option><option value="cm">cm</option
									><option value="in">inches</option></select
								></label
							>
						</div>
					</fieldset>{/if}
				{#if shortCraft === 'quilting'}<label
						>Finished size<input
							maxlength="100"
							value={str(details.size)}
							oninput={(e) =>
								updateDetail('size', e.currentTarget.value || undefined)}
						/></label
					>{#each [['piecingTechnique', 'Piecing technique'], ['quiltingMethod', 'Quilting method']] as [key, name]}<label
							>{name}<select
								value={str(details[key])}
								onchange={(e) =>
									updateDetail(key, e.currentTarget.value || undefined)}
								><option value="">Not specified</option
								>{#each options(key, craft) as option}<option value={option.id}
										>{option.label}</option
									>{/each}</select
							></label
						>{/each}{/if}
			</div>
		</details>{/if}
</section>
