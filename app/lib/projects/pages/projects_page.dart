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
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

part 'project_filter_sheet.dart';

class ProjectsPage extends ConsumerStatefulWidget {
  const ProjectsPage({super.key});

  @override
  ConsumerState<ProjectsPage> createState() => _ProjectsPageState();
}

class _ProjectsPageState extends ConsumerState<ProjectsPage> {
  var _selectedCraftIndex = 0;
  SearchSort _sort = SearchSort.chronological;
  ProjectBrowseFilters _filters = const ProjectBrowseFilters();

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    return DefaultTabController(
      length: ProjectOptionCatalogs.craftTypes.length,
      child: Scaffold(
        body: NestedScrollView(
          headerSliverBuilder: (context, innerBoxIsScrolled) => [
            SliverAppBar(
              title: Text(l10n.projectsTitle),
              pinned: true,
              actions: [
                OutlinedButton.icon(
                  onPressed: _openFilters,
                  icon: const Icon(Icons.tune, size: 18),
                  label: Text(l10n.projectsFilterAction),
                  style: _appBarControlStyle(context),
                ),
                SizedBox(width: spacing.sp2),
                Padding(
                  padding: EdgeInsetsDirectional.only(end: spacing.sp4),
                  child: SortMenuButton<SearchSort>(
                    selectedValue: _sort,
                    options: _sortOptions(l10n),
                    onChanged: (sort) => setState(() => _sort = sort),
                  ),
                ),
              ],
            ),
            SliverPersistentHeader(
              pinned: true,
              delegate: _ProjectCraftTabBarDelegate(
                onTap: (index) => setState(() {
                  _selectedCraftIndex = index;
                  _filters = const ProjectBrowseFilters();
                }),
              ),
            ),
            SliverToBoxAdapter(
              child: _ActiveFilterChips(
                filters: _filters,
                onRemove: (family, value) => setState(() {
                  _filters = _filters.withoutValue(family, value);
                }),
                onClear: () => setState(() {
                  _filters = const ProjectBrowseFilters();
                }),
              ),
            ),
          ],
          body: TabBarView(
            children: [
              for (final option in ProjectOptionCatalogs.craftTypes)
                _ProjectTabScrollView(
                  craftType: option.value,
                  filters: _filters,
                  sort: _sort,
                ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _openFilters() async {
    final craftType =
        ProjectOptionCatalogs.craftTypes[_selectedCraftIndex].value;
    final filters =
        await Navigator.of(
          context,
          rootNavigator: true,
        ).push<ProjectBrowseFilters>(
          MaterialPageRoute<ProjectBrowseFilters>(
            fullscreenDialog: true,
            builder: (_) => _ProjectFilterSheet(
              craftType: craftType,
              initialFilters: _filters,
            ),
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

  ButtonStyle _appBarControlStyle(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>() ?? const SpacingTheme();
    return OutlinedButton.styleFrom(
      foregroundColor: theme.colorScheme.onSurface,
      side: BorderSide(color: theme.colorScheme.outlineVariant, width: 1.5),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(spacing.sp2),
      ),
      padding: EdgeInsets.symmetric(
        horizontal: spacing.sp3,
        vertical: spacing.sp2,
      ),
    );
  }
}

class _ProjectCraftTabBarDelegate extends SliverPersistentHeaderDelegate {
  const _ProjectCraftTabBarDelegate({required this.onTap});

  final ValueChanged<int> onTap;

  static const double height = 48;

  @override
  double get minExtent => height;

  @override
  double get maxExtent => height;

  @override
  Widget build(
    BuildContext context,
    double shrinkOffset,
    bool overlapsContent,
  ) {
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final spacing = theme.extension<SpacingTheme>() ?? const SpacingTheme();
    return ColoredBox(
      color: swatches.paper,
      child: Column(
        children: [
          Expanded(
            child: TabBar(
              onTap: onTap,
              isScrollable: true,
              tabAlignment: TabAlignment.start,
              padding: EdgeInsets.symmetric(horizontal: spacing.sp2),
              tabs: [
                for (final option in ProjectOptionCatalogs.craftTypes)
                  Tab(text: option.label),
              ],
            ),
          ),
          const CraftskyDivider(),
        ],
      ),
    );
  }

  @override
  bool shouldRebuild(covariant _ProjectCraftTabBarDelegate oldDelegate) {
    return onTap != oldDelegate.onTap;
  }
}

class _ProjectTabScrollView extends ConsumerWidget {
  const _ProjectTabScrollView({
    required this.craftType,
    required this.filters,
    required this.sort,
  });

  final String craftType;
  final ProjectBrowseFilters filters;
  final SearchSort sort;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final query = ProjectBrowseQuery(
      craftTypes: [craftType],
      filters: filters,
      sort: sort,
    );
    final projectFeedAsync = ref.watch(projectFeedProvider(query));
    return CustomScrollView(
      key: PageStorageKey<String>('projects_tab_$craftType'),
      slivers: [
        switch (projectFeedAsync) {
          AsyncValue(:final value?) => _ProjectPostSlivers(
            posts: value.items,
            isLoadingMore: projectFeedAsync.isLoading,
            hasLoadMoreError: projectFeedAsync.hasError,
            onNearEnd: () =>
                ref.read(projectFeedProvider(query).notifier).loadMore(),
          ),
          _ when projectFeedAsync.hasError => _ProjectErrorSliver(
            onRetry: () => ref.invalidate(projectFeedProvider(query)),
          ),
          _ => const SliverFillRemaining(
            hasScrollBody: false,
            child: Center(child: StitchProgressIndicator()),
          ),
        },
      ],
    );
  }
}

class _ProjectPostSlivers extends StatelessWidget {
  const _ProjectPostSlivers({
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
    return AutoPaginatedSliverList(
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

class _ProjectErrorSliver extends StatelessWidget {
  const _ProjectErrorSliver({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return SliverFillRemaining(
      hasScrollBody: false,
      child: Center(
        child: TextButton.icon(
          onPressed: onRetry,
          icon: const Icon(Icons.refresh),
          label: Text(l10n.projectsLoadError),
        ),
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
