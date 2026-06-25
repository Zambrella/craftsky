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
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/widgets/auto_paginated_list_view.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

part 'blank_search_view.dart';
part 'search_results_tabs.dart';
part 'suggestion_list.dart';

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
    assert(query.isNotEmpty, 'query must not be empty');
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
    ref
      ..listen(saveRecentSearchProvider, (_, next) {
        if (next.hasError) context.showError(l10n.searchRecentSaveError);
      })
      ..listen(deleteRecentSearchProvider, (_, next) {
        if (next.hasError) context.showError(l10n.searchRecentDeleteError);
      });

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
                    child: BrandTextField(
                      label: l10n.searchTitle,
                      showLabel: false,
                      textFieldKey: const ValueKey('search-input'),
                      controller: _controller,
                      focusNode: _focusNode,
                      hintText: l10n.searchHint,
                      prefixIcon: const Icon(Icons.search),
                      suffixIcon: _draftQuery.isEmpty
                          ? null
                          : IconButton(
                              tooltip: l10n.searchClearAction,
                              icon: const Icon(Icons.cancel),
                              onPressed: _clearText,
                            ),
                      textInputAction: TextInputAction.search,
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
    final subtitle = profile.subtitle(context);
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

extension on ProfileSearchResult {
  String? subtitle(BuildContext context) {
    final name = displayName;
    final crafts = this.crafts
        .map((craft) => _optionLabel(ProjectOptionCatalogs.craftTypes, craft))
        .join(', ');
    if (name != null && name.isNotEmpty && crafts.isNotEmpty) {
      return AppLocalizations.of(
        context,
      ).searchProfileCraftSubtitle(name, crafts);
    }
    if (name != null && name.isNotEmpty) return name;
    if (crafts.isNotEmpty) return crafts;
    return description;
  }
}

String _optionLabel(Iterable<ProjectOption> options, String value) {
  for (final option in options) {
    if (option.value == value) return option.label;
  }
  final hash = value.lastIndexOf('#');
  if (hash >= 0 && hash < value.length - 1) return value.substring(hash + 1);
  return value;
}
