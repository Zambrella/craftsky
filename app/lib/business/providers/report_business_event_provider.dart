import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/business/providers/business_event_detail_provider.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'report_business_event_provider.g.dart';

@riverpod
class ReportBusinessEvent extends _$ReportBusinessEvent {
  @override
  FutureOr<ReportResult?> build(AccountKey account) => null;

  Future<void> submit({
    required Did owner,
    required RecordKey rkey,
    required ReportSubmission submission,
  }) async {
    if (state.isLoading) return;
    final ownership = captureActiveAccountOperation(ref);
    if (ownership != null && ownership.session.account != account) {
      throw StateError('Active account changed');
    }
    state = const AsyncLoading();
    final result = await AsyncValue.guard(
      () => ref
          .read(businessRepositoryProvider)
          .reportEvent(owner, rkey, submission),
    );
    if (!isActiveAccountOperationCurrent(ref, ownership)) return;
    if (result case AsyncError(
      :final error,
    ) when error is ApiBadRequest && error.code == 'event_not_found') {
      final target = BusinessEventDetailTarget(
        account: account,
        owner: owner,
        rkey: rkey,
      );
      ref.read(businessEventDetailProvider(target).notifier).markUnavailable();
    }
    state = result;
  }

  void reset() => state = const AsyncData(null);
}
