import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_post_feed_slivers.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class ProfileProjectsTab extends ConsumerWidget {
  const ProfileProjectsTab({
    required this.handle,
    required this.isOwnProfile,
    super.key,
  });

  final String handle;
  final bool isOwnProfile;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final projectsAsync = ref.watch(userProjectsProvider(handle));

    listenToProfilePostActions(context, ref);

    return switch (projectsAsync) {
      AsyncValue(:final value?) => ProfilePostFeedSlivers(
        posts: value.items,
        hasMore: value.hasMore,
        isLoadingMore: projectsAsync.isLoading,
        hasLoadMoreError: projectsAsync.hasError,
        isOwnProfile: isOwnProfile,
        emptyText: l10n.profileEmptyProjects,
        onLoadMore: () =>
            ref.read(userProjectsProvider(handle).notifier).loadMore(),
        onReplacePost: (post) =>
            ref.read(userProjectsProvider(handle).notifier).replace(post),
      ),
      AsyncError(:final error) => ProfileTabErrorSliver(
        message: AppErrorMapper.map(
          error,
          fallbackKind: AppErrorKind.backgroundLoadFailed,
          source: 'background_load',
        ).message(l10n),
        onRetry: () => ref.invalidate(userProjectsProvider(handle)),
      ),
      _ => const ProfileTabLoadingSliver(),
    };
  }
}
