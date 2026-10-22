import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_post_feed_slivers.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

/// Posts tab body. Returns slivers so it slots into the page's outer
/// [CustomScrollView] without nesting another scrollable.
class ProfilePostsTab extends ConsumerWidget {
  const new({
    required this.did,
    required this.isOwnProfile,
    super.key,
  });

  final Did did;
  final bool isOwnProfile;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final postsAsync = ref.watch(userPostsProvider(did));

    listenToProfilePostActions(context, ref);

    return switch (postsAsync) {
      AsyncValue(:final value?) => ProfilePostFeedSlivers(
        posts: value.items,
        hasMore: value.hasMore,
        isLoadingMore: postsAsync.isLoading,
        hasLoadMoreError: postsAsync.hasError,
        isOwnProfile: isOwnProfile,
        pinnedPostUri: value.pinnedPostUri,
        emptyState: CraftskyEmptyState(
          icon: CraftskyIcons.textPost,
          title: l10n.profileTabPosts,
          subtitle: l10n.profilePostsEmpty,
        ),
        onLoadMore: () => ref.read(userPostsProvider(did).notifier).loadMore(),
      ),
      AsyncError() => ProfileTabErrorSliver(
        message: l10n.profilePostsLoadError,
        showErrorIcon: true,
        onRetry: () => ref.invalidate(userPostsProvider(did)),
      ),
      _ => const ProfileTabLoadingSliver(),
    };
  }
}
