import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_error.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/account_saved_post_state_provider.dart';
import 'package:craftsky_app/saved_posts/widgets/save_post_dialog.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SavedPostBookmarkButton extends ConsumerWidget {
  const SavedPostBookmarkButton({
    required this.account,
    required this.post,
    super.key,
  });

  final AccountKey account;
  final Post post;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final key = SavedPostKey(account: account, uri: post.uri);
    final projected = ref.watch(savedPostPresentationProvider(key));
    final presentation = projected.value;
    if (!(presentation?.initialized ?? false)) {
      unawaited(
        Future<void>.microtask(
          () => ref
              .read(accountSavedPostStateProvider(account).notifier)
              .seedIfAbsent(post),
        ),
      );
    }
    final isSaved = presentation?.initialized ?? false
        ? presentation!.isSaved
        : post.viewerHasSaved;
    final isPending = presentation?.isPending ?? false;
    final l10n = AppLocalizations.of(context);

    return SizedBox.square(
      dimension: 48,
      child: IconButton(
        isSelected: isSaved,
        icon: const Icon(Icons.bookmark_border),
        selectedIcon: const Icon(Icons.bookmark),
        tooltip: isSaved
            ? l10n.savedPostUnsaveAction
            : l10n.savedPostSaveAction,
        onPressed: isPending
            ? null
            : () async {
                if (isSaved) {
                  await ref
                      .read(accountSavedPostStateProvider(account).notifier)
                      .unsave(post);
                  if (!context.mounted) return;
                  final result = ref
                      .read(savedPostPresentationProvider(key))
                      .value;
                  final failure = result?.lastError == null
                      ? null
                      : SavedPostFailure.from(
                          result!.lastError!,
                          operation: SavedPostOperation.unsave,
                        );
                  if (failure?.shouldPresent ?? false) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(content: Text(failure!.localizedMessage(l10n))),
                    );
                  }
                  return;
                }
                unawaited(
                  showSavePostDialog(
                    context,
                    account: account,
                    post: post,
                  ),
                );
              },
      ),
    );
  }
}
