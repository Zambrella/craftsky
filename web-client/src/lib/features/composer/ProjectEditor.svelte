<script lang="ts">
	import { graphemeCount } from '#lib/text.ts';
	import { usePrompt } from '../social/prompt';
	const ask = usePrompt();
	import { tick, untrack } from 'svelte';
	import type { CraftskyAPI, Json, JsonObject } from '#lib/api/client.ts';
	import { label, options, record, str } from './model.ts';
	import { hashtagsInText } from './facet-input';
	import FacetTextInput from './FacetTextInput.svelte';
	import FieldCounter from './FieldCounter.svelte';
	import { recordTextLimit } from './text-limit';
	import Icon from '../shell/Icon.svelte';
	import ProjectDetail from './ProjectDetail.svelte';
	import MultiSelect from '#lib/MultiSelect.svelte';
	import SingleSelect from '#lib/SingleSelect.svelte';
	import toolSizes from './tool-sizes.json';
	let {
		project,
		api,
		owner,
		disabled = false,
		part = 'all',
		precedingHashtags = [],
		onchange
	}: {
		project: JsonObject;
		api: CraftskyAPI;
		owner: string;
		disabled?: boolean;
		part?: 'main' | 'details' | 'all';
		precedingHashtags?: string[];
		onchange: (project: JsonObject) => void;
	} = $props();
	let activeDetail = $state<'pattern' | 'materials' | 'craft'>();
	let materialText = $state(''),
		materialError = $state('');
	let editingMaterial = $state<JsonObject>();
	let materialEntry = $state<HTMLDivElement>();
	$effect(() => {
		void owner;
		untrack(() => {
			activeDetail = undefined;
			materialText = '';
			materialError = '';
			editingMaterial = undefined;
		});
	});
	const common = $derived(record(project.common));
	const materialItems = $derived(
		Array.isArray(common.materials) ? common.materials.map(record) : []
	);
	const additionalItems = $derived(
		Array.isArray(common.additionalCrafts)
			? common.additionalCrafts.map(record)
			: []
	);
	const additionalValues = $derived(
		additionalItems.map((value) => str(value.craftType)).filter(Boolean)
	);
	const pattern = $derived(record(common.pattern));
	const tagPrefix = $derived([
		...precedingHashtags,
		...(Array.isArray(common.tags)
			? common.tags.filter((tag): tag is string => typeof tag === 'string')
			: [])
	]);
	const patternPrefix = $derived([
		...tagPrefix,
		...hashtagsInText(str(pattern.name))
	]);
	const publisherPrefix = $derived([
		...patternPrefix,
		...hashtagsInText(str(pattern.designer))
	]);
	const materialPrefix = $derived([
		...publisherPrefix,
		...hashtagsInText(str(pattern.publisher)),
		...materialItems
			.slice(
				0,
				editingMaterial
					? materialItems.indexOf(editingMaterial)
					: materialItems.length
			)
			.flatMap((item) => hashtagsInText(str(item.text)))
	]);

	const details = $derived(record(project.details));
	const craft = $derived(str(common.craftType));
	const shortCraft = $derived(craft.split('#').at(-1) || '');
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
		onchange({
			...project,
			details: {
				...details,
				$type: `social.craftsky.project.${shortCraft}#details`,
				projectType: value || undefined,
				projectSubtype: undefined
			}
		});
	}
	function changeCraft(value: string) {
		if (disabled) return;
		onchange({
			...project,
			common: {
				...common,
				craftType: value,
				...(Array.isArray(common.additionalCrafts)
					? {
							additionalCrafts: additionalItems.filter(
								(item) => item.craftType !== value
							)
						}
					: {})
			},
			details: undefined
		});
	}
	function withCurrent(
		choices: { id: string; label: string }[],
		value: string
	) {
		return value && !choices.some((option) => option.id === value)
			? [...choices, { id: value, label: label(value) }]
			: choices;
	}
	function selected(key: string): string[] {
		const values = common[key];
		return Array.isArray(values)
			? values.filter((value): value is string => typeof value === 'string')
			: [];
	}
	function selectOptions(family: string, values: string[], excluded = '') {
		const choices = options(family)
			.filter((item) => item.id !== excluded)
			.map((item) => ({ id: item.id, label: item.label }));
		for (const value of values)
			if (!choices.some((item) => item.id === value))
				choices.push({ id: value, label: label(value) });
		return choices.sort((a, b) => a.label.localeCompare(b.label));
	}
	function replaceSelection(key: string, values: string[]) {
		if (disabled || values.length > 10) return false;
		updateCommon(key, [...new Set(values)]);
		return true;
	}
	function additional(values: string[]) {
		if (disabled || values.length > 10) return false;
		updateCommon(
			'additionalCrafts',
			[...new Set(values)].map(
				(value) =>
					additionalItems.find((item) => item.craftType === value) || {
						craftType: value
					}
			)
		);
		return true;
	}
	function addMaterial() {
		if (disabled) return;
		const text = materialText.trim();
		if (!text) return;
		if (
			graphemeCount(text) > 100 ||
			new TextEncoder().encode(text).length > 1000
		) {
			materialError = 'Materials must be 100 characters or fewer.';
			return;
		}
		const index = editingMaterial ? materialItems.indexOf(editingMaterial) : -1;
		if (index < 0 && materialItems.length >= 10) {
			materialError = 'Choose up to ten materials.';
			return;
		}
		const material: JsonObject = {
			...(index >= 0 ? editingMaterial : {}),
			text
		};
		if (material.text !== editingMaterial?.text) delete material.facets;
		const next = [...materialItems];
		if (index < 0) next.push(material);
		else next[index] = material;
		updateCommon('materials', next);
		materialText = '';
		materialError = '';
		editingMaterial = undefined;
		focusMaterial();
	}
	function focusMaterial() {
		const account = owner;
		void tick().then(() => {
			const input = materialEntry?.querySelector('textarea');
			if (account !== owner || disabled || !input?.isConnected) return;
			input.focus();
			input.setSelectionRange(input.value.length, input.value.length);
		});
	}
	function editMaterial(material: JsonObject) {
		if (disabled || materialText) return;
		editingMaterial = material;
		materialText = str(material.text);
		materialError = '';
		focusMaterial();
	}
	function removeMaterial(material: JsonObject) {
		if (disabled) return;
		updateCommon(
			'materials',
			materialItems.filter((item) => item !== material)
		);
		if (editingMaterial === material) {
			editingMaterial = undefined;
			materialText = '';
		}
		materialError = '';
	}
	function moveMaterial(material: JsonObject, direction: number) {
		if (disabled) return;
		const index = materialItems.indexOf(material),
			destination = index + direction;
		if (index < 0 || destination < 0 || destination >= materialItems.length)
			return;
		const next = [...materialItems];
		[next[index], next[destination]] = [next[destination], next[index]];
		updateCommon('materials', next);
	}
	async function closeMaterials() {
		if (
			materialText !== str(editingMaterial?.text) &&
			!(await ask({
				title: 'Discard material changes?',
				detail:
					"Your current material input won't be saved. Materials you've already added will be kept.",
				confirmLabel: 'Discard',
				cancelLabel: 'Keep editing',
				danger: true
			}))
		)
			return false;
		materialText = '';
		editingMaterial = undefined;
		materialError = '';
		return true;
	}

	function gauge(key: string, value: string) {
		const values = { ...record(details.gauge) };
		if (value === '') delete values[key];
		else values[key] = key === 'unit' ? value : Number(value);
		updateDetail('gauge', Object.keys(values).length ? values : undefined);
	}
</script>

<section
	class="composer-section project-editor"
	class:project-main={part === 'main'}
>
	{#if part !== 'details'}
		<div class="project-basics">
			<SingleSelect
				label="Primary craft"
				required
				{disabled}
				value={craft}
				options={withCurrent(options('craft'), craft)}
				onchange={changeCraft}
			/>
		</div>
		<MultiSelect
			label="Additional crafts"
			placeholder="Find other crafts used in this project"
			values={additionalValues}
			options={selectOptions('craft', additionalValues, craft)}
			maximum={10}
			{disabled}
			onchange={additional}
		/>
		<div>
			<label
				>Project title<input
					use:recordTextLimit={{ graphemes: 200, bytes: 500 }}
					value={str(common.title)}
					oninput={(e) =>
						updateCommon('title', e.currentTarget.value || undefined)}
					placeholder="Add a short project title"
				/></label
			><FieldCounter value={str(common.title)} maximum={200} />
		</div>

		<SingleSelect
			label="Status"
			{disabled}
			value={str(common.status)}
			options={withCurrent(options('status'), str(common.status))}
			onchange={(value) => updateCommon('status', value || undefined)}
		/>
		<FacetTextInput
			{api}
			{disabled}
			label="Pattern tag or name"
			maximum={200}
			maximumBytes={500}
			precedingHashtags={tagPrefix}
			value={str(pattern.name)}
			allowed={['hashtag']}
			onchange={(value) => updatePattern('name', value || undefined)}
		/>
		<div>
			<label
				>Duration<input
					use:recordTextLimit={{ graphemes: 100, bytes: 500 }}
					value={str(common.duration)}
					oninput={(e) =>
						updateCommon('duration', e.currentTarget.value || undefined)}
					placeholder="e.g. 2 days or 4 weeks"
				/></label
			><FieldCounter value={str(common.duration)} maximum={100} />
		</div>
	{/if}
	{#if part !== 'main'}
		<div class="more-details-heading">
			<h2>Add more details</h2>
			<p class="muted">
				Optional details help other crafters discover and understand your
				project.
			</p>
		</div>
		<ProjectDetail
			title="Pattern details"
			description="Add designer, publisher, link and difficulty"
			icon="pattern"
			open={activeDetail === 'pattern'}
			{disabled}
			onopen={() => (activeDetail = 'pattern')}
			onclose={() => (activeDetail = undefined)}
		>
			<div class="stack">
				<div class="fields">
					<FacetTextInput
						{api}
						{disabled}
						label="Designer"
						maximum={200}
						maximumBytes={500}
						placeholder="Add pattern designer"
						precedingHashtags={patternPrefix}
						value={str(pattern.designer)}
						allowed={['mention']}
						onchange={(value) => updatePattern('designer', value || undefined)}
					/><FacetTextInput
						{api}
						{disabled}
						label="Publisher"
						maximum={200}
						maximumBytes={500}
						placeholder="Add pattern publisher"
						precedingHashtags={publisherPrefix}
						value={str(pattern.publisher)}
						allowed={['mention']}
						onchange={(value) => updatePattern('publisher', value || undefined)}
					/>
				</div>
				<label
					>Link<input
						type="url"
						value={str(pattern.url)}
						oninput={(e) =>
							updatePattern('url', e.currentTarget.value || undefined)}
						placeholder="https://example.com/pattern"
					/></label
				><SingleSelect
					label="Difficulty"
					{disabled}
					value={str(pattern.difficulty)}
					options={withCurrent(options('difficulty'), str(pattern.difficulty))}
					onchange={(value) => updatePattern('difficulty', value || undefined)}
				/>
				<label class="switch-row">
					<span
						><strong>Self-drafted pattern</strong><small
							>I drafted this pattern</small
						></span
					>
					<input
						type="checkbox"
						role="switch"
						aria-label="Self-drafted pattern"
						{disabled}
						checked={pattern.selfDrafted === true}
						onchange={(event) =>
							updatePattern('selfDrafted', event.currentTarget.checked)}
					/>
				</label>
			</div>
		</ProjectDetail>
		<ProjectDetail
			title="Materials and style"
			description="Help others discover projects like yours"
			icon="palette"
			open={activeDetail === 'materials'}
			beforeClose={closeMaterials}
			{disabled}
			onopen={() => (activeDetail = 'materials')}
			onclose={() => (activeDetail = undefined)}
		>
			<section class="material-field" aria-label="Materials">
				<h3>Materials</h3>
				{#each materialItems as material, index}{#if material !== editingMaterial}<div
							class="material-chip"
						>
							<span>{str(material.text)}</span>
							<div class="material-actions">
								<button
									type="button"
									class="material-up"
									aria-label={`Move material ${index + 1} up`}
									disabled={disabled || index === 0}
									onclick={() => moveMaterial(material, -1)}
									><Icon name="down" size={16} /></button
								><button
									type="button"
									aria-label={`Move material ${index + 1} down`}
									disabled={disabled || index === materialItems.length - 1}
									onclick={() => moveMaterial(material, 1)}
									><Icon name="down" size={16} /></button
								><button
									type="button"
									aria-label={`Edit material ${str(material.text)}`}
									title={materialText
										? 'Add or clear the current material first'
										: 'Edit material'}
									disabled={disabled || !!materialText}
									onclick={() => editMaterial(material)}
									><Icon name="edit" size={18} /></button
								><button
									type="button"
									aria-label={`Remove material ${str(material.text)}`}
									{disabled}
									onclick={() => removeMaterial(material)}
									><Icon name="close" size={18} /></button
								>
							</div>
						</div>{/if}{/each}
				<div class="material-entry" bind:this={materialEntry}>
					<FacetTextInput
						{api}
						{disabled}
						label="Materials"
						precedingHashtags={materialPrefix}
						value={materialText}
						placeholder="Add material"
						onchange={(value) => {
							materialText = value.replace(/[\r\n]+/g, ' ');
						}}
						onsubmit={addMaterial}
					/><button
						type="button"
						class="material-add"
						disabled={disabled || !materialText.trim()}
						onclick={addMaterial}>Add</button
					>
				</div>
				{#if materialError}<p class="material-error" role="alert">
						{materialError}
					</p>{/if}
			</section>
			<MultiSelect
				label="Colours"
				placeholder="Search colours"
				values={selected('colors')}
				options={selectOptions('color', selected('colors'))}
				maximum={10}
				{disabled}
				onchange={(values) => replaceSelection('colors', values)}
			/>
			<MultiSelect
				label="Design"
				placeholder="Search designs"
				values={selected('designTags')}
				options={selectOptions('designTag', selected('designTags'))}
				maximum={10}
				{disabled}
				onchange={(values) => replaceSelection('designTags', values)}
			/>
		</ProjectDetail>
		{#if ['sewing', 'knitting', 'crochet', 'quilting'].includes(shortCraft)}
			<ProjectDetail
				title={label(craft) + ' details'}
				description="Add project type and craft-specific details"
				icon="adjustments"
				open={activeDetail === 'craft'}
				{disabled}
				onopen={() => (activeDetail = 'craft')}
				onclose={() => (activeDetail = undefined)}
			>
				<div class="stack">
					<div class="fields">
						<SingleSelect
							label="Project type"
							{disabled}
							value={str(details.projectType)}
							options={withCurrent(
								options('projectType', craft).sort((a, b) =>
									a.label.localeCompare(b.label)
								),
								str(details.projectType)
							)}
							onchange={projectType}
						/><SingleSelect
							label="Project subtype"
							disabled={disabled ||
								!details.projectType ||
								!options('projectSubtype', craft, str(details.projectType))
									.length}
							value={str(details.projectSubtype)}
							options={details.projectType
								? withCurrent(
										options(
											'projectSubtype',
											craft,
											str(details.projectType)
										).sort((a, b) => a.label.localeCompare(b.label)),
										str(details.projectSubtype)
									)
								: []}
							onchange={(value) =>
								updateDetail('projectSubtype', value || undefined)}
						/>
					</div>
					{#if shortCraft === 'sewing'}<div>
							<label
								>Size made<input
									use:recordTextLimit={{ graphemes: 100, bytes: 500 }}
									placeholder="e.g. Medium or custom measurements"
									value={str(details.sizeMade)}
									oninput={(e) =>
										updateDetail(
											'sizeMade',
											e.currentTarget.value || undefined
										)}
								/></label
							><FieldCounter value={str(details.sizeMade)} maximum={100} />
						</div>
						<div>
							<label
								>Fit notes<textarea
									rows="3"
									use:recordTextLimit={{ graphemes: 600, bytes: 6000 }}
									placeholder="Add fit notes"
									value={str(details.fitNotes)}
									oninput={(e) =>
										updateDetail(
											'fitNotes',
											e.currentTarget.value || undefined
										)}></textarea></label
							><FieldCounter value={str(details.fitNotes)} maximum={600} />
						</div>{/if}
					{#if textile}<div class="fields">
							<SingleSelect
								label="Yarn weight"
								{disabled}
								value={str(details.yarnWeight)}
								options={withCurrent(
									options('yarnWeight'),
									str(details.yarnWeight)
								)}
								onchange={(value) =>
									updateDetail('yarnWeight', value || undefined)}
							/><SingleSelect
								label={shortCraft === 'knitting' ? 'Needle size' : 'Hook size'}
								{disabled}
								value={str(
									details[
										shortCraft === 'knitting' ? 'needleSizeMm' : 'hookSizeMm'
									]
								)}
								options={withCurrent(
									toolSizes[shortCraft === 'knitting' ? 'needle' : 'hook'].map(
										(value) => ({ id: value, label: value })
									),
									str(
										details[
											shortCraft === 'knitting' ? 'needleSizeMm' : 'hookSizeMm'
										]
									)
								)}
								onchange={(value) =>
									updateDetail(
										shortCraft === 'knitting' ? 'needleSizeMm' : 'hookSizeMm',
										value || undefined
									)}
							/>
						</div>
						{#if shortCraft === 'knitting'}
							<div>
								<label
									>Size made<input
										use:recordTextLimit={{ graphemes: 100, bytes: 500 }}
										placeholder="Add size made"
										value={str(details.finishedSize)}
										oninput={(e) =>
											updateDetail(
												'finishedSize',
												e.currentTarget.value || undefined
											)}
									/></label
								><FieldCounter
									value={str(details.finishedSize)}
									maximum={100}
								/>
							</div>{/if}
						<div class="gauge-fields">
							<div class="fields gauge">
								{#each [['stitches', 'Stitches'], ['rows', 'Rows'], ['measurement', 'Measurement']] as [key, name]}<label
										>Gauge {name.toLowerCase()}<input
											placeholder={name}
											type="number"
											min="1"
											step="1"
											value={typeof record(details.gauge)[key] === 'number'
												? Number(record(details.gauge)[key])
												: ''}
											oninput={(e) => gauge(key, e.currentTarget.value)}
										/></label
									>{/each}<SingleSelect
									label="Gauge unit"
									{disabled}
									value={str(record(details.gauge).unit)}
									options={[
										{ id: 'cm', label: 'cm' },
										{ id: 'in', label: 'in' }
									]}
									onchange={(value) => gauge('unit', value)}
								/>
							</div>
						</div>
						{#if shortCraft === 'crochet'}
							<div>
								<label
									>Finished size<input
										use:recordTextLimit={{ graphemes: 100, bytes: 500 }}
										placeholder="Add finished size"
										value={str(details.finishedSize)}
										oninput={(e) =>
											updateDetail(
												'finishedSize',
												e.currentTarget.value || undefined
											)}
									/></label
								><FieldCounter
									value={str(details.finishedSize)}
									maximum={100}
								/>
							</div>{/if}{/if}
					{#if shortCraft === 'quilting'}<div>
							<label
								>Size<input
									use:recordTextLimit={{ graphemes: 100, bytes: 500 }}
									placeholder="Add finished size"
									value={str(details.size)}
									oninput={(e) =>
										updateDetail('size', e.currentTarget.value || undefined)}
								/></label
							><FieldCounter value={str(details.size)} maximum={100} />
						</div>
						{#each [['piecingTechnique', 'Piecing technique'], ['quiltingMethod', 'Quilting method']] as [key, name]}<SingleSelect
								label={name}
								{disabled}
								value={str(details[key])}
								options={withCurrent(options(key, craft), str(details[key]))}
								onchange={(value) => updateDetail(key, value || undefined)}
							/>{/each}{/if}
				</div>
			</ProjectDetail>{/if}
	{/if}
</section>

<style>
	.switch-row {
		display: flex;
		flex-direction: row;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
	}
	.switch-row small {
		display: block;
		font-weight: 400;
		color: var(--ink-3);
	}
	.project-editor :global(.fields) {
		grid-template-columns: 1fr;
	}
	.project-editor :global(.gauge-fields) {
		display: grid;
		gap: 16px;
	}
	.project-editor
		:global(.native-multi-select .selection-values input[type='search']) {
		border: 0;
		border-radius: 0;
		background: transparent;
		padding: 0;
		min-height: 30px;
		width: 100px;
		min-width: 50px;
		flex: 1;
		box-shadow: none;
	}
	.project-editor :global(.native-multi-select .select-option) {
		display: flex;
		flex-direction: row;
		align-items: center;
		gap: 16px;
		margin: 0;
		padding: 10px 14px;
		font-weight: 400;
	}
	.material-field {
		display: grid;
		gap: 8px;
	}
	.material-field h3 {
		font:
			700 16px/1.5 Outfit,
			sans-serif;
		margin: 0;
	}
	.material-chip {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 12px 8px 12px 16px;
		background: var(--surface);
		border: 1.5px solid var(--ink-3);
		border-radius: 14px;
	}
	.material-chip > span {
		flex: 1;
		min-width: 0;
		overflow-wrap: anywhere;
		font-size: 16px;
		font-weight: 600;
	}
	.material-actions {
		display: flex;
		flex: none;
		align-items: center;
	}
	.material-actions button {
		display: grid;
		place-items: center;
		width: 28px;
		height: 32px;
		min-height: 32px;
		padding: 4px;
		border: 0;
		border-radius: 50%;
		background: transparent;
		box-shadow: none;
		color: var(--ink);
	}
	.material-up :global(svg) {
		transform: rotate(180deg);
	}
	.material-entry {
		position: relative;
	}
	.material-entry :global(.facet-input > label) {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
	}
	.material-entry :global(.facet-input textarea) {
		padding-right: 64px;
		min-height: 56px;
		border-radius: 14px;
	}
	.material-add {
		position: absolute;
		right: 8px;
		top: 6px;
		min-height: 44px;
		padding: 8px;
		color: var(--primary);
		background: transparent;
		border: 0;
		box-shadow: none;
	}
	.material-error {
		font-size: 14px;
		color: var(--red-ink);
		margin: 0;
	}
</style>
