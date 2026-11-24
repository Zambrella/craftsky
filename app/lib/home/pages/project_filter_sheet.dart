part of 'home_page.dart';

class _ProjectFilterSheet extends StatefulWidget {
  const new({
    required this.craftType,
    required this.initialFilters,
  });

  final String craftType;
  final ProjectBrowseFilters initialFilters;

  @override
  State<_ProjectFilterSheet> createState() => _ProjectFilterSheetState();
}

class _ProjectFilterSheetState extends State<_ProjectFilterSheet> {
  late ProjectBrowseFilters _filters;

  @override
  void initState() {
    super.initState();
    _filters = widget.initialFilters;
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final craftLabel = _optionLabel(
      ProjectOptionCatalogs.craftTypes,
      widget.craftType,
    );
    final itemOptions = projectItemSelectOptions(widget.craftType);
    final selectedItems = _filters.selectedProjectItems;
    final detailFilters = ProjectDetailFilters.forCraft(widget.craftType);
    return SizedBox.expand(
      child: Scaffold(
        appBar: AppBar(
          title: CraftIconLabel(
            craft: widget.craftType,
            label: l10n.projectsFiltersTitle(craftLabel),
            flexibleLabel: true,
          ),
          leading: IconButton(
            icon: const Icon(CraftskyIconsBold.close),
            onPressed: () => Navigator.of(context).pop(),
          ),
        ),
        body: SingleChildScrollView(
          padding: EdgeInsets.all(spacing.sp4),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              _OptionFilterField(
                title: l10n.projectsFilterStatus,
                options: ProjectOptionCatalogs.statuses,
                selectedValues: _filters.status,
                onChanged: (values) => _replaceValues(
                  ProjectBrowseFilterFamily.status,
                  values,
                ),
              ),
              _OptionFilterField(
                title: l10n.projectsFilterDifficulty,
                options: ProjectOptionCatalogs.patternDifficulties,
                selectedValues: _filters.patternDifficulty,
                onChanged: (values) => _replaceValues(
                  ProjectBrowseFilterFamily.patternDifficulty,
                  values,
                ),
              ),
              _FilterFieldPadding(
                child: CheckboxListTile(
                  contentPadding: EdgeInsets.zero,
                  title: Text(
                    l10n.projectsFilterSelfDrafted,
                    style: const TextStyle(fontWeight: FontWeight.bold),
                  ),
                  value: _filters.selfDrafted,
                  onChanged: (value) => setState(
                    () => _filters = _filters.copyWith(
                      selfDrafted: value ?? false,
                    ),
                  ),
                ),
              ),
              if (itemOptions.isNotEmpty || selectedItems.isNotEmpty)
                _FilterFieldPadding(
                  child: CraftskySearchableMultiSelectInput<String>(
                    label:
                        widget.craftType ==
                            ProjectOptionCatalogs.dyeingCraftToken
                        ? l10n.projectComposerDyedItemLabel
                        : l10n.projectsFilterProjectType,
                    options: itemOptions,
                    values: selectedItems,
                    selectedLabels: {
                      for (final value in selectedItems)
                        value: projectItemFilterLabel(itemOptions, value),
                    },
                    maxSelected: ProjectOptionCatalogs.maxFilterValuesPerFamily,
                    maxSelectedErrorText: l10n
                        .projectComposerMultiSelectMaxSelectedError(
                          ProjectOptionCatalogs.maxFilterValuesPerFamily,
                        ),
                    onChanged: (values) => _replaceValues(
                      ProjectBrowseFilterFamily.projectItem,
                      values,
                    ),
                  ),
                ),
              for (final filter in detailFilters)
                _OptionFilterField(
                  title: filter.labelFor(widget.craftType),
                  options: filter.allOptionsFor(widget.craftType),
                  selectedValues: _filters.valuesFor(filter.family),
                  onChanged: (values) => _replaceValues(filter.family, values),
                ),
            ],
          ),
        ),
        bottomNavigationBar: SafeArea(
          child: Padding(
            padding: EdgeInsets.all(spacing.sp4),
            child: Row(
              children: [
                TextButton(
                  onPressed: () =>
                      setState(() => _filters = const ProjectBrowseFilters()),
                  child: Text(l10n.projectsClearFiltersAction),
                ),
                const Spacer(),
                FilledButton(
                  onPressed: () => Navigator.of(context).pop(_filters),
                  child: Text(l10n.projectsApplyFiltersAction),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  void _replaceValues(ProjectBrowseFilterFamily family, List<String> values) {
    _setFilters(
      ProjectDetailFilters.retainCompatibleSelections(
        widget.craftType,
        _filters,
        _filters.withValues(family, values),
      ),
    );
  }

  void _setFilters(ProjectBrowseFilters filters) {
    final available =
        ProjectOptionCatalogs.maxFilterValues -
        (widget.craftType.isEmpty ? 0 : 1);
    if (filters.selectedValueCount > available) {
      // Re-apply the previous values because the picker updates optimistically.
      setState(() {});
      context.showError(
        AppLocalizations.of(context)
            .projectComposerMultiSelectMaxSelectedError(available),
      );
      return;
    }
    setState(() => _filters = filters);
  }
}

class _OptionFilterField extends StatelessWidget {
  const new({
    required this.title,
    required this.options,
    required this.selectedValues,
    required this.onChanged,
  });

  final String title;
  final List<ProjectOption> options;
  final List<String> selectedValues;
  final ValueChanged<List<String>> onChanged;

  @override
  Widget build(BuildContext context) {
    if (options.isEmpty) return const SizedBox.shrink();
    final l10n = AppLocalizations.of(context);
    return _FilterFieldPadding(
      child: CraftskySearchableMultiSelectInput<String>(
        label: title,
        options: _selectOptions(options),
        values: selectedValues,
        enabled: options.isNotEmpty,
        maxSelected: ProjectOptionCatalogs.maxFilterValuesPerFamily,
        maxSelectedErrorText: l10n.projectComposerMultiSelectMaxSelectedError(
          ProjectOptionCatalogs.maxFilterValuesPerFamily,
        ),
        onChanged: onChanged,
      ),
    );
  }
}

class _FilterFieldPadding extends StatelessWidget {
  const new({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    return Padding(
      padding: EdgeInsets.only(bottom: spacing.sp5),
      child: child,
    );
  }
}

List<CraftskySelectOption<String>> _selectOptions(
  List<ProjectOption> options,
) {
  return alphabetizedSelectOptions([
    for (final option in options)
      CraftskySelectOption<String>(
        value: option.value,
        label: option.label,
        group: option.group,
        aliases: option.aliases,
      ),
  ], lastValue: ProjectOptionCatalogs.otherProjectTypeToken);
}
