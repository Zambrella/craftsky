import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/shared/widgets/auto_paginated_list_view.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/shared/widgets/sort_menu_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class TagSearchPage extends ConsumerStatefulWidget {
  const TagSearchPage({required this.tag, super.key});

  final String tag;

  @override
  ConsumerState<TagSearchPage> createState() => _TagSearchPageState();
}

class _TagSearchPageState extends ConsumerState<TagSearchPage> {
  SearchSort _sort = SearchSort.chronological;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final provider = hashtagSearchProvider(
      HashtagSearchQuery(tag: widget.tag, sort: _sort),
    );
    final tagResultsAsync = ref.watch(provider);

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.tagSearchTitle(widget.tag)),
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
      body: switch (tagResultsAsync) {
        AsyncValue(:final value?) => _TagPostList(
          posts: value.items,
          isLoadingMore: tagResultsAsync.isLoading,
          hasLoadMoreError: tagResultsAsync.hasError,
          onNearEnd: () => ref.read(provider.notifier).loadMore(),
          onRefresh: () => ref.refresh(provider.future),
        ),
        _ when tagResultsAsync.hasError => Center(
          child: TextButton.icon(
            onPressed: () => ref.invalidate(provider),
            icon: const Icon(CraftskyIconsBold.refresh),
            label: Text(l10n.searchLoadError),
          ),
        ),
        _ => const Center(child: StitchProgressIndicator()),
      },
    );
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

class _TagPostList extends StatelessWidget {
  const _TagPostList({
    required this.posts,
    required this.isLoadingMore,
    required this.hasLoadMoreError,
    required this.onNearEnd,
    required this.onRefresh,
  });

  final List<Post> posts;
  final bool isLoadingMore;
  final bool hasLoadMoreError;
  final VoidCallback onNearEnd;
  final RefreshCallback onRefresh;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return RefreshIndicator(
      onRefresh: onRefresh,
      child: CustomScrollView(
        physics: const AlwaysScrollableScrollPhysics(),
        slivers: [
          AutoPaginatedSliverList(
            itemCount: posts.length,
            emptyState: CraftskyEmptyState(
              icon: CraftskyIcons.search,
              title: l10n.searchHashtagsHeading,
              subtitle: l10n.tagSearchEmpty,
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
          ),
        ],
      ),
    );
  }
}
