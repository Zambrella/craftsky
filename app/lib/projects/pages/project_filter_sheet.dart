part of 'projects_page.dart';

class _ProjectFilterSheet extends StatefulWidget {
  const _ProjectFilterSheet({
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
    final subtypeOptions = ProjectOptionCatalogs.projectSubtypesForTypes(
      craftToken: widget.craftType,
      projectTypeTokens: _filters.projectType,
    );
    final usesYarn =
        widget.craftType == ProjectOptionCatalogs.knittingCraftToken ||
        widget.craftType == ProjectOptionCatalogs.crochetCraftToken;
    final isQuilting =
        widget.craftType == ProjectOptionCatalogs.quiltingCraftToken;
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
        body: ListView(
          padding: EdgeInsets.all(spacing.sp4),
          children: [
            _OptionFilterGroup(
              title: l10n.projectsFilterProjectType,
              options: _alphabetizedOptions(
                ProjectOptionCatalogs.projectTypesForCraft(widget.craftType),
              ),
              selectedValues: _filters.projectType,
              onChanged: _replaceProjectTypes,
            ),
            _OptionFilterGroup(
              title: l10n.projectsFilterProjectSubtype,
              options: _alphabetizedOptions(subtypeOptions),
              selectedValues: _filters.projectSubtype,
              enabled: subtypeOptions.isNotEmpty,
              showWhenEmpty: true,
              onChanged: (values) => _replaceValues(
                ProjectBrowseFilterFamily.projectSubtype,
                values,
              ),
            ),
            _OptionFilterGroup(
              title: l10n.projectsFilterDifficulty,
              options: ProjectOptionCatalogs.patternDifficulties,
              selectedValues: _filters.patternDifficulty,
              onChanged: (values) => _replaceValues(
                ProjectBrowseFilterFamily.patternDifficulty,
                values,
              ),
            ),
            _OptionFilterGroup(
              title: l10n.projectsFilterColor,
              options: _alphabetizedOptions(ProjectOptionCatalogs.colours),
              selectedValues: _filters.color,
              onChanged: (values) => _replaceValues(
                ProjectBrowseFilterFamily.color,
                values,
              ),
            ),
            _OptionFilterGroup(
              title: l10n.projectsFilterDesignTag,
              options: _alphabetizedOptions(ProjectOptionCatalogs.designTags),
              selectedValues: _filters.designTag,
              onChanged: (values) => _replaceValues(
                ProjectBrowseFilterFamily.designTag,
                values,
              ),
            ),
            if (usesYarn)
              _OptionFilterGroup(
                title: l10n.projectsFilterYarnWeight,
                options: ProjectOptionCatalogs.yarnWeights,
                selectedValues: _filters.yarnWeight,
                onChanged: (values) => _replaceValues(
                  ProjectBrowseFilterFamily.yarnWeight,
                  values,
                ),
              ),
            if (isQuilting) ...[
              _OptionFilterGroup(
                title: l10n.projectsFilterPiecingTechnique,
                options: _alphabetizedOptions(
                  ProjectOptionCatalogs.quiltingPiecingTechniques,
                ),
                selectedValues: _filters.piecingTechnique,
                onChanged: (values) => _replaceValues(
                  ProjectBrowseFilterFamily.piecingTechnique,
                  values,
                ),
              ),
              _OptionFilterGroup(
                title: l10n.projectsFilterQuiltingMethod,
                options: _alphabetizedOptions(
                  ProjectOptionCatalogs.quiltingMethods,
                ),
                selectedValues: _filters.quiltingMethod,
                onChanged: (values) => _replaceValues(
                  ProjectBrowseFilterFamily.quiltingMethod,
                  values,
                ),
              ),
            ],
            _OptionFilterGroup(
              title: l10n.projectsFilterStatus,
              options: ProjectOptionCatalogs.statuses,
              selectedValues: _filters.status,
              onChanged: (values) => _replaceValues(
                ProjectBrowseFilterFamily.status,
                values,
              ),
            ),
            _FilterFieldPadding(
              child: CheckboxListTile(
                contentPadding: EdgeInsets.zero,
                title: Text(l10n.projectsFilterSelfDrafted),
                value: _filters.selfDrafted,
                onChanged: (value) => setState(
                  () => _filters = _filters.copyWith(
                    selfDrafted: value ?? false,
                  ),
                ),
              ),
            ),
          ],
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
    setState(() => _filters = _filters.withValues(family, values));
  }

  void _replaceProjectTypes(List<String> values) {
    setState(() {
      _filters = _filters
          .withValues(ProjectBrowseFilterFamily.projectType, values)
          .withValues(
            ProjectBrowseFilterFamily.projectSubtype,
            ProjectOptionCatalogs.retainValidSubtypes(
              craftToken: widget.craftType,
              projectTypeTokens: values,
              subtypeTokens: _filters.projectSubtype,
            ),
          );
    });
  }
}

class _OptionFilterGroup extends StatelessWidget {
  const _OptionFilterGroup({
    required this.title,
    required this.options,
    required this.selectedValues,
    required this.onChanged,
    this.enabled = true,
    this.showWhenEmpty = false,
  });

  final String title;
  final List<ProjectOption> options;
  final List<String> selectedValues;
  final ValueChanged<List<String>> onChanged;
  final bool enabled;
  final bool showWhenEmpty;

  @override
  Widget build(BuildContext context) {
    if (options.isEmpty && !showWhenEmpty) return const SizedBox.shrink();
    final l10n = AppLocalizations.of(context);
    return _FilterFieldPadding(
      child: CraftskySearchableMultiSelectInput<String>(
        label: title,
        options: _selectOptions(options),
        values: selectedValues,
        enabled: enabled,
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
  const _FilterFieldPadding({required this.child});

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

List<CraftskySelectOption<String>> _selectOptions(List<ProjectOption> options) {
  return [
    for (final option in options)
      CraftskySelectOption<String>(
        value: option.value,
        label: option.label,
        description: option.description,
      ),
  ];
}

List<ProjectOption> _alphabetizedOptions(Iterable<ProjectOption> options) {
  return options.toList()
    ..sort((left, right) => left.label.compareTo(right.label));
}
