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
    return SizedBox.expand(
      child: Scaffold(
        appBar: AppBar(
          title: Text(l10n.projectsFiltersTitle(craftLabel)),
          leading: IconButton(
            icon: const Icon(Icons.close),
            onPressed: () => Navigator.of(context).pop(),
          ),
        ),
        body: ListView(
          padding: EdgeInsets.all(spacing.sp4),
          children: [
            _OptionFilterGroup(
              title: l10n.projectsFilterProjectType,
              options: ProjectOptionCatalogs.projectTypesForCraft(
                widget.craftType,
              ),
              selectedValues: _filters.projectType,
              onChanged: (values) => _replaceValues(
                ProjectBrowseFilterFamily.projectType,
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
              options: ProjectOptionCatalogs.colours,
              selectedValues: _filters.color,
              onChanged: (values) => _replaceValues(
                ProjectBrowseFilterFamily.color,
                values,
              ),
            ),
            _OptionFilterGroup(
              title: l10n.projectsFilterDesignTag,
              options: ProjectOptionCatalogs.designTags,
              selectedValues: _filters.designTag,
              onChanged: (values) => _replaceValues(
                ProjectBrowseFilterFamily.designTag,
                values,
              ),
            ),
            _FreeTextFilterGroup(
              title: l10n.projectsFilterMaterial,
              values: _filters.material,
              onChanged: (values) => _replaceValues(
                ProjectBrowseFilterFamily.material,
                values,
              ),
            ),
            _FreeTextFilterGroup(
              title: l10n.projectsFilterProjectTag,
              values: _filters.projectTag,
              onChanged: (values) => _replaceValues(
                ProjectBrowseFilterFamily.projectTag,
                values,
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
}

class _OptionFilterGroup extends StatelessWidget {
  const _OptionFilterGroup({
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
    return _FilterFieldPadding(
      child: CraftskySearchableMultiSelectInput<String>(
        label: title,
        options: _selectOptions(options),
        values: selectedValues,
        onChanged: onChanged,
      ),
    );
  }
}

class _FreeTextFilterGroup extends StatelessWidget {
  const _FreeTextFilterGroup({
    required this.title,
    required this.values,
    required this.onChanged,
  });

  final String title;
  final List<String> values;
  final ValueChanged<List<String>> onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return _FilterFieldPadding(
      child: CraftskyTokenInput(
        label: title,
        values: values,
        inputHintText: l10n.projectsFreeTextHint,
        addButtonLabel: l10n.projectsAddFilterValueAction,
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
