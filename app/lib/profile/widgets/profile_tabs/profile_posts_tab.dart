import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_post_feed_slivers.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Posts tab body. Returns slivers so it slots into the page's outer
/// [CustomScrollView] without nesting another scrollable.
class ProfilePostsTab extends ConsumerWidget {
  const ProfilePostsTab({
    required this.handle,
    required this.isOwnProfile,
    super.key,
  });

  final String handle;
  final bool isOwnProfile;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final postsAsync = ref.watch(userPostsProvider(handle));

    listenToProfilePostActions(context, ref);

    return switch (postsAsync) {
      AsyncValue(:final value?) => ProfilePostFeedSlivers(
        posts: value.items,
        hasMore: value.hasMore,
        isLoadingMore: postsAsync.isLoading,
        hasLoadMoreError: postsAsync.hasError,
        isOwnProfile: isOwnProfile,
        emptyText: l10n.profilePostsEmpty,
        showComposeButton: isOwnProfile,
        onLoadMore: () =>
            ref.read(userPostsProvider(handle).notifier).loadMore(),
        onReplacePost: (post) =>
            ref.read(userPostsProvider(handle).notifier).replace(post),
      ),
      AsyncError() => ProfileTabErrorSliver(
        message: l10n.profilePostsLoadError,
        showErrorIcon: true,
        onRetry: () => ref.invalidate(userPostsProvider(handle)),
      ),
      _ => const ProfileTabLoadingSliver(),
    };
  }
}
