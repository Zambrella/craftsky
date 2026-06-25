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
  final _materialController = TextEditingController();
  final _projectTagController = TextEditingController();

  @override
  void initState() {
    super.initState();
    _filters = widget.initialFilters;
  }

  @override
  void dispose() {
    _materialController.dispose();
    _projectTagController.dispose();
    super.dispose();
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
          title: Text(l10n.projectsFiltersTitle),
          leading: IconButton(
            icon: const Icon(Icons.close),
            onPressed: () => Navigator.of(context).pop(),
          ),
        ),
        body: ListView(
          padding: EdgeInsets.all(spacing.sp4),
          children: [
            Text(l10n.projectsCraftContext(craftLabel)),
            SizedBox(height: spacing.sp4),
            _OptionFilterGroup(
              title: l10n.projectsFilterProjectType,
              options: ProjectOptionCatalogs.projectTypesForCraft(
                widget.craftType,
              ),
              selectedValues: _filters.projectType,
              onToggle: (value) => _toggle(
                ProjectBrowseFilterFamily.projectType,
                value,
              ),
            ),
            _OptionFilterGroup(
              title: l10n.projectsFilterDifficulty,
              options: ProjectOptionCatalogs.patternDifficulties,
              selectedValues: _filters.patternDifficulty,
              onToggle: (value) =>
                  _toggle(ProjectBrowseFilterFamily.patternDifficulty, value),
            ),
            _OptionFilterGroup(
              title: l10n.projectsFilterColor,
              options: ProjectOptionCatalogs.colours,
              selectedValues: _filters.color,
              onToggle: (value) => _toggle(
                ProjectBrowseFilterFamily.color,
                value,
              ),
            ),
            _OptionFilterGroup(
              title: l10n.projectsFilterDesignTag,
              options: ProjectOptionCatalogs.designTags,
              selectedValues: _filters.designTag,
              onToggle: (value) => _toggle(
                ProjectBrowseFilterFamily.designTag,
                value,
              ),
            ),
            _FreeTextFilterGroup(
              title: l10n.projectsFilterMaterial,
              controller: _materialController,
              values: _filters.material,
              onAdd: (value) => _addFreeText(
                ProjectBrowseFilterFamily.material,
                value,
              ),
              onRemove: (value) => _remove(
                ProjectBrowseFilterFamily.material,
                value,
              ),
            ),
            _FreeTextFilterGroup(
              title: l10n.projectsFilterProjectTag,
              controller: _projectTagController,
              values: _filters.projectTag,
              onAdd: (value) => _addFreeText(
                ProjectBrowseFilterFamily.projectTag,
                value,
              ),
              onRemove: (value) => _remove(
                ProjectBrowseFilterFamily.projectTag,
                value,
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

  void _toggle(ProjectBrowseFilterFamily family, String value) {
    setState(() => _filters = _filters.toggleValue(family, value));
  }

  void _addFreeText(ProjectBrowseFilterFamily family, String raw) {
    final value = raw.trim();
    if (value.isEmpty) return;
    setState(() => _filters = _filters.withValue(family, value));
  }

  void _remove(ProjectBrowseFilterFamily family, String value) {
    setState(() => _filters = _filters.withoutValue(family, value));
  }
}

class _OptionFilterGroup extends StatelessWidget {
  const _OptionFilterGroup({
    required this.title,
    required this.options,
    required this.selectedValues,
    required this.onToggle,
  });

  final String title;
  final List<ProjectOption> options;
  final List<String> selectedValues;
  final ValueChanged<String> onToggle;

  @override
  Widget build(BuildContext context) {
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    if (options.isEmpty) return const SizedBox.shrink();
    return Padding(
      padding: EdgeInsets.only(bottom: spacing.sp5),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleMedium),
          SizedBox(height: spacing.sp2),
          Wrap(
            spacing: spacing.sp2,
            runSpacing: spacing.sp2,
            children: [
              for (final option in options)
                FilterChip(
                  label: Text(option.label),
                  selected: selectedValues.contains(option.value),
                  onSelected: (_) => onToggle(option.value),
                ),
            ],
          ),
        ],
      ),
    );
  }
}

class _FreeTextFilterGroup extends StatelessWidget {
  const _FreeTextFilterGroup({
    required this.title,
    required this.controller,
    required this.values,
    required this.onAdd,
    required this.onRemove,
  });

  final String title;
  final TextEditingController controller;
  final List<String> values;
  final ValueChanged<String> onAdd;
  final ValueChanged<String> onRemove;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    return Padding(
      padding: EdgeInsets.only(bottom: spacing.sp5),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleMedium),
          SizedBox(height: spacing.sp2),
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: controller,
                  decoration: InputDecoration(
                    hintText: l10n.projectsFreeTextHint,
                  ),
                  onSubmitted: _addAndClear,
                ),
              ),
              SizedBox(width: spacing.sp2),
              FilledButton(
                onPressed: () => _addAndClear(controller.text),
                child: Text(l10n.projectsAddFilterValueAction),
              ),
            ],
          ),
          if (values.isNotEmpty) ...[
            SizedBox(height: spacing.sp2),
            Wrap(
              spacing: spacing.sp2,
              runSpacing: spacing.sp2,
              children: [
                for (final value in values)
                  InputChip(
                    label: Text(value),
                    onDeleted: () => onRemove(value),
                  ),
              ],
            ),
          ],
        ],
      ),
    );
  }

  void _addAndClear(String value) {
    onAdd(value);
    controller.clear();
  }
}
