import 'dart:async';

import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'report_profile_provider.g.dart';

@riverpod
class ReportProfile extends _$ReportProfile {
  @override
  FutureOr<ReportResult?> build() => null;

  Future<void> submit({
    required String handleOrDid,
    required ReportSubmission submission,
  }) async {
    if (state.isLoading) return;
    final ownership = captureActiveAccountOperation(ref);
    state = const AsyncLoading();
    final result = await AsyncValue.guard(() async {
      final repo = ref.read(profileRepositoryProvider);
      return repo.report(handleOrDid, submission);
    });
    if (!isActiveAccountOperationCurrent(ref, ownership)) return;
    state = result;
  }

  void reset() => state = const AsyncData(null);
}
