import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_provider.dart';
import 'package:craftsky_app/feed/providers/post_record_overlay.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

typedef PostEditRefresh = void Function(
  Post post,
  ActiveAccountLease? ownership,
);

/// Revalidate the edited record without rebuilding accumulated list providers.
/// It outlives the composer so refreshes can finish after navigation.
final postEditRefreshProvider = Provider<PostEditRefresh>((ref) {
  final pending = <PdsMutationScope>{};
  return (post, ownership) {
    if (!ref.mounted) return;
    final operation = ActiveAccountOperation.fromLease(ref, ownership);
    if (!operation.isCurrent) return;
    final scope = postRecordMutationScope(ref, post);
    if (!pending.add(scope)) return;
    final provider = postProvider(post.author.did, post.rkey);
    ref.invalidate(provider);
    final subscription = ref.listen(provider, (_, _) {});
    unawaited(
      ref
          .read(provider.future)
          .then<void>(
            (_) {
              if (ref.mounted && operation.isCurrent) {
                notifyPostRecordOverlays(ref);
              }
            },
            onError: (Object error, StackTrace stack) {
              // The public read provider owns diagnostics. Refresh failures
              // cannot fail a confirmed write or discard loaded pages.
            },
          )
          .whenComplete(() {
            pending.remove(scope);
            subscription.close();
          }),
    );
  };
});
