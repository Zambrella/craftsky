import 'dart:async';

import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'report_post_provider.g.dart';

@riverpod
class ReportPost extends _$ReportPost {
  @override
  FutureOr<ReportResult?> build() => null;

  Future<void> submit({
    required Did did,
    required RecordKey rkey,
    required ReportSubmission submission,
  }) async {
    if (state.isLoading) return;
    final ownership = captureActiveAccountOperation(ref);
    state = const AsyncLoading();
    final result = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      return repo.report(did, rkey, submission);
    });
    if (!isActiveAccountOperationCurrent(ref, ownership)) return;
    state = result;
  }

  void reset() => state = const AsyncData(null);
}
