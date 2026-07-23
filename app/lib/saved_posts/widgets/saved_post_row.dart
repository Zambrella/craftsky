import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/account_saved_post_state_provider.dart';
import 'package:craftsky_app/shared/time/relative_time_text.dart';
import 'package:craftsky_app/shared/widgets/post_summary.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SavedPostRow extends ConsumerWidget {
  const SavedPostRow({
    required this.account,
    required this.item,
    required this.onOpen,
    required this.onMove,
    required this.onUnsave,
    super.key,
  });

  final AccountKey account;
  final SavedPostItem item;
  final VoidCallback onOpen;
  final VoidCallback onMove;
  final VoidCallback onUnsave;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final key = SavedPostKey(account: account, uri: item.post.uri);
    final presentation = ref.watch(savedPostPresentationProvider(key)).value;
    if (!(presentation?.initialized ?? false)) {
      unawaited(
        Future<void>.microtask(
          () => ref
              .read(accountSavedPostStateProvider(account).notifier)
              .reconcileSavedItem(item),
        ),
      );
    } else if (!presentation!.isSaved) {
      return const SizedBox.shrink();
    }
    final l10n = AppLocalizations.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        PostSummary(
          onTap: onOpen,
          data: PostSummaryData.savedPost(item.post),
        ),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 8),
          child: Wrap(
            alignment: WrapAlignment.end,
            crossAxisAlignment: WrapCrossAlignment.center,
            spacing: 4,
            runSpacing: 4,
            children: [
              RelativeTimeText(timestamp: item.savedAt),
              TextButton(
                onPressed: onMove,
                child: Text(l10n.savedPostMoveAction),
              ),
              TextButton(
                onPressed: onUnsave,
                child: Text(l10n.savedPostRowUnsaveAction),
              ),
            ],
          ),
        ),
        const Divider(height: 1),
      ],
    );
  }
}
