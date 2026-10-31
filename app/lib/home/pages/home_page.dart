import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/feed/providers/delete_post_provider.dart';
import 'package:craftsky_app/feed/providers/feed_reselect_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_like_post_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_repost_post_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/feed/widgets/post_type_chooser.dart';
import 'package:craftsky_app/home/models/home_feed_definition.dart';
import 'package:craftsky_app/keys.dart';
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
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/craftsky_floating_action_button.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:craftsky_app/theme/yarn_refresh_scroll_view.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

part 'project_filter_sheet.dart';

class HomePage extends ConsumerStatefulWidget {
  const new({this.craft, super.key});

  final String? craft;

  @override
  ConsumerState<HomePage> createState() => _HomePageState();
}

class _HomePageState extends ConsumerState<HomePage>
    with TickerProviderStateMixin {
  static const _scrollToTopThreshold = 200.0;
  late TabController _tabController;
  final _scrollControllers = <String, ScrollController>{};
  var _selectedFeedIndex = 0;
  late List<HomeFeedDefinition> _feeds;
  var _hasSelectedCraft = false;
  var _reorderingCrafts = false;
  final _sorts = <String, SearchSort>{};
  SearchSort _sortFor(String id) => _sorts[id] ?? SearchSort.chronological;
  final _craftFilters = <String, ProjectBrowseFilters>{};
  var _isPastScrollThreshold = false;

  @override
  void initState() {
    super.initState();
    _feeds = _definitions(ref.read(preferredProjectCraftsProvider));
    final requested = _requestedIndex(widget.craft);
    _selectedFeedIndex = requested ?? 0;
    _hasSelectedCraft = requested != null;
    _tabController = TabController(
      initialIndex: _selectedFeedIndex,
      length: _feeds.length,
      vsync: this,
    )..addListener(_handleTabChanged);
    ref.listenManual(preferredProjectCraftsProvider, (_, crafts) {
      final activeId = _feeds[_selectedFeedIndex].id;
      setState(() {
        _feeds = _definitions(crafts);
        _selectedFeedIndex = _hasSelectedCraft
            ? _feeds.indexWhere((feed) => feed.id == activeId)
            : 0;
        if (_selectedFeedIndex < 0) _selectedFeedIndex = 0;
        _reorderingCrafts = true;
        if (_tabController.length != _feeds.length) {
          _tabController
            ..removeListener(_handleTabChanged)
            ..dispose();
          _tabController = TabController(
            length: _feeds.length,
            initialIndex: _selectedFeedIndex,
            vsync: this,
          )..addListener(_handleTabChanged);
        } else {
          _tabController.index = _selectedFeedIndex;
        }
        _reorderingCrafts = false;
      });
    });
  }

  List<HomeFeedDefinition> _definitions(List<ProjectOption> crafts) => [
    const HomeFeedDefinition.following(),
    for (final craft in crafts) HomeFeedDefinition.project(craft),
  ];

  int? _requestedIndex(String? craft) {
    if (craft == null) return null;
    final index = _feeds.indexWhere(
      (option) => option.craftType?.split('#').last == craft,
    );
    return index < 0 ? null : index;
  }

  @override
  void didUpdateWidget(HomePage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.craft == widget.craft) return;
    final requested = _requestedIndex(widget.craft);
    _hasSelectedCraft = requested != null;
    _selectedFeedIndex = requested ?? 0;
    _reorderingCrafts = true;
    _tabController.index = _selectedFeedIndex;
    _reorderingCrafts = false;
    _isPastScrollThreshold = false;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _updateScrollThreshold();
    });
  }

  @override
  void dispose() {
    _tabController
      ..removeListener(_handleTabChanged)
      ..dispose();
    for (final controller in _scrollControllers.values) {
      controller.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    if (_feeds[_selectedFeedIndex].source == HomeFeedSource.projects) {
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
    }
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final activeFeed = _feeds[_selectedFeedIndex];
    final activeCraft = activeFeed.craftType;
    final activeQuery = activeCraft == null
        ? null
        : ProjectBrowseQuery(
            craftTypes: [activeCraft],
            filters: _filtersFor(activeCraft),
            sort: _sortFor(activeFeed.id),
          );
    final hasActiveProjects =
        activeQuery != null &&
        (ref.watch(projectFeedProvider(activeQuery)).value?.items.isNotEmpty ??
            false);
    ref.listen(feedReselectProvider, (_, _) {
      if (activeQuery == null) return;
      _scrollToTop();
      ref.invalidate(projectFeedProvider(activeQuery));
    });
    return Scaffold(
      floatingActionButton: activeFeed.supportsFilters
          ? CraftskyFloatingActionButton.extended(
              tooltip: l10n.projectsFilterAction,
              onPressed: _openFilters,
              icon: const Icon(CraftskyIconsBold.adjustments),
              label: Text(l10n.projectsFilterAction),
            )
          : FormFactor.fromWidth(MediaQuery.sizeOf(context).width).isSmall
          ? Builder(
              builder: (buttonContext) => CraftskyFloatingActionButton.extended(
                key: keys.feed.compose,
                tooltip: l10n.postComposeAction,
                icon: const Icon(CraftskyIconsBold.add),
                label: Text(l10n.postComposeAction),
                onPressed: () => unawaited(
                  showTopLevelPostComposerChooser(
                    buttonContext,
                    position: craftskyContextMenuAnchorPosition(buttonContext),
                  ),
                ),
              ),
            )
          : null,
      appBar: AppBar(
        leading: AppShellDrawerScope.maybeOf(context) == null
            ? null
            : const AppShellDrawerButton(),
        title: Text(l10n.homeTitle),
        actions: [
          if (activeFeed.supportsSort)
            Padding(
              padding: EdgeInsetsDirectional.only(end: spacing.sp4),
              child: SortMenuButton<SearchSort>(
                selectedValue: _sortFor(activeFeed.id),
                options: _sortOptions(l10n),
                onChanged: (sort) => _setSort(activeFeed.id, sort),
              ),
            ),
        ],
      ),
      body: Stack(
        children: [
          Column(
            children: [
              _HomeFeedTabBar(
                controller: _tabController,
                feeds: _feeds,
              ),
              Expanded(
                child: TabBarView(
                  controller: _tabController,
                  children: [
                    for (final option in _feeds)
                      if (option.source == HomeFeedSource.following)
                        FeedPage(
                          key: ValueKey(option.id),
                          embedded: true,
                          active: activeFeed.id == option.id,
                        )
                      else
                        _ProjectTabScrollView(
                          key: ValueKey(option.id),
                          craftType: option.craftType!,
                          controller: _controllerFor(option.id),
                          filters: _filtersFor(option.craftType!),
                          sort: _sortFor(option.id),
                          onRemoveFilter: (family, value) => _setFilters(
                            option.craftType!,
                            _filtersFor(option.craftType!)
                                .withoutValue(family, value),
                          ),
                          onClearFilters: () => _setFilters(
                            option.craftType!,
                            const ProjectBrowseFilters(),
                          ),
                        ),
                  ],
                ),
              ),
            ],
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
    if (_reorderingCrafts || _selectedFeedIndex == _tabController.index) {
      return;
    }
    setState(() {
      _hasSelectedCraft = true;
      _selectedFeedIndex = _tabController.index;
      _isPastScrollThreshold = false;
    });
    if (GoRouter.maybeOf(context) != null) {
      FeedRoute(
        craft: _feeds[_selectedFeedIndex].craftType?.split('#').last,
      ).replace(context);
    }
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _updateScrollThreshold();
    });
  }

  ScrollController _controllerFor(String craftType) =>
      _scrollControllers.putIfAbsent(
        craftType,
        () => ScrollController(
          onAttach: (_) {
            WidgetsBinding.instance.addPostFrameCallback((_) {
              if (mounted) _updateScrollThreshold();
            });
          },
        )..addListener(_updateScrollThreshold),
      );

  ScrollController get _activeScrollController =>
      _controllerFor(_feeds[_selectedFeedIndex].id);

  void _updateScrollThreshold() {
    final controller = _activeScrollController;
    _setScrollThreshold(
      controller.hasClients && controller.offset >= _scrollToTopThreshold,
    );
  }

  void _setScrollThreshold(bool value) {
    if (!mounted || value == _isPastScrollThreshold) return;
    setState(() => _isPastScrollThreshold = value);
  }

  void _scrollToTop() {
    final controller = _activeScrollController;
    if (!controller.hasClients) return;
    if (MediaQuery.disableAnimationsOf(context)) {
      controller.jumpTo(0);
      return;
    }
    final durations = Theme.of(context).extension<DurationTheme>()!;
    controller.animateTo(
      0,
      duration: durations.medium,
      curve: durations.ease,
    );
  }

  ProjectBrowseFilters _filtersFor(String craftType) =>
      _craftFilters[craftType] ?? const ProjectBrowseFilters();

  void _setFilters(String craftType, ProjectBrowseFilters filters) {
    if (filters == _filtersFor(craftType)) return;
    final controller = _controllerFor(HomeFeedDefinition.projectId(craftType));
    setState(() => _craftFilters[craftType] = filters);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !controller.hasClients) return;
      controller.jumpTo(0);
    });
  }

  void _setSort(String id, SearchSort sort) {
    if (sort == _sortFor(id)) return;
    final controller = _controllerFor(id);
    setState(() => _sorts[id] = sort);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted && controller.hasClients) controller.jumpTo(0);
    });
  }

  Future<void> _openFilters() async {
    final craftType = _feeds[_selectedFeedIndex].craftType;
    if (craftType == null) return;
    final filters = await responsiveModalNavigator(context)
        .push<ProjectBrowseFilters>(
          FullscreenModalRoute<ProjectBrowseFilters>(
            builder: (_) => _ProjectFilterSheet(
              craftType: craftType,
              initialFilters: _filtersFor(craftType),
            ),
          ),
        );
    if (filters == null || !mounted) return;
    _setFilters(craftType, filters);
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

class _HomeFeedTabBar extends StatelessWidget {
  const new({
    required this.controller,
    required this.feeds,
  });

  final TabController controller;
  final List<HomeFeedDefinition> feeds;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final spacing = theme.extension<SpacingTheme>() ?? const SpacingTheme();
    return SizedBox(
      height: 48,
      child: ColoredBox(
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
                  for (final option in feeds)
                    Tab(
                      child: switch (option.craftType) {
                        final craft? => CraftIconLabel(
                          craft: craft,
                          label: option.label!,
                        ),
                        null => Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            const Icon(CraftskyIcons.people, size: 18),
                            const SizedBox(width: 6),
                            Text(AppLocalizations.of(context).homeFollowingTab),
                          ],
                        ),
                      },
                    ),
                ],
              ),
            ),
            const CraftskyDivider(),
          ],
        ),
      ),
    );
  }
}

class _ProjectTabScrollView extends ConsumerStatefulWidget {
  const new({
    required this.craftType,
    required this.filters,
    required this.sort,
    required this.controller,
    required this.onRemoveFilter,
    required this.onClearFilters,
    super.key,
  });

  final String craftType;
  final ProjectBrowseFilters filters;
  final SearchSort sort;
  final ScrollController controller;
  final void Function(ProjectBrowseFilterFamily family, String value)
  onRemoveFilter;
  final VoidCallback onClearFilters;

  @override
  ConsumerState<_ProjectTabScrollView> createState() =>
      _ProjectTabScrollViewState();
}

class _ProjectTabScrollViewState extends ConsumerState<_ProjectTabScrollView>
    with AutomaticKeepAliveClientMixin<_ProjectTabScrollView> {
  // Retain both the scroll position and the feed subscription while offscreen.
  // A reload could otherwise clamp a deep offset to the loading skeleton.
  @override
  bool get wantKeepAlive => true;

  @override
  Widget build(BuildContext context) {
    super.build(context);
    final query = ProjectBrowseQuery(
      craftTypes: [widget.craftType],
      filters: widget.filters,
      sort: widget.sort,
    );
    final projectFeedAsync = ref.watch(projectFeedProvider(query));
    return YarnRefreshScrollView(
      controller: widget.controller,
      onRefresh: () async {
        final _ = await ref.refresh(projectFeedProvider(query).future);
      },
      slivers: [
        SliverToBoxAdapter(
          child: _ActiveFilterChips(
            craftType: widget.craftType,
            filters: widget.filters,
            onRemove: widget.onRemoveFilter,
            onClear: widget.onClearFilters,
          ),
        ),
        switch (projectFeedAsync) {
          AsyncValue(:final value?) => _ProjectPostSlivers(
            posts: value.items,
            isLoadingMore: projectFeedAsync.isLoading,
            hasLoadMoreError: projectFeedAsync.hasError,
            hasActiveFilters: widget.filters.toQueryParameters().isNotEmpty,
            onClearFilters: widget.onClearFilters,
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
      scrollViewKey: PageStorageKey<String>('projects_tab_${widget.craftType}'),
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
          onReply: () => PostThreadRoute(
            did: post.author.did,
            rkey: post.rkey,
            showComments: true,
          ).push<void>(context),
          onLike: () =>
              ref.read(toggleLikePostProvider.notifier).toggle(post: post),
          onRepost: () =>
              ref.read(toggleRepostPostProvider.notifier).toggle(post: post),
          onQuote: (imageIndex) => unawaited(
            showPostComposerSheet(
              context,
              quoteTarget: post,
              quoteImageIndex: imageIndex,
            ),
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
