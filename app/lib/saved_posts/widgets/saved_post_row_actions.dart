import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_error.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/navigation/saved_post_destination.dart';
import 'package:craftsky_app/saved_posts/providers/account_saved_post_state_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_posts_provider.dart';
import 'package:craftsky_app/saved_posts/widgets/save_post_dialog.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

void openSavedPost(BuildContext context, SavedPostItem item) {
  final destination = SavedPostDestination.forItem(item);
  final parts = parseCraftskyPostUri(destination.threadUri);
  if (parts == null) return;
  unawaited(
    PostThreadRoute(
      did: parts.did,
      rkey: parts.rkey,
      focus: destination.focusUri?.toString(),
    ).push<void>(context),
  );
}

Future<void> moveSavedPost(
  BuildContext context,
  WidgetRef ref, {
  required AccountKey account,
  required SavedPostItem item,
  required SavedPostListKey sourceKey,
}) async {
  final moved = await showMoveSavedPostDialog(
    context,
    account: account,
    item: item,
  );
  if (moved != true || !context.mounted) return;
  final presentation = ref
      .read(
        savedPostPresentationProvider(
          SavedPostKey(account: account, uri: item.post.uri),
        ),
      )
      .value;
  if (presentation == null ||
      !presentation.isSaved ||
      presentation.savedAt == null ||
      presentation.folderId == sourceKey.scope.folderId) {
    return;
  }

  final destinationFolderId = presentation.folderId;
  final destinationScope = destinationFolderId == null
      ? const SavedPostScope.unfiled()
      : SavedPostScope.folder(destinationFolderId);
  final confirmedItem = SavedPostItem(
    post: item.post,
    savedAt: presentation.savedAt!,
    folderId: destinationFolderId,
  );
  if (ref.exists(savedPostsProvider(sourceKey))) {
    ref
        .read(savedPostsProvider(sourceKey).notifier)
        .removeConfirmed(item.post.uri);
  }
  for (final sort in SavedPostSort.values) {
    final destinationKey = SavedPostListKey(
      account: account,
      scope: destinationScope,
      sort: sort,
    );
    if (ref.exists(savedPostsProvider(destinationKey))) {
      ref
          .read(savedPostsProvider(destinationKey).notifier)
          .upsertConfirmed(confirmedItem);
    }
  }
}

Future<void> unsaveSavedPost(
  BuildContext context,
  WidgetRef ref, {
  required AccountKey account,
  required SavedPostItem item,
  required SavedPostListKey sourceKey,
}) async {
  await ref
      .read(accountSavedPostStateProvider(account).notifier)
      .unsave(item.post);
  if (!context.mounted) return;
  final presentation = ref
      .read(
        savedPostPresentationProvider(
          SavedPostKey(account: account, uri: item.post.uri),
        ),
      )
      .value;
  if (presentation != null && !presentation.isSaved && !presentation.hasError) {
    ref
        .read(savedPostsProvider(sourceKey).notifier)
        .removeConfirmed(item.post.uri);
  } else if (presentation?.lastError case final error?) {
    final failure = SavedPostFailure.from(
      error,
      operation: SavedPostOperation.unsave,
    );
    if (failure.shouldPresent) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            failure.localizedMessage(AppLocalizations.of(context)),
          ),
        ),
      );
    }
  }
}
