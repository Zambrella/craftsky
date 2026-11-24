export interface SelectOption {
	id: string;
	label: string;
	aliases?: readonly string[];
	description?: string;
	group?: string;
	parentId?: string;
	selectionLabel?: string;
}

export function selectOptionMatches(option: SelectOption, query: string) {
	return [option.label, option.description || '', ...(option.aliases ?? [])]
		.join(' ')
		.toLocaleLowerCase()
		.includes(query.trim().toLocaleLowerCase());
}

export function matchingSelectOptions(options: SelectOption[], query: string) {
	if (!query.trim()) return options;
	const matches = options.filter((option) =>
		selectOptionMatches(option, query)
	);
	const matched = new Set(matches);
	const matchedParents = new Set(
		matches.filter(({ parentId }) => !parentId).map(({ id }) => id)
	);
	const context = new Set(matches.map(({ parentId }) => parentId));
	return options.filter(
		(option) =>
			matched.has(option) ||
			(option.parentId
				? matchedParents.has(option.parentId)
				: context.has(option.id))
	);
}

export function selectedOptionLabel(option: SelectOption) {
	return option.selectionLabel ?? option.label;
}
