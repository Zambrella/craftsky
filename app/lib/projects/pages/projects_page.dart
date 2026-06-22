import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/providers/project_feed_provider.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/shared/widgets/auto_paginated_list_view.dart';
import 'package:craftsky_app/shared/widgets/sort_menu_button.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

// TODO(Agent): Split this down so widgets are in their own files.

class ProjectsPage extends ConsumerStatefulWidget {
  const ProjectsPage({super.key});

  @override
  ConsumerState<ProjectsPage> createState() => _ProjectsPageState();
}

class _ProjectsPageState extends ConsumerState<ProjectsPage> {
  String _craftType = ProjectOptionCatalogs.defaultSupportedCraftTokens.first;
  SearchSort _sort = SearchSort.chronological;
  ProjectBrowseFilters _filters = const ProjectBrowseFilters();

  ProjectBrowseQuery get _query => ProjectBrowseQuery(
    craftTypes: [_craftType],
    filters: _filters,
    sort: _sort,
  );

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final projectFeedAsync = ref.watch(projectFeedProvider(_query));
    return DefaultTabController(
      length: ProjectOptionCatalogs.craftTypes.length,
      child: Scaffold(
        appBar: AppBar(title: Text(l10n.projectsTitle)),
        body: SafeArea(
          child: Column(
            children: [
              TabBar(
                isScrollable: true,
                tabAlignment: TabAlignment.start,
                onTap: (index) {
                  setState(() {
                    _craftType = ProjectOptionCatalogs.craftTypes[index].value;
                    _filters = const ProjectBrowseFilters();
                  });
                },
                tabs: [
                  for (final option in ProjectOptionCatalogs.craftTypes)
                    Tab(text: option.label),
                ],
              ),
              Padding(
                padding: EdgeInsets.fromLTRB(
                  spacing.sp4,
                  spacing.sp3,
                  spacing.sp4,
                  spacing.sp2,
                ),
                child: Row(
                  children: [
                    FilledButton.icon(
                      onPressed: _openFilters,
                      icon: const Icon(Icons.tune),
                      label: Text(l10n.projectsFilterAction),
                    ),
                    const Spacer(),
                    SortMenuButton<SearchSort>(
                      selectedValue: _sort,
                      options: _sortOptions(l10n),
                      onChanged: (sort) => setState(() => _sort = sort),
                    ),
                  ],
                ),
              ),
              _ActiveFilterChips(
                filters: _filters,
                onRemove: (family, value) => setState(() {
                  _filters = _filters.withoutValue(family, value);
                }),
                onClear: () => setState(() {
                  _filters = const ProjectBrowseFilters();
                }),
              ),
              Expanded(
                child: switch (projectFeedAsync) {
                  AsyncValue(:final value?) => _ProjectPostList(
                    posts: value.items,
                    isLoadingMore: projectFeedAsync.isLoading,
                    hasLoadMoreError: projectFeedAsync.hasError,
                    onNearEnd: () => ref
                        .read(projectFeedProvider(_query).notifier)
                        .loadMore(),
                  ),
                  _ when projectFeedAsync.hasError => _ProjectErrorView(
                    onRetry: () => ref.invalidate(projectFeedProvider(_query)),
                  ),
                  _ => const Center(child: StitchProgressIndicator()),
                },
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _openFilters() async {
    // TODO(Agent): Change this to the same page type that the post composer and
    // profile editor uses
    final filters = await showModalBottomSheet<ProjectBrowseFilters>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      builder: (context) => _ProjectFilterSheet(
        craftType: _craftType,
        initialFilters: _filters,
      ),
    );
    if (filters == null || !mounted) return;
    setState(() => _filters = filters);
  }

  List<SortMenuOption<SearchSort>> _sortOptions(AppLocalizations l10n) => [
    SortMenuOption(
      value: SearchSort.chronological,
      label: l10n.searchSortNewest,
      description: l10n.searchSortNewestDescription,
    ),
    SortMenuOption(
      value: SearchSort.popular,
      label: l10n.searchSortPopular,
      description: l10n.searchSortPopularDescription,
    ),
  ];
}

class _ProjectPostList extends StatelessWidget {
  const _ProjectPostList({
    required this.posts,
    required this.isLoadingMore,
    required this.hasLoadMoreError,
    required this.onNearEnd,
  });

  final List<Post> posts;
  final bool isLoadingMore;
  final bool hasLoadMoreError;
  final VoidCallback onNearEnd;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return AutoPaginatedListView(
      itemCount: posts.length,
      emptyText: l10n.projectsEmpty,
      isLoadingMore: isLoadingMore,
      hasLoadMoreError: hasLoadMoreError,
      onNearEnd: onNearEnd,
      itemBuilder: (context, index) {
        final post = posts[index];
        return PostCard(
          post: post,
          onTap: () => PostThreadRoute(
            did: post.author.did,
            rkey: post.rkey,
          ).push<void>(context),
        );
      },
    );
  }
}

class _ProjectErrorView extends StatelessWidget {
  const _ProjectErrorView({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Center(
      child: TextButton.icon(
        onPressed: onRetry,
        icon: const Icon(Icons.refresh),
        label: Text(l10n.projectsLoadError),
      ),
    );
  }
}

class _ActiveFilterChips extends StatelessWidget {
  const _ActiveFilterChips({
    required this.filters,
    required this.onRemove,
    required this.onClear,
  });

  final ProjectBrowseFilters filters;
  final void Function(ProjectBrowseFilterFamily family, String value) onRemove;
  final VoidCallback onClear;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final chips = _activeFilters(filters);
    if (chips.isEmpty) return const SizedBox.shrink();
    return Padding(
      padding: EdgeInsets.fromLTRB(spacing.sp4, 0, spacing.sp4, spacing.sp2),
      child: Align(
        alignment: Alignment.centerLeft,
        child: Wrap(
          spacing: spacing.sp2,
          runSpacing: spacing.sp2,
          children: [
            for (final chip in chips)
              InputChip(
                label: Text(chip.label),
                onDeleted: () => onRemove(chip.family, chip.value),
              ),
            TextButton(
              onPressed: onClear,
              child: Text(l10n.projectsClearFiltersAction),
            ),
          ],
        ),
      ),
    );
  }
}

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

class _FilterChipData {
  const _FilterChipData({
    required this.family,
    required this.value,
    required this.label,
  });

  final ProjectBrowseFilterFamily family;
  final String value;
  final String label;
}

List<_FilterChipData> _activeFilters(ProjectBrowseFilters filters) => [
  for (final value in filters.projectType)
    _FilterChipData(
      family: ProjectBrowseFilterFamily.projectType,
      value: value,
      label: _optionLabel(ProjectOptionCatalogs.projectTypes, value),
    ),
  for (final value in filters.patternDifficulty)
    _FilterChipData(
      family: ProjectBrowseFilterFamily.patternDifficulty,
      value: value,
      label: _optionLabel(ProjectOptionCatalogs.patternDifficulties, value),
    ),
  for (final value in filters.color)
    _FilterChipData(
      family: ProjectBrowseFilterFamily.color,
      value: value,
      label: _optionLabel(ProjectOptionCatalogs.colours, value),
    ),
  for (final value in filters.material)
    _FilterChipData(
      family: ProjectBrowseFilterFamily.material,
      value: value,
      label: value,
    ),
  for (final value in filters.designTag)
    _FilterChipData(
      family: ProjectBrowseFilterFamily.designTag,
      value: value,
      label: _optionLabel(ProjectOptionCatalogs.designTags, value),
    ),
  for (final value in filters.projectTag)
    _FilterChipData(
      family: ProjectBrowseFilterFamily.projectTag,
      value: value,
      label: value,
    ),
];

String _optionLabel(Iterable<ProjectOption> options, String value) {
  for (final option in options) {
    if (option.value == value) return option.label;
  }
  final hash = value.lastIndexOf('#');
  if (hash >= 0 && hash < value.length - 1) return value.substring(hash + 1);
  return value;
}
