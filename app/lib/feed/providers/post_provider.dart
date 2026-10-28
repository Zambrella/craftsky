import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/like_post_overlay.dart';
import 'package:craftsky_app/feed/providers/post_record_overlay.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'post_provider.g.dart';

/// Single-post read by `(did, rkey)`. No UI consumer in v1; exists for
/// future routes (deep-link share, thread page).
@riverpod
Future<Post> post(Ref ref, Did did, RecordKey rkey) async {
  final accountOperation = ActiveAccountOperation.capture(ref);
  final result = await ref.watch(postRepositoryProvider).fetch(did, rkey);
  accountOperation.requireCurrent(discarded: const ApiCanceled());
  final current = applyPostRecordOverlay(ref, result);
  if (current == null) throw const ApiBadRequest('not_found');
  return applyPostInteractionOverlays(ref, current);
}
