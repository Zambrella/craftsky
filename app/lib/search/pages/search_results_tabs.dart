part of 'search_page.dart';

class _SearchResultsTabs extends StatefulWidget {
  const _SearchResultsTabs({
    required this.query,
    required this.initialTab,
    required this.onOpenHashtag,
    required this.headerSlivers,
  });

  final String query;
  final SearchResultsTab initialTab;
  final ValueChanged<String> onOpenHashtag;
  final List<Widget> headerSlivers;

  @override
  State<_SearchResultsTabs> createState() => _SearchResultsTabsState();
}

class _SearchResultsTabsState extends State<_SearchResultsTabs>
    with SingleTickerProviderStateMixin {
  late TabController _controller;
  late SearchResultsTab _selectedTab;

  @override
  void initState() {
    super.initState();
    _selectedTab = widget.initialTab;
    _controller = TabController(
      length: SearchResultsTab.values.length,
      initialIndex: _selectedTab.index,
      vsync: this,
    )..addListener(_handleTabChange);
  }

  @override
  void didUpdateWidget(covariant _SearchResultsTabs oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.initialTab != widget.initialTab &&
        widget.initialTab.index != _controller.index) {
      _selectedTab = widget.initialTab;
      _controller.index = widget.initialTab.index;
    }
  }

  @override
  void dispose() {
    _controller
      ..removeListener(_handleTabChange)
      ..dispose();
    super.dispose();
  }

  void _handleTabChange() {
    if (_controller.indexIsChanging) return;
    final next = SearchResultsTab.values[_controller.index];
    if (next == _selectedTab) return;
    _selectedTab = next;
    SearchRoute(q: widget.query, tab: next).go(context);
  }

  @override
  Widget build(BuildContext context) {
    return NestedScrollView(
      headerSliverBuilder: (context, innerBoxIsScrolled) => [
        ...widget.headerSlivers,
        SliverPersistentHeader(
          pinned: true,
          delegate: _SearchResultsTabBarDelegate(controller: _controller),
        ),
      ],
      body: TabBarView(
        controller: _controller,
        children: [
          for (final tab in SearchResultsTab.values)
            _SearchResultTabScrollView(
              tab: tab,
              query: widget.query,
              onOpenHashtag: widget.onOpenHashtag,
            ),
        ],
      ),
    );
  }
}

class _SearchResultsTabBarDelegate extends SliverPersistentHeaderDelegate {
  const _SearchResultsTabBarDelegate({required this.controller});

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
    final l10n = AppLocalizations.of(context);
    return ColoredBox(
      color: swatches.paper,
      child: Column(
        children: [
          Expanded(
            child: TabBar(
              controller: controller,
              padding: EdgeInsets.symmetric(horizontal: spacing.sp2),
              tabs: [
                Tab(text: l10n.searchTabPosts),
                Tab(text: l10n.searchTabProjects),
                Tab(text: l10n.searchTabProfiles),
                Tab(text: l10n.searchTabTags),
              ],
            ),
          ),
          const CraftskyDivider(),
        ],
      ),
    );
  }

  @override
  bool shouldRebuild(covariant _SearchResultsTabBarDelegate oldDelegate) {
    return controller != oldDelegate.controller;
  }
}

class _SearchResultTabScrollView extends ConsumerWidget {
  const _SearchResultTabScrollView({
    required this.tab,
    required this.query,
    required this.onOpenHashtag,
  });

  final SearchResultsTab tab;
  final String query;
  final ValueChanged<String> onOpenHashtag;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return RefreshIndicator(
      onRefresh: () => _refresh(ref),
      child: CustomScrollView(
        key: PageStorageKey<String>('search_results_tab_${tab.name}'),
        physics: const AlwaysScrollableScrollPhysics(),
        slivers: [_sliverForTab(tab)],
      ),
    );
  }

  Future<void> _refresh(WidgetRef ref) => switch (tab) {
    SearchResultsTab.posts => ref.refresh(
      postSearchProvider(PostSearchQuery(q: query)).future,
    ),
    SearchResultsTab.projects => ref.refresh(
      projectSearchProvider(ProjectSearchQuery(q: query)).future,
    ),
    SearchResultsTab.profiles => ref.refresh(
      profileSearchProvider(ProfileSearchQuery(q: query)).future,
    ),
    SearchResultsTab.tags => ref.refresh(
      hashtagResultSearchProvider(HashtagResultSearchQuery(q: query)).future,
    ),
  };

  Widget _sliverForTab(SearchResultsTab tab) {
    return switch (tab) {
      SearchResultsTab.posts => _SubmittedPostResults(query: query),
      SearchResultsTab.projects => _SubmittedProjectResults(query: query),
      SearchResultsTab.profiles => _ProfileResultsSliver(query: query),
      SearchResultsTab.tags => _HashtagResultsSliver(
        query: query,
        onOpenHashtag: onOpenHashtag,
      ),
    };
  }
}

class _SubmittedPostResults extends ConsumerWidget {
  const _SubmittedPostResults({required this.query});

  final String query;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final provider = postSearchProvider(PostSearchQuery(q: query));
    final postResultsAsync = ref.watch(provider);
    return switch (postResultsAsync) {
      AsyncValue(:final value?) => _PostList(
        posts: value.items,
        emptyState: CraftskyEmptyState(
          icon: CraftskyIcons.textPost,
          title: l10n.searchTabPosts,
          subtitle: l10n.searchEmptyPosts,
        ),
        isLoadingMore: postResultsAsync.isLoading,
        hasLoadMoreError: postResultsAsync.hasError,
        onNearEnd: () => ref.read(provider.notifier).loadMore(),
      ),
      _ when postResultsAsync.hasError => _ErrorView(
        message: l10n.searchLoadError,
        onRetry: () => ref.invalidate(provider),
      ),
      _ => const SliverFillRemaining(
        hasScrollBody: false,
        child: Center(child: StitchProgressIndicator()),
      ),
    };
  }
}

class _SubmittedProjectResults extends ConsumerWidget {
  const _SubmittedProjectResults({required this.query});

  final String query;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final provider = projectSearchProvider(ProjectSearchQuery(q: query));
    final projectResultsAsync = ref.watch(provider);
    return switch (projectResultsAsync) {
      AsyncValue(:final value?) => _PostList(
        posts: value.items,
        emptyState: CraftskyEmptyState(
          icon: CraftskyIcons.projects,
          title: l10n.searchTabProjects,
          subtitle: l10n.searchEmptyProjects,
        ),
        isLoadingMore: projectResultsAsync.isLoading,
        hasLoadMoreError: projectResultsAsync.hasError,
        onNearEnd: () => ref.read(provider.notifier).loadMore(),
      ),
      _ when projectResultsAsync.hasError => _ErrorView(
        message: l10n.searchLoadError,
        onRetry: () => ref.invalidate(provider),
      ),
      _ => const SliverFillRemaining(
        hasScrollBody: false,
        child: Center(child: StitchProgressIndicator()),
      ),
    };
  }
}

class _ProfileResultsSliver extends ConsumerWidget {
  const _ProfileResultsSliver({required this.query});

  final String query;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final provider = profileSearchProvider(ProfileSearchQuery(q: query));
    final profileResultsAsync = ref.watch(provider);
    return switch (profileResultsAsync) {
      AsyncValue(:final value?) => AutoPaginatedSliverList(
        itemCount: value.items.length,
        emptyState: CraftskyEmptyState(
          icon: CraftskyIcons.people,
          title: l10n.searchTabProfiles,
          subtitle: l10n.searchEmptyProfiles,
        ),
        isLoadingMore: profileResultsAsync.isLoading,
        hasLoadMoreError: profileResultsAsync.hasError,
        onNearEnd: () => ref.read(provider.notifier).loadMore(),
        itemBuilder: (context, index) {
          final profile = value.items[index];
          return _ProfileResultTile(
            profile: profile,
            onTap: () => unawaited(
              showUserProfileCard(
                context,
                handleOrDid: profile.handle.toString(),
              ),
            ),
          );
        },
      ),
      _ when profileResultsAsync.hasError => _ErrorView(
        message: l10n.searchLoadError,
        onRetry: () => ref.invalidate(provider),
      ),
      _ => const SliverFillRemaining(
        hasScrollBody: false,
        child: Center(child: StitchProgressIndicator()),
      ),
    };
  }
}

class _HashtagResultsSliver extends ConsumerWidget {
  const _HashtagResultsSliver({
    required this.query,
    required this.onOpenHashtag,
  });

  final String query;
  final ValueChanged<String> onOpenHashtag;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final provider = hashtagResultSearchProvider(
      HashtagResultSearchQuery(q: query),
    );
    final hashtagResultsAsync = ref.watch(provider);
    return switch (hashtagResultsAsync) {
      AsyncValue(:final value?) => AutoPaginatedSliverList(
        itemCount: value.items.length,
        emptyState: CraftskyEmptyState(
          icon: CraftskyIcons.search,
          title: l10n.searchTabTags,
          subtitle: l10n.searchEmptyTags,
        ),
        isLoadingMore: hashtagResultsAsync.isLoading,
        hasLoadMoreError: hashtagResultsAsync.hasError,
        onNearEnd: () => ref.read(provider.notifier).loadMore(),
        itemBuilder: (context, index) {
          final hashtag = value.items[index];
          return _HashtagResultTile(
            hashtag: hashtag,
            onTap: () => onOpenHashtag(hashtag.tag),
          );
        },
      ),
      _ when hashtagResultsAsync.hasError => _ErrorView(
        message: l10n.searchLoadError,
        onRetry: () => ref.invalidate(provider),
      ),
      _ => const SliverFillRemaining(
        hasScrollBody: false,
        child: Center(child: StitchProgressIndicator()),
      ),
    };
  }
}
