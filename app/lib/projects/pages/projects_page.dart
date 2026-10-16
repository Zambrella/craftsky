import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/delete_post_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_like_post_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_repost_post_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/moderation/widgets/report_flow.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/providers/preferred_project_crafts_provider.dart';
import 'package:craftsky_app/projects/providers/project_feed_provider.dart';
import 'package:craftsky_app/router/app_shell_drawer.dart';
import 'package:craftsky_app/router/responsive_modal_navigation.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/widgets/auto_paginated_list_view.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/shared/widgets/craftsky_skeleton.dart';
import 'package:craftsky_app/shared/widgets/scroll_to_top_button.dart';
import 'package:craftsky_app/shared/widgets/sort_menu_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/craftsky_floating_action_button.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:craftsky_app/theme/yarn_refresh_scroll_view.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

part 'project_filter_sheet.dart';

class ProjectsPage extends ConsumerStatefulWidget {
  const new({this.craft, super.key});

  final String? craft;

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
  late List<ProjectOption> _craftTypes;
  var _hasSelectedCraft = false;
  var _reorderingCrafts = false;
  SearchSort _sort = SearchSort.chronological;
  ProjectBrowseFilters _filters = const ProjectBrowseFilters();
  var _isPastScrollThreshold = false;
  final _innerScrollOffsets = <String, double>{};

  @override
  void initState() {
    super.initState();
    _craftTypes = ref.read(preferredProjectCraftsProvider);
    final requested = _requestedIndex(widget.craft);
    _selectedCraftIndex = requested ?? 0;
    _hasSelectedCraft = requested != null;
    _tabController = TabController(
      initialIndex: _selectedCraftIndex,
      length: _craftTypes.length,
      vsync: this,
    )..addListener(_handleTabChanged);
    ref.listenManual(preferredProjectCraftsProvider, (_, crafts) {
      final activeCraft = _craftTypes[_selectedCraftIndex].value;
      setState(() {
        _craftTypes = crafts;
        _selectedCraftIndex = _hasSelectedCraft
            ? crafts.indexWhere((option) => option.value == activeCraft)
            : 0;
        _reorderingCrafts = true;
        _tabController.index = _selectedCraftIndex;
        _reorderingCrafts = false;
      });
    });
    _scrollController.addListener(_handleOuterScroll);
  }

  int? _requestedIndex(String? craft) {
    final index = _craftTypes.indexWhere(
      (option) => option.value.split('#').last == craft,
    );
    return index < 0 ? null : index;
  }

  @override
  void didUpdateWidget(ProjectsPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.craft == widget.craft) return;
    final requested = _requestedIndex(widget.craft);
    _hasSelectedCraft = requested != null;
    _selectedCraftIndex = requested ?? 0;
    _reorderingCrafts = true;
    _tabController.index = _selectedCraftIndex;
    _reorderingCrafts = false;
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
    ref
      ..listen(deletePostProvider, (previous, next) {
        switch ((previous, next)) {
          case (AsyncLoading(), AsyncData(value: != null)):
            context.showInfo(l10n.postDeleteSuccess);
            ref.read(deletePostProvider.notifier).reset();
          case (AsyncLoading(), AsyncError()):
            context.showError(l10n.postDeleteError);
            ref.read(deletePostProvider.notifier).reset();
          case _:
            break;
        }
      })
      ..listen(toggleLikePostProvider, (previous, next) {
        if (next.hasError) {
          context.showError(l10n.postLikeError);
          ref.read(toggleLikePostProvider.notifier).reset();
        }
      })
      ..listen(toggleRepostPostProvider, (previous, next) {
        if (next.hasError) {
          context.showError(l10n.postRepostError);
          ref.read(toggleRepostPostProvider.notifier).reset();
        }
      });
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final activeCraft = _craftTypes[_selectedCraftIndex].value;
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
                    craftTypes: _craftTypes,
                  ),
                ),
                SliverToBoxAdapter(
                  child: _ActiveFilterChips(
                    craftType: activeCraft,
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
                  for (final option in _craftTypes)
                    _ProjectTabScrollView(
                      key: ValueKey(option.value),
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
    if (_reorderingCrafts || _selectedCraftIndex == _tabController.index) {
      return;
    }
    setState(() {
      _hasSelectedCraft = true;
      _selectedCraftIndex = _tabController.index;
      _filters = const ProjectBrowseFilters();
      _isPastScrollThreshold = false;
    });
    if (GoRouter.maybeOf(context) != null) {
      ProjectsRoute(
        craft: _craftTypes[_selectedCraftIndex].value.split('#').last,
      ).replace(context);
    }
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _updateScrollThreshold();
    });
  }

  void _handleOuterScroll() => _updateScrollThreshold();

  void _handleInnerScroll(String craftType, double offset) {
    final activeCraft = _craftTypes[_selectedCraftIndex].value;
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
    final activeCraft = _craftTypes[_selectedCraftIndex].value;
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
    final activeCraft = _craftTypes[_selectedCraftIndex].value;
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
      controller.animateTo(
        0,
        duration: durations.medium,
        curve: durations.ease,
      );
    }
  }

  void _setFilters(ProjectBrowseFilters filters) {
    if (filters == _filters) return;
    final activeCraft = _craftTypes[_selectedCraftIndex].value;
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
    final craftType = _craftTypes[_selectedCraftIndex].value;
    final filters = await responsiveModalNavigator(context)
        .push<ProjectBrowseFilters>(
          FullscreenModalRoute<ProjectBrowseFilters>(
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
      icon: CraftskyIcons.schedule,
      label: l10n.searchSortNewest,
      description: l10n.searchSortNewestDescription,
    ),
    SortMenuOption(
      value: SearchSort.popular,
      icon: CraftskyIcons.trending,
      label: l10n.searchSortPopular,
      description: l10n.searchSortPopularDescription,
    ),
  ];
}

class _ProjectCraftTabBarDelegate extends SliverPersistentHeaderDelegate {
  const new({
    required this.controller,
    required this.craftTypes,
  });

  final TabController controller;
  final List<ProjectOption> craftTypes;

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
                for (final option in craftTypes)
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
    return controller != oldDelegate.controller ||
        craftTypes != oldDelegate.craftTypes;
  }
}

class _ProjectTabScrollView extends ConsumerWidget {
  const new({
    required this.craftType,
    required this.filters,
    required this.sort,
    required this.onScrollOffsetChanged,
    required this.onClearFilters,
    super.key,
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
      child: YarnRefreshScrollView(
        onRefresh: () async {
          final _ = await ref.refresh(projectFeedProvider(query).future);
        },
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
            _ => CraftskySkeletonSliverList(
              itemCount: 3,
              itemBuilder: (context, index) => PostCardSkeleton(
                showMedia: index == 0,
              ),
            ),
          },
        ],
        scrollViewKey: PageStorageKey<String>('projects_tab_$craftType'),
      ),
    );
  }
}

class _ProjectPostSlivers extends ConsumerWidget {
  const new({
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
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final viewerDid = switch (ref.watch(authSessionProvider).value) {
      SignedIn(:final did) => did,
      _ => null,
    };
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
          allowProfilePinAction: true,
          onTap: () => PostThreadRoute(
            did: post.author.did,
            rkey: post.rkey,
          ).push<void>(context),
          onReply: () => unawaited(
            showPostComposerSheet(context, replyTarget: post),
          ),
          onLike: () =>
              ref.read(toggleLikePostProvider.notifier).toggle(post: post),
          onRepost: () =>
              ref.read(toggleRepostPostProvider.notifier).toggle(post: post),
          onQuote: () => unawaited(
            showPostComposerSheet(context, quoteTarget: post),
          ),
          onDelete: viewerDid == post.author.did
              ? () => unawaited(_confirmProjectPostDelete(context, ref, post))
              : null,
          onReport: viewerDid != null && viewerDid != post.author.did
              ? () => showPostReportSheet(context, ref, post)
              : null,
          replyTooltip: l10n.postCommentAction,
        );
      },
    );
  }
}

Future<void> _confirmProjectPostDelete(
  BuildContext context,
  WidgetRef ref,
  Post post,
) async {
  final l10n = AppLocalizations.of(context);
  await showCraftskyDestructiveConfirmDialog(
    context,
    title: l10n.postDeleteTitle,
    message: l10n.postDeleteMessage,
    confirmLabel: l10n.postDeleteConfirm,
    onConfirm: () => ref.read(deletePostProvider.notifier).delete(post: post),
  );
}

class _ProjectErrorSliver extends StatelessWidget {
  const new({required this.onRetry});

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
  const new({
    required this.craftType,
    required this.filters,
    required this.onRemove,
    required this.onClear,
  });

  final String craftType;
  final ProjectBrowseFilters filters;
  final void Function(ProjectBrowseFilterFamily family, String value) onRemove;
  final VoidCallback onClear;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final chips = _activeFilters(
      filters,
      craftType,
      l10n.projectsFilterSelfDrafted,
    );
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
  const new({
    required this.family,
    required this.value,
    required this.label,
  });

  final ProjectBrowseFilterFamily family;
  final String value;
  final String label;
}

List<_FilterChipData> _activeFilters(
  ProjectBrowseFilters filters,
  String craftType,
  String selfDraftedLabel,
) => [
  for (final value in filters.projectType)
    _FilterChipData(
      family: ProjectBrowseFilterFamily.projectType,
      value: value,
      label: _optionLabel(ProjectOptionCatalogs.projectTypes, value),
    ),
  for (final value in filters.status)
    _FilterChipData(
      family: ProjectBrowseFilterFamily.status,
      value: value,
      label: _optionLabel(ProjectOptionCatalogs.statuses, value),
    ),
  for (final value in filters.projectSubtype)
    _FilterChipData(
      family: ProjectBrowseFilterFamily.projectSubtype,
      value: value,
      label: _optionLabel(
        ProjectOptionCatalogs.projectSubtypesForCraft(craftType),
        value,
      ),
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
  for (final value in filters.designTag)
    _FilterChipData(
      family: ProjectBrowseFilterFamily.designTag,
      value: value,
      label: _optionLabel(ProjectOptionCatalogs.designTags, value),
    ),
  for (final value in filters.yarnWeight)
    _FilterChipData(
      family: ProjectBrowseFilterFamily.yarnWeight,
      value: value,
      label: _optionLabel(ProjectOptionCatalogs.yarnWeights, value),
    ),
  for (final value in filters.piecingTechnique)
    _FilterChipData(
      family: ProjectBrowseFilterFamily.piecingTechnique,
      value: value,
      label: _optionLabel(
        ProjectOptionCatalogs.quiltingPiecingTechniques,
        value,
      ),
    ),
  for (final value in filters.quiltingMethod)
    _FilterChipData(
      family: ProjectBrowseFilterFamily.quiltingMethod,
      value: value,
      label: _optionLabel(ProjectOptionCatalogs.quiltingMethods, value),
    ),
  if (filters.selfDrafted)
    _FilterChipData(
      family: ProjectBrowseFilterFamily.selfDrafted,
      value: 'true',
      label: selfDraftedLabel,
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
