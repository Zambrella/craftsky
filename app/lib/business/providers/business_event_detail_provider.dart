import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/providers/business_projection_overlay_provider.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
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
      throw StateError('Active account changed');
    }
    final lease =
        ownership?.session ??
        AccountSessionLease(account: target.account, sessionGeneration: 0);
    final overlay = ref.read(businessProjectionOverlayProvider.notifier);
    final key = BusinessProjectionKey.event(
      target.account,
      target.owner,
      target.rkey,
    );
    final readFence = overlay.captureRead(lease);
    try {
      final event = await ref
          .watch(businessRepositoryProvider)
          .getEvent(target.owner, target.rkey);
      if (!isActiveAccountOperationCurrent(ref, ownership)) {
        throw StateError('Active account changed');
      }
      final reconciliation = overlay.reconcile<BusinessEvent>(
        key: key,
        fence: readFence,
        authoritativeCid: event.cid,
        authoritativeView: event,
      );
      if (reconciliation.isStale && state.value != null) {
        return state.value!;
      }
      final reconciled = reconciliation.view;
      return reconciled == null
          ? const BusinessEventDetailUnavailable()
          : BusinessEventDetailAvailable(reconciled);
    } on ApiBadRequest catch (error) {
      final retained = _retainAfterStaleRead(overlay, key, readFence);
      if (retained != null) return retained;
      if (error.code == 'event_not_found') {
        final reconciliation = overlay.reconcile<BusinessEvent>(
          key: key,
          fence: readFence,
          authoritativeCid: null,
          authoritativeView: null,
        );
        if (reconciliation.isStale && state.value != null) {
          return state.value!;
        }
        final reconciled = reconciliation.view;
        return reconciled == null
            ? const BusinessEventDetailUnavailable()
            : BusinessEventDetailAvailable(reconciled);
      }
      overlay.markReadFailure(
        key: key,
        fence: readFence,
        error: error,
      );
      rethrow;
    } on Object catch (error) {
      final retained = _retainAfterStaleRead(overlay, key, readFence);
      if (retained != null) return retained;
      overlay.markReadFailure(
        key: key,
        fence: readFence,
        error: error,
      );
      rethrow;
    }
  }

  void retry() => ref.invalidateSelf();

  BusinessEventDetailState? _retainAfterStaleRead(
    BusinessProjectionOverlayController overlay,
    BusinessProjectionKey key,
    BusinessProjectionReadFence fence,
  ) {
    if (overlay.isRecordReadCurrent(key, fence)) return null;
    if (state.value case final current?) return current;
    final reconciliation = overlay.reconcile<BusinessEvent>(
      key: key,
      fence: fence,
      authoritativeCid: null,
      authoritativeView: null,
    );
    if (reconciliation.overlay == null) return null;
    final event = reconciliation.view;
    return event == null
        ? const BusinessEventDetailUnavailable()
        : BusinessEventDetailAvailable(event);
  }

  void markUnavailable() {
    state = const AsyncData(BusinessEventDetailUnavailable());
  }
}
