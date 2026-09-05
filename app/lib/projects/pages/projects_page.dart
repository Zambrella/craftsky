import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/providers/project_feed_provider.dart';
import 'package:craftsky_app/router/app_shell_drawer.dart';
import 'package:craftsky_app/router/responsive_modal_navigation.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/shared/widgets/auto_paginated_list_view.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/shared/widgets/scroll_to_top_button.dart';
import 'package:craftsky_app/shared/widgets/sort_menu_button.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/craftsky_floating_action_button.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
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

class _ProjectsPageState extends ConsumerState<ProjectsPage>
    with SingleTickerProviderStateMixin {
  static const _scrollToTopThreshold = 200.0;
  late final TabController _tabController;
  final _scrollController = ScrollController();
  final _nestedScrollKey = GlobalKey<NestedScrollViewState>();
  var _selectedCraftIndex = 0;
  SearchSort _sort = SearchSort.chronological;
  ProjectBrowseFilters _filters = const ProjectBrowseFilters();
  var _isPastScrollThreshold = false;
  final _innerScrollOffsets = <String, double>{};

  @override
  void initState() {
    super.initState();
    _tabController = TabController(
      length: ProjectOptionCatalogs.craftTypes.length,
      vsync: this,
    )..addListener(_handleTabChanged);
    _scrollController.addListener(_handleOuterScroll);
  }

  @override
  void dispose() {
    _tabController
      ..removeListener(_handleTabChanged)
      ..dispose();
    _scrollController
      ..removeListener(_handleOuterScroll)
      ..dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final activeCraft =
        ProjectOptionCatalogs.craftTypes[_selectedCraftIndex].value;
    final activeQuery = ProjectBrowseQuery(
      craftTypes: [activeCraft],
      filters: _filters,
      sort: _sort,
    );
    final hasActiveProjects =
        ref.watch(projectFeedProvider(activeQuery)).value?.items.isNotEmpty ??
        false;
    return Scaffold(
      floatingActionButton: CraftskyFloatingActionButton.extended(
        tooltip: l10n.projectsFilterAction,
        onPressed: _openFilters,
        icon: const Icon(CraftskyIconsBold.adjustments),
        label: Text(l10n.projectsFilterAction),
      ),
      body: Stack(
        children: [
          NotificationListener<ScrollNotification>(
            onNotification: _handleScrollNotification,
            child: NestedScrollView(
              key: _nestedScrollKey,
              controller: _scrollController,
              headerSliverBuilder: (context, innerBoxIsScrolled) => [
                SliverAppBar(
                  leading: AppShellDrawerScope.maybeOf(context) == null
                      ? null
                      : const AppShellDrawerButton(),
                  title: Text(l10n.projectsTitle),
                  pinned: true,
                  actions: [
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
                    controller: _tabController,
                  ),
                ),
                SliverToBoxAdapter(
                  child: _ActiveFilterChips(
                    filters: _filters,
                    onRemove: (family, value) =>
                        _setFilters(_filters.withoutValue(family, value)),
                    onClear: () => _setFilters(const ProjectBrowseFilters()),
                  ),
                ),
              ],
              body: TabBarView(
                controller: _tabController,
                children: [
                  for (final option in ProjectOptionCatalogs.craftTypes)
                    _ProjectTabScrollView(
                      craftType: option.value,
                      filters: _filters,
                      sort: _sort,
                      onScrollOffsetChanged: (offset) =>
                          _handleInnerScroll(option.value, offset),
                      onClearFilters: () =>
                          _setFilters(const ProjectBrowseFilters()),
                    ),
                ],
              ),
            ),
          ),
          Positioned(
            left: spacing.sp4,
            bottom: spacing.sp4,
            child: ScrollToTopButton(
              visible: hasActiveProjects && _isPastScrollThreshold,
              tooltip: l10n.scrollToTopAction,
              onPressed: _scrollToTop,
            ),
          ),
        ],
      ),
    );
  }

  void _handleTabChanged() {
    if (_selectedCraftIndex == _tabController.index) return;
    setState(() {
      _selectedCraftIndex = _tabController.index;
      _filters = const ProjectBrowseFilters();
      _isPastScrollThreshold = false;
    });
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _updateScrollThreshold();
    });
  }

  void _handleOuterScroll() => _updateScrollThreshold();

  void _handleInnerScroll(String craftType, double offset) {
    final activeCraft =
        ProjectOptionCatalogs.craftTypes[_selectedCraftIndex].value;
    final tabPosition = _tabController.animation?.value;
    if (craftType != activeCraft ||
        (tabPosition != null &&
            (tabPosition - _selectedCraftIndex).abs() > 0.001)) {
      return;
    }
    _innerScrollOffsets[craftType] = offset;
    _updateScrollThreshold();
  }

  bool _handleScrollNotification(ScrollNotification notification) {
    if (notification.metrics.axis == Axis.vertical) {
      _updateScrollThreshold();
    }
    return false;
  }

  void _updateScrollThreshold() {
    final outerOffset = _scrollController.hasClients
        ? _scrollController.offset
        : 0.0;
    final activeCraft =
        ProjectOptionCatalogs.craftTypes[_selectedCraftIndex].value;
    final innerOffset = _innerScrollOffsets[activeCraft] ?? 0.0;
    _setScrollThreshold(
      outerOffset + innerOffset >= _scrollToTopThreshold,
    );
  }

  void _setScrollThreshold(bool value) {
    if (!mounted || value == _isPastScrollThreshold) return;
    setState(() => _isPastScrollThreshold = value);
  }

  void _scrollToTop() {
    final activeCraft =
        ProjectOptionCatalogs.craftTypes[_selectedCraftIndex].value;
    _innerScrollOffsets[activeCraft] = 0;
    _setScrollThreshold(false);
    final innerController = _nestedScrollKey.currentState?.innerController;
    final controllers = [
      if (innerController?.hasClients ?? false) innerController!,
      if (_scrollController.hasClients) _scrollController,
    ];
    if (controllers.isEmpty) return;
    if (MediaQuery.disableAnimationsOf(context)) {
      for (final controller in controllers) {
        controller.jumpTo(0);
      }
      return;
    }
    final durations = Theme.of(context).extension<DurationTheme>()!;
    for (final controller in controllers) {
      unawaited(
        controller.animateTo(
          0,
          duration: durations.medium,
          curve: durations.ease,
        ),
      );
    }
  }

  void _setFilters(ProjectBrowseFilters filters) {
    if (filters == _filters) return;
    final activeCraft =
        ProjectOptionCatalogs.craftTypes[_selectedCraftIndex].value;
    _innerScrollOffsets[activeCraft] = 0;
    setState(() => _filters = filters);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_scrollController.hasClients) return;
      final innerController = _nestedScrollKey.currentState?.innerController;
      if (innerController?.hasClients ?? false) innerController!.jumpTo(0);
      _scrollController.jumpTo(0);
    });
  }

  Future<void> _openFilters() async {
    final craftType =
        ProjectOptionCatalogs.craftTypes[_selectedCraftIndex].value;
    final filters = await responsiveModalNavigator(context)
        .push<ProjectBrowseFilters>(
          MaterialPageRoute<ProjectBrowseFilters>(
            fullscreenDialog: true,
            builder: (_) => _ProjectFilterSheet(
              craftType: craftType,
              initialFilters: _filters,
            ),
          ),
        );
    if (filters == null || !mounted) return;
    _setFilters(filters);
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

class _ProjectCraftTabBarDelegate extends SliverPersistentHeaderDelegate {
  const _ProjectCraftTabBarDelegate({required this.controller});

  final TabController controller;

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
              controller: controller,
              isScrollable: true,
              tabAlignment: TabAlignment.start,
              padding: EdgeInsets.symmetric(horizontal: spacing.sp2),
              tabs: [
                for (final option in ProjectOptionCatalogs.craftTypes)
                  Tab(
                    child: CraftIconLabel(
                      craft: option.value,
                      label: option.label,
                    ),
                  ),
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
    return controller != oldDelegate.controller;
  }
}

class _ProjectTabScrollView extends ConsumerWidget {
  const _ProjectTabScrollView({
    required this.craftType,
    required this.filters,
    required this.sort,
    required this.onScrollOffsetChanged,
    required this.onClearFilters,
  });

  final String craftType;
  final ProjectBrowseFilters filters;
  final SearchSort sort;
  final ValueChanged<double> onScrollOffsetChanged;
  final VoidCallback onClearFilters;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final query = ProjectBrowseQuery(
      craftTypes: [craftType],
      filters: filters,
      sort: sort,
    );
    final projectFeedAsync = ref.watch(projectFeedProvider(query));
    return NotificationListener<ScrollNotification>(
      onNotification: (notification) {
        if (notification.metrics.axis == Axis.vertical) {
          onScrollOffsetChanged(notification.metrics.pixels);
        }
        return false;
      },
      child: RefreshIndicator(
        onRefresh: () async {
          final _ = await ref.refresh(projectFeedProvider(query).future);
        },
        child: CustomScrollView(
          key: PageStorageKey<String>('projects_tab_$craftType'),
          physics: const AlwaysScrollableScrollPhysics(),
          slivers: [
            switch (projectFeedAsync) {
              AsyncValue(:final value?) => _ProjectPostSlivers(
                posts: value.items,
                isLoadingMore: projectFeedAsync.isLoading,
                hasLoadMoreError: projectFeedAsync.hasError,
                hasActiveFilters: filters.toQueryParameters().isNotEmpty,
                onClearFilters: onClearFilters,
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
        ),
      ),
    );
  }
}

class _ProjectPostSlivers extends StatelessWidget {
  const _ProjectPostSlivers({
    required this.posts,
    required this.isLoadingMore,
    required this.hasLoadMoreError,
    required this.hasActiveFilters,
    required this.onClearFilters,
    required this.onNearEnd,
  });

  final List<Post> posts;
  final bool isLoadingMore;
  final bool hasLoadMoreError;
  final bool hasActiveFilters;
  final VoidCallback onClearFilters;
  final VoidCallback onNearEnd;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return AutoPaginatedSliverList(
      itemCount: posts.length,
      emptyState: CraftskyEmptyState(
        icon: CraftskyIcons.projects,
        title: l10n.projectsTitle,
        subtitle: l10n.projectsEmpty,
        actionLabel: hasActiveFilters ? l10n.projectsClearFiltersAction : null,
        onAction: hasActiveFilters ? onClearFilters : null,
      ),
      isLoadingMore: isLoadingMore,
      hasLoadMoreError: hasLoadMoreError,
      onNearEnd: onNearEnd,
      itemBuilder: (context, index) {
        final post = posts[index];
        return PostCard(
          post: post,
          collapseBody: true,
          imageInteractionMode: PostCardImageInteractionMode.navigate,
          hideWhenAuthorProtected: true,
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
          icon: const Icon(CraftskyIconsBold.refresh),
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
