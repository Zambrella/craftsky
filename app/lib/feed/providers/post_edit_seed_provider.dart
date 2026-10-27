import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'post_edit_seed_provider.g.dart';

/// Editor state belongs to one activation, including a same-account re-login.
@riverpod
Future<PostEditSeed> postEditSeed(
  Ref ref,
  ActiveAccountLease ownership,
  Did did,
  RecordKey key, {
  String? requestId,
}) async {
  ref.watch(
    sessionRegistryProvider.select((state) => state.value?.activeLease),
  );
  final operation = ActiveAccountOperation.fromLease(ref, ownership)
    ..requireCurrent();
  if (ownership.session.account.did != did) {
    throw const AccountOperationDiscarded();
  }
  final seed = await ref.watch(postRepositoryProvider).fetchEdit(did, key);
  operation.requireCurrent();
  if (seed.post.author.did != did || seed.post.rkey != key) {
    throw const FormatException('Post edit seed identity mismatch');
  }
  return seed;
}
