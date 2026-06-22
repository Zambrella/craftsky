import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/shared/widgets/auto_paginated_list_view.dart';
import 'package:craftsky_app/shared/widgets/sort_menu_button.dart';
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
    final async = ref.watch(provider);

    return Scaffold(
      appBar: AppBar(title: Text(l10n.tagSearchTitle(widget.tag))),
      body: switch (async) {
        AsyncValue(:final value?) => Column(
          children: [
            Padding(
              padding: EdgeInsets.fromLTRB(
                spacing.sp4,
                spacing.sp2,
                spacing.sp4,
                spacing.sp1,
              ),
              child: Align(
                alignment: Alignment.centerRight,
                child: SortMenuButton<SearchSort>(
                  selectedValue: _sort,
                  options: _sortOptions(l10n),
                  onChanged: (sort) => setState(() => _sort = sort),
                ),
              ),
            ),
            Expanded(
              child: _TagPostList(
                posts: value.items,
                isLoadingMore: async.isLoading,
                hasLoadMoreError: async.hasError,
                onNearEnd: () => ref.read(provider.notifier).loadMore(),
              ),
            ),
          ],
        ),
        _ when async.hasError => Center(
          child: TextButton.icon(
            onPressed: () => ref.invalidate(provider),
            icon: const Icon(Icons.refresh),
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
      emptyText: l10n.tagSearchEmpty,
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
