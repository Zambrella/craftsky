import 'package:craftsky_app/feed/providers/user_reposts_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_post_feed_slivers.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class ProfileRepostsTab extends ConsumerWidget {
  const ProfileRepostsTab({required this.did, super.key});

  final Did did;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final repostsAsync = ref.watch(userRepostsProvider(did));

    listenToProfilePostActions(context, ref);

    return switch (repostsAsync) {
      AsyncValue(:final value?) => ProfilePostFeedSlivers(
        posts: value.items,
        hasMore: value.hasMore,
        isLoadingMore: repostsAsync.isLoading,
        hasLoadMoreError: repostsAsync.hasError,
        isOwnProfile: false,
        pinnedPostUri: null,
        emptyState: CraftskyEmptyState(
          icon: CraftskyIcons.repost,
          title: l10n.profileTabReposts,
          subtitle: l10n.profileEmptyReposts,
        ),
        onLoadMore: () =>
            ref.read(userRepostsProvider(did).notifier).loadMore(),
        onReplyCreated: () => ref.invalidate(userRepostsProvider(did)),
      ),
      AsyncError() => ProfileTabErrorSliver(
        message: l10n.profileRepostsLoadError,
        showErrorIcon: true,
        onRetry: () => ref.invalidate(userRepostsProvider(did)),
      ),
      _ => const ProfileTabLoadingSliver(),
    };
  }
}
