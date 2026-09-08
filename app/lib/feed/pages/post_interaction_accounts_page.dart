import 'dart:async';

import 'package:craftsky_app/feed/providers/post_interaction_lists_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_account_list_tile.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/widgets/auto_paginated_list_view.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/shared/widgets/craftsky_skeleton.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class PostInteractionAccountsPage extends ConsumerWidget {
  const PostInteractionAccountsPage({
    required this.did,
    required this.rkey,
    required this.kind,
    super.key,
  });

  final Did did;
  final RecordKey rkey;
  final PostInteractionAccountKind kind;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final provider = postInteractionAccountsProvider(did, rkey, kind);
    final accounts = ref.watch(provider);
    final data = accounts.value;
    final l10n = AppLocalizations.of(context);

    return Scaffold(
      appBar: AppBar(title: Text(_title(l10n, data?.totalCount))),
      body: switch ((data, accounts.error)) {
        (null, null) => CraftskySkeletonList(
          itemBuilder: (context, index) => const AccountRowSkeleton(),
        ),
        (null, final error?) => _InitialError(
          error: error,
          onRetry: () => unawaited(ref.read(provider.notifier).refresh()),
        ),
        (final state?, _) => AutoPaginatedListView(
          itemCount: state.items.length,
          emptyState: _emptyState(context, l10n),
          isLoadingMore: accounts.isLoading && state.hasMore,
          hasLoadMoreError: accounts.hasError && state.hasMore,
          onNearEnd: () => unawaited(ref.read(provider.notifier).loadMore()),
          itemBuilder: (_, index) =>
              ProfileAccountListTile(account: state.items[index]),
        ),
      },
    );
  }

  String _plainTitle(AppLocalizations l10n) => switch (kind) {
    PostInteractionAccountKind.likes => l10n.postInteractionLikesTitle,
    PostInteractionAccountKind.reposts => l10n.postInteractionRepostsTitle,
  };

  Widget _emptyState(BuildContext context, AppLocalizations l10n) {
    final subtitle = switch (kind) {
      PostInteractionAccountKind.likes => l10n.postInteractionLikesEmpty,
      PostInteractionAccountKind.reposts => l10n.postInteractionRepostsEmpty,
    };
    return Semantics(
      label: subtitle,
      liveRegion: true,
      container: true,
      excludeSemantics: true,
      child: CraftskyEmptyState(
        icon: switch (kind) {
          PostInteractionAccountKind.likes => CraftskyIcons.like,
          PostInteractionAccountKind.reposts => CraftskyIcons.repost,
        },
        title: _plainTitle(l10n),
        subtitle: subtitle,
      ),
    );
  }

  String _title(AppLocalizations l10n, int? totalCount) {
    if (totalCount == null) return _plainTitle(l10n);
    return switch (kind) {
      PostInteractionAccountKind.likes =>
        l10n.postInteractionLikesTitleWithCount(totalCount),
      PostInteractionAccountKind.reposts =>
        l10n.postInteractionRepostsTitleWithCount(totalCount),
    };
  }
}

class _InitialError extends StatelessWidget {
  const _InitialError({required this.error, required this.onRetry});

  final Object error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final unavailable = switch (error) {
      ApiBadRequest(:final code, :final details) =>
        code == 'post_not_found' && details.statusCode == 404,
      _ => false,
    };
    return Center(
      child: Padding(
        padding: EdgeInsets.all(spacing.sp5),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Semantics(
              liveRegion: true,
              child: Text(
                unavailable
                    ? l10n.postUnavailablePlaceholder
                    : l10n.errorBackgroundLoadFailed,
                style: Theme.of(context).textTheme.titleLarge,
                textAlign: TextAlign.center,
              ),
            ),
            SizedBox(height: spacing.sp3),
            if (unavailable)
              TextButton(
                onPressed: () {
                  final navigator = Navigator.of(context);
                  if (navigator.canPop()) {
                    navigator.pop();
                  } else {
                    const FeedRoute().go(context);
                  }
                },
                child: Text(l10n.backButton),
              )
            else
              TextButton.icon(
                onPressed: onRetry,
                icon: const Icon(CraftskyIconsBold.refresh),
                label: Text(l10n.retryButton),
              ),
          ],
        ),
      ),
    );
  }
}
