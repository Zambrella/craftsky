<script lang="ts">
	import { getContext, untrack } from 'svelte';
	import type {
		CraftskyAPI,
		Json,
		JsonObject,
		Profile
	} from '#lib/api/client.ts';
	import { findField, focusField } from '#lib/field-validation.ts';
	import {
		hasPatternContent,
		label,
		options,
		patternFields,
		patternRows,
		projectPatternLabels,
		record,
		str
	} from './model.ts';
	import { hashtagsInText } from './facet-input';
	import FacetTextInput from './FacetTextInput.svelte';
	import FieldCounter from './FieldCounter.svelte';
	import { recordTextLimit } from './text-limit';
	import Icon from '../shell/Icon.svelte';
	import { craftFeeds } from '../social/craft-feeds';
	import ProjectDetail from './ProjectDetail.svelte';
	import ProjectCraftFields from './ProjectCraftFields.svelte';
	import {
		craftChoice,
		otherProjectType,
		projectItemChoices,
		projectOptionFor,
		sortedProjectOptions
	} from './project-options';
	import MultiSelect from '#lib/MultiSelect.svelte';
	import SingleSelect from '#lib/SingleSelect.svelte';
	let {
		project,
		api,
		owner,
		disabled = false,
		precedingHashtags = [],
		onchange
	}: {
		project: JsonObject;
		api: CraftskyAPI;
		owner: string;
		disabled?: boolean;
		precedingHashtags?: string[];
		onchange: (project: JsonObject) => void;
	} = $props();
	const currentProfile = getContext<(() => Profile | undefined) | undefined>(
		'craftsky-profile'
	);
	let activeDetail = $state<'pattern' | 'craft'>();
	$effect(() => {
		void owner;
		untrack(() => {
			activeDetail = undefined;
		});
	});
	const common = $derived(record(project.common));
	const additionalItems = $derived(
		Array.isArray(common.additionalCrafts)
			? common.additionalCrafts.map(record)
			: []
	);
	const additionalValues = $derived(
		additionalItems.map((value) => str(value.craftType)).filter(Boolean)
	);
	const patterns = $derived(patternRows(common));
	const tagPrefix = $derived([
		...precedingHashtags,
		...(Array.isArray(common.tags)
			? common.tags.filter((tag): tag is string => typeof tag === 'string')
			: [])
	]);
	const patternTags = (pattern: JsonObject) =>
		patternFields.flatMap((field) => hashtagsInText(str(pattern[field])));
	function patternPrefix(index: number, field: (typeof patternFields)[number]) {
		return [
			...tagPrefix,
			...patterns.slice(0, index).flatMap(patternTags),
			...patternFields
				.slice(0, patternFields.indexOf(field))
				.flatMap((key) => hashtagsInText(str(patterns[index]?.[key])))
		];
	}
	const craft = $derived(str(common.craftType));
	const craftTitle = $derived(
		craft ? `${label(craft)} details` : 'Craft details'
	);
	const patternLabels = $derived(projectPatternLabels(craft));
	const craftsWithoutPatterns = new Set(['spinning', 'dyeing']);
	const showPatternDetails = $derived(
		!craftsWithoutPatterns.has(craftChoice(craft)?.profileId ?? '') ||
			additionalValues.some((value) => {
				const kind = craftChoice(value)?.profileId;
				return kind && !craftsWithoutPatterns.has(kind);
			}) ||
			patterns.some(hasPatternContent)
	);
	const orderedCrafts = $derived(craftFeeds(currentProfile?.()?.crafts));
	const primaryOptions = $derived(
		craft && !orderedCrafts.some((choice) => choice.id === craft)
			? [...orderedCrafts, { id: craft, label: label(craft) }]
			: orderedCrafts
	);
	const craftDetails = $derived(craftChoice(craft)?.detailsType);
	const details = $derived(record(project.details));
	const itemType = $derived(str(details.projectType));
	const itemSubtype = $derived(str(details.projectSubtype));
	const itemChoices = $derived(projectItemChoices(craft));
	const isOther = $derived(itemType === otherProjectType);
	const showItemFields = $derived(!!craftDetails && itemChoices.length > 0);
	const itemSubtypeLabel = $derived(
		isOther
			? itemSubtype
			: (projectOptionFor(itemSubtype)?.label ?? itemSubtype)
	);
	const itemSummary = $derived(
		showItemFields
			? [label(itemType), itemSubtypeLabel]
					.map((part) => part.trim())
					.filter(Boolean)
					.join(' · ')
			: ''
	);
	const itemSelection = $derived(itemID(itemType, isOther ? '' : itemSubtype));
	const itemPickerOptions = $derived.by(() => {
		const rows = itemChoices.map((option) => ({
			...option,
			id: option.parentId ? itemID(option.parentId, option.id) : option.id,
			type: option.parentId ?? option.id,
			subtype: option.parentId ? option.id : ''
		}));
		if (!itemSelection || rows.some(({ id }) => id === itemSelection))
			return rows;
		const retained = {
			id: itemSelection,
			label: itemSubtypeLabel || label(itemType),
			parentId: itemSubtype && itemType ? itemType : undefined,
			selectionLabel: itemSummary,
			type: itemType,
			subtype: itemSubtype
		};
		const parent = rows.findIndex(({ id }) => id === itemType);
		rows.splice(parent + 1, 0, retained);
		return rows;
	});
	function itemID(type: string, subtype: string) {
		return subtype ? JSON.stringify([type, subtype]) : type;
	}
	function updateCommon(key: string, value: Json | undefined) {
		onchange({ ...project, common: { ...common, [key]: value } });
	}
	function updatePatterns(value: JsonObject[]) {
		if (disabled || value.length > 5) return;
		onchange({
			...project,
			common: { ...common, pattern: undefined, patterns: value }
		});
	}
	function updatePattern(index: number, key: string, value: Json | undefined) {
		const next = [...patterns];
		const entry = { ...next[index], [key]: value };
		if (key === 'name' || key === 'designer' || key === 'publisher')
			delete entry[`${key}Facets`];
		next[index] = entry;
		updatePatterns(next);
	}
	function updateDetail(key: string, value: Json | undefined) {
		onchange({
			...project,
			details: {
				...details,
				$type: craftDetails,
				[key]: value
			}
		});
	}
	function selectProjectItem(value: string) {
		if (value === itemSelection) return;
		const option = itemPickerOptions.find(({ id }) => id === value);
		onchange({
			...project,
			details: {
				...details,
				$type: craftDetails,
				projectType: option?.type || undefined,
				projectSubtype: option?.subtype || undefined
			}
		});
	}
	function changeCraft(value: string) {
		if (disabled || value === craft) return;
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
		choices: {
			id: string;
			label: string;
			aliases?: readonly string[];
			description?: string;
		}[],
		value: string,
		retainedLabel = label(value)
	) {
		return sortedProjectOptions(
			value && !choices.some((option) => option.id === value)
				? [...choices, { id: value, label: retainedLabel }]
				: choices
		);
	}
	function additionalOptions() {
		const choices = options('craft').filter((item) => item.id !== craft);
		return sortedProjectOptions([
			...choices,
			...additionalValues
				.filter(
					(value) =>
						value !== craft && !choices.some((item) => item.id === value)
				)
				.map((value) => ({ id: value, label: label(value) }))
		]);
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
</script>

<section class="composer-section project-editor">
	<SingleSelect
		label="Primary craft"
		required
		{disabled}
		value={craft}
		options={primaryOptions}
		onchange={changeCraft}
	/>
	<div class="project-detail-links">
		<ProjectDetail
			title={craftTitle}
			description={itemSummary ||
				(showItemFields
					? 'Add other crafts and describe what you made'
					: 'Add other crafts used in this project')}
			icon="adjustments"
			open={activeDetail === 'craft'}
			disabled={disabled || !craft}
			onopen={() => (activeDetail = 'craft')}
			onclose={() => (activeDetail = undefined)}
		>
			{#if showItemFields}<div class="stack">
					<SingleSelect
						label={craftChoice(craft)?.profileId === 'dyeing'
							? 'What did you dye?'
							: 'Project type'}
						{disabled}
						value={itemSelection}
						options={itemPickerOptions}
						alwaysSearchable
						onchange={selectProjectItem}
					/>
					{#if isOther}
						<label>
							{craftChoice(craft)?.profileId === 'dyeing'
								? 'Custom item name'
								: 'What did you make?'}
							<input
								{disabled}
								use:recordTextLimit={{ graphemes: 100, bytes: 100 }}
								value={itemSubtype}
								onchange={(event) => {
									const trimmed = event.currentTarget.value.trim();
									if (trimmed !== itemSubtype)
										updateDetail('projectSubtype', trimmed || undefined);
								}}
								oninput={(event) => {
									updateDetail(
										'projectSubtype',
										event.currentTarget.value || undefined
									);
								}}
							/>
						</label>
					{/if}
					<ProjectCraftFields
						{craft}
						{details}
						{disabled}
						onchange={updateDetail}
					/>
				</div>{/if}
			<MultiSelect
				label="Additional crafts"
				placeholder="Find other crafts used in this project"
				values={additionalValues}
				options={additionalOptions()}
				maximum={10}
				{disabled}
				onchange={additional}
			/>
		</ProjectDetail>
		<ProjectDetail
			title={patternLabels.title}
			description={patternLabels.description}
			icon="pattern"
			open={activeDetail === 'pattern'}
			visible={showPatternDetails}
			{disabled}
			onopen={() => (activeDetail = 'pattern')}
			onclose={() => {
				activeDetail = undefined;
				if (!showPatternDetails) focusField(findField(craftTitle));
			}}
		>
			{#each patterns.length ? patterns : [{}] as pattern, index}
				<fieldset class="stack pattern-entry">
					<legend class="visually-hidden">
						{patterns.length > 1 ? `Pattern ${index + 1}` : 'Pattern'}
					</legend>
					{#if patterns.length > 1}
						<div class="pattern-heading">
							<h2>Pattern {index + 1}</h2>
							<button
								type="button"
								class="pattern-action pattern-remove"
								{disabled}
								aria-label={`Remove pattern ${index + 1}`}
								onclick={() =>
									updatePatterns(
										patterns.filter((_, current) => index !== current)
									)}><Icon name="trash" size={18} bold />Remove pattern</button
							>
						</div>
					{/if}
					<FacetTextInput
						{api}
						{disabled}
						label={patternLabels.name}
						placeholder="#PatternName"
						maximum={200}
						maximumBytes={500}
						precedingHashtags={patternPrefix(index, 'name')}
						value={str(pattern.name)}
						allowed={['hashtag']}
						onchange={(value) =>
							updatePattern(index, 'name', value || undefined)}
					/>
					<div class="fields">
						<FacetTextInput
							{api}
							{disabled}
							label="Designer"
							maximum={200}
							maximumBytes={500}
							placeholder="Add or @mention pattern designer"
							precedingHashtags={patternPrefix(index, 'designer')}
							value={str(pattern.designer)}
							allowed={['mention']}
							onchange={(value) =>
								updatePattern(index, 'designer', value || undefined)}
						/><FacetTextInput
							{api}
							{disabled}
							label="Publisher"
							maximum={200}
							maximumBytes={500}
							placeholder="Add or @mention pattern publisher"
							precedingHashtags={patternPrefix(index, 'publisher')}
							value={str(pattern.publisher)}
							allowed={['mention']}
							onchange={(value) =>
								updatePattern(index, 'publisher', value || undefined)}
						/>
					</div>
					<label
						>Link<input
							type="url"
							value={str(pattern.url)}
							oninput={(e) =>
								updatePattern(index, 'url', e.currentTarget.value || undefined)}
							placeholder="https://example.com/pattern"
						/></label
					><SingleSelect
						label="Difficulty"
						{disabled}
						value={str(pattern.difficulty)}
						options={withCurrent(
							options('difficulty'),
							str(pattern.difficulty)
						)}
						onchange={(value) =>
							updatePattern(index, 'difficulty', value || undefined)}
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
								updatePattern(
									index,
									'selfDrafted',
									event.currentTarget.checked
								)}
						/>
					</label>
				</fieldset>
			{/each}
			{#if patterns.length < 5}
				<button
					type="button"
					class="pattern-action"
					{disabled}
					onclick={() =>
						updatePatterns(patterns.length ? [...patterns, {}] : [{}, {}])}
					><Icon name="plus" size={18} bold />Add another pattern</button
				>
			{/if}
		</ProjectDetail>
	</div>

	<SingleSelect
		label="Status"
		required
		{disabled}
		value={str(common.status)}
		options={withCurrent(options('status'), str(common.status))}
		onchange={(value) => updateCommon('status', value)}
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
</section>

<style>
	.project-detail-links {
		display: grid;
		gap: 4px;
		min-width: 0;
	}
	.pattern-entry {
		min-inline-size: 0;
		margin: 0;
		padding: 0;
		border: 0;
	}
	.pattern-heading {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: center;
		gap: 4px 12px;
	}
	.pattern-heading h2 {
		margin: 0;
		font-size: 20px;
		line-height: 1.25;
	}
	.pattern-entry + .pattern-entry > .pattern-heading {
		padding-block-start: 12px;
		border-block-start: 1px solid var(--hairline);
	}
	button.pattern-action {
		--pattern-action-ink: var(--primary);
		justify-self: start;
		padding: 8px 16px;
		border: 0;
		background: transparent;
		box-shadow: none;
		color: var(--pattern-action-ink);
		font-size: 16px;
		font-weight: 600;
		line-height: 1.25;
	}
	button.pattern-remove {
		--pattern-action-ink: var(--red-ink);
	}
	button.pattern-action:hover:not(:disabled) {
		background: color-mix(in srgb, var(--pattern-action-ink) 8%, transparent);
	}
	.pattern-action :global(svg) {
		flex: none;
		inline-size: 18px;
		block-size: 18px;
	}
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
	.switch-row strong {
		font-weight: 700;
	}
	.project-editor :global(.fields) {
		grid-template-columns: 1fr;
	}
</style>
