part of 'search_page.dart';

class _SearchResultsTabs extends StatefulWidget {
  const _SearchResultsTabs({
    required this.query,
    required this.initialTab,
    required this.onOpenHashtag,
  });

  final String query;
  final SearchResultsTab initialTab;
  final ValueChanged<String> onOpenHashtag;

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
    final l10n = AppLocalizations.of(context);
    return Column(
      children: [
        TabBar(
          controller: _controller,
          tabs: [
            Tab(text: l10n.searchTabPosts),
            Tab(text: l10n.searchTabProjects),
            Tab(text: l10n.searchTabProfiles),
            Tab(text: l10n.searchTabTags),
          ],
        ),
        Expanded(
          child: TabBarView(
            controller: _controller,
            children: [
              _PostResultsList(
                query: widget.query,
                type: _PostResultType.posts,
              ),
              _PostResultsList(
                query: widget.query,
                type: _PostResultType.projects,
              ),
              _ProfileResultsList(query: widget.query),
              _HashtagResultsList(
                query: widget.query,
                onOpenHashtag: widget.onOpenHashtag,
              ),
            ],
          ),
        ),
      ],
    );
  }
}

enum _PostResultType { posts, projects }

class _PostResultsList extends ConsumerWidget {
  const _PostResultsList({required this.query, required this.type});

  final String query;
  final _PostResultType type;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return switch (type) {
      _PostResultType.posts => _SubmittedPostResults(query: query),
      _PostResultType.projects => _SubmittedProjectResults(query: query),
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
        emptyText: l10n.searchEmptyPosts,
        isLoadingMore: postResultsAsync.isLoading,
        hasLoadMoreError: postResultsAsync.hasError,
        onNearEnd: () => ref.read(provider.notifier).loadMore(),
      ),
      _ when postResultsAsync.hasError => _ErrorView(
        message: l10n.searchLoadError,
        onRetry: () => ref.invalidate(provider),
      ),
      _ => const Center(child: StitchProgressIndicator()),
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
        emptyText: l10n.searchEmptyProjects,
        isLoadingMore: projectResultsAsync.isLoading,
        hasLoadMoreError: projectResultsAsync.hasError,
        onNearEnd: () => ref.read(provider.notifier).loadMore(),
      ),
      _ when projectResultsAsync.hasError => _ErrorView(
        message: l10n.searchLoadError,
        onRetry: () => ref.invalidate(provider),
      ),
      _ => const Center(child: StitchProgressIndicator()),
    };
  }
}

class _ProfileResultsList extends ConsumerWidget {
  const _ProfileResultsList({required this.query});

  final String query;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final provider = profileSearchProvider(ProfileSearchQuery(q: query));
    final profileResultsAsync = ref.watch(provider);
    return switch (profileResultsAsync) {
      AsyncValue(:final value?) => AutoPaginatedListView(
        itemCount: value.items.length,
        emptyText: l10n.searchEmptyProfiles,
        isLoadingMore: profileResultsAsync.isLoading,
        hasLoadMoreError: profileResultsAsync.hasError,
        onNearEnd: () => ref.read(provider.notifier).loadMore(),
        itemBuilder: (context, index) {
          final profile = value.items[index];
          return _ProfileResultTile(
            profile: profile,
            onTap: () => UserProfileRoute(
              handle: profile.handle.toString(),
            ).push<void>(context),
          );
        },
      ),
      _ when profileResultsAsync.hasError => _ErrorView(
        message: l10n.searchLoadError,
        onRetry: () => ref.invalidate(provider),
      ),
      _ => const Center(child: StitchProgressIndicator()),
    };
  }
}

class _HashtagResultsList extends ConsumerWidget {
  const _HashtagResultsList({required this.query, required this.onOpenHashtag});

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
      AsyncValue(:final value?) => AutoPaginatedListView(
        itemCount: value.items.length,
        emptyText: l10n.searchEmptyTags,
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
      _ => const Center(child: StitchProgressIndicator()),
    };
  }
}
