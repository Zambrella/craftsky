import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/search/models/blank_search_data.dart';
import 'package:craftsky_app/search/models/hashtag_search_page.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/models/search_results_tab.dart';
import 'package:craftsky_app/search/models/search_suggestions.dart';
import 'package:craftsky_app/search/providers/blank_search_provider.dart';
import 'package:craftsky_app/search/providers/hashtag_result_search_provider.dart';
import 'package:craftsky_app/search/providers/post_search_provider.dart';
import 'package:craftsky_app/search/providers/profile_search_provider.dart';
import 'package:craftsky_app/search/providers/project_search_provider.dart';
import 'package:craftsky_app/search/providers/recent_searches_provider.dart';
import 'package:craftsky_app/search/providers/search_suggestions_provider.dart';
import 'package:craftsky_app/shared/widgets/auto_paginated_list_view.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

// TODO(Agent): Split this down so widgets are in their own files.

const _suggestionDebounce = Duration(milliseconds: 300);

class SearchPage extends ConsumerStatefulWidget {
  const SearchPage({super.key, this.q, this.tab});

  final String? q;
  final SearchResultsTab? tab;

  @override
  ConsumerState<SearchPage> createState() => _SearchPageState();
}

class _SearchPageState extends ConsumerState<SearchPage> {
  late final TextEditingController _controller;
  late final FocusNode _focusNode;
  Timer? _debounce;
  String _draftQuery = '';
  String _debouncedQuery = '';

  bool get _hasSubmittedQuery => (widget.q ?? '').trim().isNotEmpty;

  @override
  void initState() {
    super.initState();
    _draftQuery = widget.q ?? '';
    _debouncedQuery = _draftQuery;
    _controller = TextEditingController(text: _draftQuery);
    _focusNode = FocusNode()..addListener(_handleFocusChange);
  }

  @override
  void didUpdateWidget(covariant SearchPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.q != widget.q) {
      final next = widget.q ?? '';
      _debounce?.cancel();
      _draftQuery = next;
      _debouncedQuery = next;
      _controller.value = TextEditingValue(
        text: next,
        selection: TextSelection.collapsed(offset: next.length),
      );
      if (next.isNotEmpty) _focusNode.unfocus();
    }
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _focusNode
      ..removeListener(_handleFocusChange)
      ..dispose();
    _controller.dispose();
    super.dispose();
  }

  void _handleFocusChange() => setState(() {});

  void _onQueryChanged(String value) {
    setState(() => _draftQuery = value);
    _debounce?.cancel();
    _debounce = Timer(_suggestionDebounce, () {
      if (!mounted) return;
      setState(() => _debouncedQuery = value);
    });
  }

  void _clearText() {
    _debounce?.cancel();
    _controller.clear();
    setState(() {
      _draftQuery = '';
      _debouncedQuery = '';
    });
    _focusNode.requestFocus();
  }

  void _cancelSearch() {
    _debounce?.cancel();
    _controller.clear();
    setState(() {
      _draftQuery = '';
      _debouncedQuery = '';
    });
    _focusNode.unfocus();
    const SearchRoute().go(context);
  }

  Future<void> _submitQuery([String? value]) async {
    final query = (value ?? _controller.text).trim();
    if (query.isEmpty) return;
    await _saveQueryRecent(query);
    if (!mounted) return;
    SearchRoute(q: query).go(context);
  }

  Future<void> _saveQueryRecent(String query) {
    return ref
        .read(saveRecentSearchProvider.notifier)
        .save(
          SaveRecentSearchRequest(
            type: RecentSearchType.query,
            displayLabel: query,
            payload: QueryRecentSearchPayload(q: query),
          ),
        );
  }

  Future<void> _openResultTab(SearchResultsTab tab) async {
    final query = _controller.text.trim();
    if (query.isEmpty) return;
    await _saveQueryRecent(query);
    if (!mounted) return;
    SearchRoute(q: query, tab: tab).go(context);
  }

  Future<void> _openProfile(ProfileSearchResult profile) async {
    await ref
        .read(saveRecentSearchProvider.notifier)
        .save(
          SaveRecentSearchRequest(
            type: RecentSearchType.profile,
            displayLabel: '@${profile.handle}',
            payload: ProfileRecentSearchPayload(
              did: profile.did.toString(),
              handle: profile.handle.toString(),
              displayName: profile.displayName,
              avatar: profile.avatar,
            ),
          ),
        );
    if (!mounted) return;
    await UserProfileRoute(handle: profile.handle.toString()).push<void>(
      context,
    );
  }

  Future<void> _openHashtag(String tag) async {
    await _saveHashtagRecent(tag);
    if (!mounted) return;
    await TagSearchRoute(tag: tag).push<void>(context);
  }

  Future<void> _saveHashtagRecent(String tag) {
    return ref
        .read(saveRecentSearchProvider.notifier)
        .save(
          SaveRecentSearchRequest(
            type: RecentSearchType.hashtag,
            displayLabel: '#$tag',
            payload: HashtagRecentSearchPayload(tag: tag),
          ),
        );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final draft = _draftQuery.trim();
    final showSuggestions = !_hasSubmittedQuery && _focusNode.hasFocus;
    final body = switch ((_hasSubmittedQuery, showSuggestions, draft.isEmpty)) {
      (true, _, _) => _SearchResultsTabs(
        query: widget.q!.trim(),
        initialTab: widget.tab ?? SearchResultsTab.posts,
        onOpenHashtag: _openHashtag,
      ),
      (false, true, true) => const SizedBox.shrink(),
      (false, true, false) => _SuggestionList(
        query: _debouncedQuery.trim(),
        isWaitingForDebounce: _debouncedQuery.trim() != draft,
        onOpenProfile: _openProfile,
        onOpenHashtag: _openHashtag,
        onViewAllProfiles: () => _openResultTab(SearchResultsTab.profiles),
        onViewAllHashtags: () => _openResultTab(SearchResultsTab.tags),
      ),
      _ => _BlankSearchView(
        onOpenQuery: (query) => SearchRoute(q: query).go(context),
        onOpenHashtag: (tag) => TagSearchRoute(tag: tag).push<void>(context),
        onOpenProfile: (handle) => UserProfileRoute(handle: handle).push<void>(
          context,
        ),
      ),
    };

    return Scaffold(
      appBar: AppBar(title: Text(l10n.searchTitle)),
      body: SafeArea(
        child: Column(
          children: [
            Padding(
              padding: EdgeInsets.fromLTRB(
                spacing.sp4,
                spacing.sp3,
                spacing.sp4,
                spacing.sp2,
              ),
              child: Row(
                children: [
                  Expanded(
                    child: TextField(
                      key: const ValueKey('search-input'),
                      controller: _controller,
                      focusNode: _focusNode,
                      textInputAction: TextInputAction.search,
                      decoration: InputDecoration(
                        hintText: l10n.searchHint,
                        prefixIcon: const Icon(Icons.search),
                        suffixIcon: _draftQuery.isEmpty
                            ? null
                            : IconButton(
                                tooltip: l10n.searchClearAction,
                                icon: const Icon(Icons.cancel),
                                onPressed: _clearText,
                              ),
                      ),
                      onChanged: _onQueryChanged,
                      onSubmitted: _submitQuery,
                    ),
                  ),
                  if (_focusNode.hasFocus) ...[
                    SizedBox(width: spacing.sp2),
                    TextButton(
                      onPressed: _cancelSearch,
                      child: Text(l10n.searchCancelAction),
                    ),
                  ],
                ],
              ),
            ),
            Expanded(child: body),
          ],
        ),
      ),
    );
  }
}

class _BlankSearchView extends ConsumerWidget {
  const _BlankSearchView({
    required this.onOpenQuery,
    required this.onOpenHashtag,
    required this.onOpenProfile,
  });

  final ValueChanged<String> onOpenQuery;
  final ValueChanged<String> onOpenHashtag;
  final ValueChanged<String> onOpenProfile;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final blankAsync = ref.watch(blankSearchProvider);
    return switch (blankAsync) {
      AsyncValue(:final value?) => ListView(
        padding: EdgeInsets.fromLTRB(
          spacing.sp4,
          spacing.sp2,
          spacing.sp4,
          spacing.sp5,
        ),
        children: [
          _RecentSearchSection(
            data: value,
            onOpenQuery: onOpenQuery,
            onOpenHashtag: onOpenHashtag,
            onOpenProfile: onOpenProfile,
          ),
          SizedBox(height: spacing.sp5),
          _TopHashtagSection(data: value, onOpenHashtag: onOpenHashtag),
        ],
      ),
      _ when blankAsync.hasError => _ErrorView(
        message: l10n.searchLoadError,
        onRetry: () => ref.invalidate(blankSearchProvider),
      ),
      _ => const Center(child: StitchProgressIndicator()),
    };
  }
}

class _RecentSearchSection extends ConsumerWidget {
  const _RecentSearchSection({
    required this.data,
    required this.onOpenQuery,
    required this.onOpenHashtag,
    required this.onOpenProfile,
  });

  final BlankSearchData data;
  final ValueChanged<String> onOpenQuery;
  final ValueChanged<String> onOpenHashtag;
  final ValueChanged<String> onOpenProfile;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    if (data.recentSearches.items.isEmpty) return const SizedBox.shrink();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _SectionTitle(title: l10n.searchRecentHeading),
        for (final recent in data.recentSearches.items)
          ListTile(
            dense: true,
            contentPadding: EdgeInsets.zero,
            title: Text(recent.displayLabel),
            trailing: IconButton(
              tooltip: l10n.searchDeleteRecentAction,
              icon: const Icon(Icons.close),
              onPressed: () => ref
                  .read(deleteRecentSearchProvider.notifier)
                  .delete(recent.id),
            ),
            onTap: () => _openRecent(recent),
          ),
      ],
    );
  }

  void _openRecent(RecentSearchItem recent) {
    switch (recent.payload) {
      case QueryRecentSearchPayload(:final q):
        onOpenQuery(q);
      case HashtagRecentSearchPayload(:final tag):
        onOpenHashtag(tag);
      case ProfileRecentSearchPayload(:final handle):
        onOpenProfile(handle);
      case PostRecentSearchPayload(:final q):
        onOpenQuery(q);
      case ProjectRecentSearchPayload(:final q):
        if (q != null && q.isNotEmpty) onOpenQuery(q);
    }
  }
}

class _TopHashtagSection extends StatelessWidget {
  const _TopHashtagSection({required this.data, required this.onOpenHashtag});

  final BlankSearchData data;
  final ValueChanged<String> onOpenHashtag;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final groups = data.topHashtags.groups;
    if (groups.isEmpty) return const SizedBox.shrink();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _SectionTitle(title: l10n.searchTrendingHashtagsHeading),
        for (final group in groups) ...[
          SizedBox(height: spacing.sp2),
          Text(
            _optionLabel(ProjectOptionCatalogs.craftTypes, group.craftType),
            style: Theme.of(context).textTheme.titleSmall,
          ),
          SizedBox(height: spacing.sp2),
          Wrap(
            spacing: spacing.sp2,
            runSpacing: spacing.sp2,
            children: [
              for (final item in group.items)
                ActionChip(
                  label: Text('#${item.tag}'),
                  onPressed: () => onOpenHashtag(item.tag),
                ),
            ],
          ),
        ],
      ],
    );
  }
}

class _SuggestionList extends ConsumerWidget {
  const _SuggestionList({
    required this.query,
    required this.isWaitingForDebounce,
    required this.onOpenProfile,
    required this.onOpenHashtag,
    required this.onViewAllProfiles,
    required this.onViewAllHashtags,
  });

  final String query;
  final bool isWaitingForDebounce;
  final ValueChanged<ProfileSearchResult> onOpenProfile;
  final ValueChanged<String> onOpenHashtag;
  final VoidCallback onViewAllProfiles;
  final VoidCallback onViewAllHashtags;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    if (isWaitingForDebounce || query.isEmpty) {
      return const Center(child: StitchProgressIndicator());
    }
    final suggestionsAsync = ref.watch(
      searchSuggestionsProvider(SearchSuggestionQuery(q: query)),
    );
    return switch (suggestionsAsync) {
      AsyncValue(:final value?) => ListView(
        padding: EdgeInsets.fromLTRB(
          spacing.sp4,
          spacing.sp2,
          spacing.sp4,
          spacing.sp5,
        ),
        children: [
          _SuggestionProfileSection(
            suggestions: value,
            onOpenProfile: onOpenProfile,
            onViewAll: onViewAllProfiles,
          ),
          SizedBox(height: spacing.sp3),
          _SuggestionHashtagSection(
            suggestions: value,
            onOpenHashtag: onOpenHashtag,
            onViewAll: onViewAllHashtags,
          ),
        ],
      ),
      _ when suggestionsAsync.hasError => _ErrorView(
        message: l10n.searchLoadError,
        onRetry: () => ref.invalidate(
          searchSuggestionsProvider(SearchSuggestionQuery(q: query)),
        ),
      ),
      _ => const Center(child: StitchProgressIndicator()),
    };
  }
}

class _SuggestionProfileSection extends StatelessWidget {
  const _SuggestionProfileSection({
    required this.suggestions,
    required this.onOpenProfile,
    required this.onViewAll,
  });

  final SearchSuggestions suggestions;
  final ValueChanged<ProfileSearchResult> onOpenProfile;
  final VoidCallback onViewAll;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return _SuggestionSectionScaffold(
      title: l10n.searchProfilesHeading,
      hasMore: suggestions.profiles.hasMore,
      onViewAll: onViewAll,
      children: [
        for (final profile in suggestions.profiles.items)
          _ProfileResultTile(
            profile: profile,
            onTap: () => onOpenProfile(profile),
          ),
      ],
    );
  }
}

class _SuggestionHashtagSection extends StatelessWidget {
  const _SuggestionHashtagSection({
    required this.suggestions,
    required this.onOpenHashtag,
    required this.onViewAll,
  });

  final SearchSuggestions suggestions;
  final ValueChanged<String> onOpenHashtag;
  final VoidCallback onViewAll;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return _SuggestionSectionScaffold(
      title: l10n.searchHashtagsHeading,
      hasMore: suggestions.hashtags.hasMore,
      onViewAll: onViewAll,
      children: [
        for (final hashtag in suggestions.hashtags.items)
          _HashtagResultTile(
            hashtag: hashtag,
            onTap: () => onOpenHashtag(hashtag.tag),
          ),
      ],
    );
  }
}

class _SuggestionSectionScaffold extends StatelessWidget {
  const _SuggestionSectionScaffold({
    required this.title,
    required this.hasMore,
    required this.onViewAll,
    required this.children,
  });

  final String title;
  final bool hasMore;
  final VoidCallback onViewAll;
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    if (children.isEmpty) return const SizedBox.shrink();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            Expanded(child: _SectionTitle(title: title)),
            if (hasMore)
              TextButton(
                onPressed: onViewAll,
                child: Text(l10n.searchViewAllAction),
              ),
          ],
        ),
        ...children,
      ],
    );
  }
}

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
    final async = ref.watch(provider);
    return switch (async) {
      AsyncValue(:final value?) => _PostList(
        posts: value.items,
        emptyText: l10n.searchEmptyPosts,
        isLoadingMore: async.isLoading,
        hasLoadMoreError: async.hasError,
        onNearEnd: () => ref.read(provider.notifier).loadMore(),
      ),
      _ when async.hasError => _ErrorView(
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
    final async = ref.watch(provider);
    return switch (async) {
      AsyncValue(:final value?) => _PostList(
        posts: value.items,
        emptyText: l10n.searchEmptyProjects,
        isLoadingMore: async.isLoading,
        hasLoadMoreError: async.hasError,
        onNearEnd: () => ref.read(provider.notifier).loadMore(),
      ),
      _ when async.hasError => _ErrorView(
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
    final async = ref.watch(provider);
    return switch (async) {
      AsyncValue(:final value?) => AutoPaginatedListView(
        itemCount: value.items.length,
        emptyText: l10n.searchEmptyProfiles,
        isLoadingMore: async.isLoading,
        hasLoadMoreError: async.hasError,
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
      _ when async.hasError => _ErrorView(
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
    final async = ref.watch(provider);
    return switch (async) {
      AsyncValue(:final value?) => AutoPaginatedListView(
        itemCount: value.items.length,
        emptyText: l10n.searchEmptyTags,
        isLoadingMore: async.isLoading,
        hasLoadMoreError: async.hasError,
        onNearEnd: () => ref.read(provider.notifier).loadMore(),
        itemBuilder: (context, index) {
          final hashtag = value.items[index];
          return _HashtagResultTile(
            hashtag: hashtag,
            onTap: () => onOpenHashtag(hashtag.tag),
          );
        },
      ),
      _ when async.hasError => _ErrorView(
        message: l10n.searchLoadError,
        onRetry: () => ref.invalidate(provider),
      ),
      _ => const Center(child: StitchProgressIndicator()),
    };
  }
}

class _PostList extends StatelessWidget {
  const _PostList({
    required this.posts,
    required this.emptyText,
    required this.isLoadingMore,
    required this.hasLoadMoreError,
    required this.onNearEnd,
  });

  final List<Post> posts;
  final String emptyText;
  final bool isLoadingMore;
  final bool hasLoadMoreError;
  final VoidCallback onNearEnd;

  @override
  Widget build(BuildContext context) {
    return AutoPaginatedListView(
      itemCount: posts.length,
      emptyText: emptyText,
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

class _ProfileResultTile extends StatelessWidget {
  const _ProfileResultTile({required this.profile, required this.onTap});

  final ProfileSearchResult profile;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final title = '@${profile.handle}';
    final subtitle = _profileSubtitle(context, profile);
    return ListTile(
      contentPadding: EdgeInsets.zero,
      leading: ProfileAvatar(
        seed: profile.displayName ?? profile.handle,
        size: ProfileAvatarSize.small,
      ),
      title: Text(title),
      subtitle: subtitle == null ? null : Text(subtitle),
      onTap: onTap,
    );
  }
}

class _HashtagResultTile extends StatelessWidget {
  const _HashtagResultTile({required this.hashtag, required this.onTap});

  final HashtagSearchResult hashtag;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return ListTile(
      contentPadding: EdgeInsets.zero,
      leading: const CircleAvatar(child: Text('#')),
      title: Text('#${hashtag.tag}'),
      subtitle: Text(l10n.searchTagPostCount(hashtag.postsLast28Days)),
      onTap: onTap,
    );
  }
}

class _SectionTitle extends StatelessWidget {
  const _SectionTitle({required this.title});

  final String title;

  @override
  Widget build(BuildContext context) {
    return Text(title, style: Theme.of(context).textTheme.titleMedium);
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(message),
          SizedBox(height: spacing.sp2),
          TextButton.icon(
            onPressed: onRetry,
            icon: const Icon(Icons.refresh),
            label: Text(l10n.retryButton),
          ),
        ],
      ),
    );
  }
}

String? _profileSubtitle(BuildContext context, ProfileSearchResult profile) {
  final name = profile.displayName;
  final crafts = profile.crafts
      .map((craft) => _optionLabel(ProjectOptionCatalogs.craftTypes, craft))
      .join(', ');
  if (name != null && name.isNotEmpty && crafts.isNotEmpty) {
    return AppLocalizations.of(
      context,
    ).searchProfileCraftSubtitle(name, crafts);
  }
  if (name != null && name.isNotEmpty) return name;
  if (crafts.isNotEmpty) return crafts;
  return profile.description;
}

String _optionLabel(Iterable<ProjectOption> options, String value) {
  for (final option in options) {
    if (option.value == value) return option.label;
  }
  final hash = value.lastIndexOf('#');
  if (hash >= 0 && hash < value.length - 1) return value.substring(hash + 1);
  return value;
}
