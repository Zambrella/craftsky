import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/providers/business_record_overlay.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'business_event_detail_provider.g.dart';

@immutable
final class BusinessEventDetailTarget {
  const BusinessEventDetailTarget({
    required this.account,
    required this.owner,
    required this.rkey,
  });

  final AccountKey account;
  final Did owner;
  final RecordKey rkey;

  @override
  bool operator ==(Object other) =>
      other is BusinessEventDetailTarget &&
      other.account == account &&
      other.owner == owner &&
      other.rkey == rkey;

  @override
  int get hashCode => Object.hash(account, owner, rkey);

  @override
  String toString() => 'BusinessEventDetailTarget(<redacted>)';
}

sealed class BusinessEventDetailState {
  const BusinessEventDetailState();
}

final class BusinessEventDetailAvailable extends BusinessEventDetailState {
  const BusinessEventDetailAvailable(this.event);

  final BusinessEvent event;
}

final class BusinessEventDetailUnavailable extends BusinessEventDetailState {
  const BusinessEventDetailUnavailable();
}

@riverpod
class BusinessEventDetail extends _$BusinessEventDetail {
  @override
  Future<BusinessEventDetailState> build(
    BusinessEventDetailTarget target,
  ) async {
    final ownership = captureActiveAccountOperation(ref);
    if (ownership != null && ownership.session.account != target.account) {
      throw DiagnosticStateError('Active account changed');
    }
    final lease =
        ownership?.session ??
        AccountSessionLease(account: target.account, sessionGeneration: 0);
    final uri =
        'at://${target.owner}/social.craftsky.business.event/${target.rkey}';
    final readFence = captureBusinessEventRead(ref, lease, uri);
    try {
      final event = await ref
          .watch(businessRepositoryProvider)
          .getEvent(target.owner, target.rkey);
      if (!isActiveAccountOperationCurrent(ref, ownership)) {
        throw DiagnosticStateError('Active account changed');
      }
      if (!isBusinessRecordReadCurrent(ref, readFence) &&
          !hasBusinessEventOverlay(ref, lease, uri) &&
          state.value != null) {
        return state.value!;
      }
      final reconciled = applyBusinessEventOverlay(
        ref,
        lease,
        uri,
        event,
      );
      return reconciled == null
          ? const BusinessEventDetailUnavailable()
          : BusinessEventDetailAvailable(reconciled);
    } on ApiBadRequest catch (error) {
      final retained = _retainAfterStaleRead(lease, uri, readFence);
      if (retained != null) return retained;
      if (error.code == 'event_not_found') {
        final reconciled = applyBusinessEventOverlay(
          ref,
          lease,
          uri,
          null,
        );
        return reconciled == null
            ? const BusinessEventDetailUnavailable()
            : BusinessEventDetailAvailable(reconciled);
      }
      rethrow;
    } on Object {
      final retained = _retainAfterStaleRead(lease, uri, readFence);
      if (retained != null) return retained;
      rethrow;
    }
  }

  void retry() => ref.invalidateSelf();

  BusinessEventDetailState? _retainAfterStaleRead(
    AccountSessionLease lease,
    String uri,
    BusinessRecordReadFence fence,
  ) {
    if (isBusinessRecordReadCurrent(ref, fence)) return null;
    if (hasBusinessEventOverlay(ref, lease, uri)) {
      final event = applyBusinessEventOverlay(ref, lease, uri, null);
      return event == null
          ? const BusinessEventDetailUnavailable()
          : BusinessEventDetailAvailable(event);
    }
    if (state.value case final current?) return current;
    return null;
  }

  void markUnavailable() {
    state = const AsyncData(BusinessEventDetailUnavailable());
  }
}
