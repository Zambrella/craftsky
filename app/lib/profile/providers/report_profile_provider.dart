import 'dart:async';

import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/profile/providers/profile_api_client_provider.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'report_profile_provider.g.dart';

@riverpod
class ReportProfile extends _$ReportProfile {
  @override
  FutureOr<ReportResult?> build() => null;

  Future<void> submit({
    required Did did,
    required ReportSubmission submission,
  }) async {
    if (state.isLoading) return;
    final accountOperation = ActiveAccountOperation.capture(ref);
    state = const AsyncLoading();
    final result = await AsyncValue.guard(() async {
      final repo = ref.read(profileRepositoryProvider);
      return await repo.report(did.toString(), submission);
    });
    accountOperation.publish(() => state = result);
  }

  void reset() => state = const AsyncData(null);
}

@riverpod
class ReportProfileLink extends _$ReportProfileLink {
  @override
  FutureOr<ReportResult?> build() => null;

  Future<void> submit({
    required String uri,
    required ReportSubmission submission,
  }) async {
    if (state.isLoading) return;
    final operation = ActiveAccountOperation.capture(ref);
    final api = ref.read(profileApiClientProvider);
    state = const AsyncLoading();
    final result = await AsyncValue.guard(
      () => api.reportLink(uri, submission),
    );
    operation.publish(() => state = result);
  }

  void reset() => state = const AsyncData(null);
}
