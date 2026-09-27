import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final lease = AccountSessionLease(
    account: AccountKey('did:plc:operation-controller'),
    sessionGeneration: 7,
  );
  final scope = PdsMutationScope(lease: lease, identity: 'like:post-1');
  final start = DateTime.utc(2026, 9, 23, 12);

  test('ambiguous operation freezes immutable input and same-key retry', () {
    final controller = PdsRecordOperationController(now: () => start);
    final token = controller.begin(
      scope: scope,
      operationKey: '00000000-0000-4000-8000-000000000001',
      endpoint: '/v1/posts/post-1/like',
      immutableBody: '{"active":true}',
    );

    expect(controller.markAmbiguous(token, retryAfterSeconds: 3), isTrue);
    expect(
      controller.canRetry(
        token,
        operationKey: token.operationKey,
        endpoint: token.endpoint,
        immutableBody: token.immutableBody,
      ),
      isTrue,
    );
    expect(
      controller.canRetry(
        token,
        operationKey: token.operationKey,
        endpoint: token.endpoint,
        immutableBody: '{"active":false}',
      ),
      isFalse,
    );
    expect(
      () => controller.begin(
        scope: scope,
        operationKey: '00000000-0000-4000-8000-000000000002',
        endpoint: token.endpoint,
        immutableBody: '{"active":false}',
      ),
      throwsStateError,
    );

    final retry = controller.retryToken(
      scope: scope,
      endpoint: token.endpoint,
      immutableBody: token.immutableBody,
    );
    expect(retry, same(token));
    expect(retry?.operationKey, token.operationKey);
    expect(
      controller.retryToken(
        scope: scope,
        endpoint: token.endpoint,
        immutableBody: '{"active":false}',
      ),
      isNull,
    );
  });

  test('newest sequence and account reset discard late results', () {
    final controller = PdsRecordOperationController(now: () => start);
    final older = controller.begin(
      scope: scope,
      operationKey: 'older',
      endpoint: '/mutation',
      immutableBody: '{}',
    );
    final newer = controller.begin(
      scope: scope,
      operationKey: 'newer',
      endpoint: '/mutation',
      immutableBody: '{}',
    );

    expect(controller.markFailed(older), isFalse);
    expect(controller.markAmbiguous(newer, retryAfterSeconds: 1), isTrue);
    controller.reset();
    expect(controller.markFailed(newer), isFalse);
    expect(controller.operationFor(scope), isNull);
  });

  test('accepted operation can complete without a reconciliation overlay', () {
    final controller = PdsRecordOperationController();
    final token = controller.begin(
      scope: scope,
      operationKey: 'operation-key',
      endpoint: '/v1/posts',
      immutableBody: '{"reply":true}',
    );

    expect(controller.markAcceptedWithoutOverlay(token), isTrue);
    expect(controller.operationFor(scope), isNull);
    expect(controller.overlayFor(scope), isNull);
  });

  test('accepted overlay reconciles, refreshes once, and expires once', () {
    var now = start;
    final refreshed = <PdsMutationScope>[];
    final controller = PdsRecordOperationController(
      now: () => now,
      refreshScope: refreshed.add,
    );
    final token = controller.begin(
      scope: scope,
      operationKey: 'accepted',
      endpoint: '/mutation',
      immutableBody: '{}',
    );
    expect(
      controller.markAccepted(
        token,
        optimisticValue: true,
        agrees: (value) => value == true,
      ),
      isTrue,
    );
    expect(refreshed, [scope]);

    expect(controller.reconcile(scope, false), isFalse);
    expect(controller.overlayFor(scope)?.optimisticValue, isTrue);
    now = start.add(const Duration(seconds: 2));
    controller
      ..advanceTime()
      ..advanceTime();
    expect(refreshed, [scope, scope]);
    expect(controller.overlayFor(scope), isNotNull);

    now = start.add(const Duration(seconds: 30));
    controller
      ..advanceTime()
      ..advanceTime();
    expect(refreshed, [scope, scope, scope]);
    expect(controller.overlayFor(scope), isNull);
  });

  test('acceptance schedules grace and expiry checks', () {
    var now = start;
    final scheduled = <({Duration delay, void Function() callback})>[];
    final refreshed = <PdsMutationScope>[];
    final controller = PdsRecordOperationController(
      now: () => now,
      refreshScope: refreshed.add,
      schedule: (delay, callback) => scheduled.add((
        delay: delay,
        callback: callback,
      )),
    );
    final token = controller.begin(
      scope: scope,
      operationKey: 'scheduled',
      endpoint: '/mutation',
      immutableBody: '{}',
    );

    controller.markAccepted(
      token,
      optimisticValue: true,
      agrees: (value) => value == true,
    );

    expect(refreshed, [scope]);
    expect(scheduled.map((entry) => entry.delay), [
      const Duration(seconds: 2),
      const Duration(seconds: 30),
    ]);
    now = start.add(const Duration(seconds: 2));
    scheduled.first.callback();
    expect(refreshed, [scope, scope]);
    now = start.add(const Duration(seconds: 30));
    scheduled.last.callback();
    expect(refreshed, [scope, scope, scope]);
    expect(controller.overlayFor(scope), isNull);
  });

  test('authoritative agreement retires only the current overlay', () {
    final controller = PdsRecordOperationController(now: () => start);
    final token = controller.begin(
      scope: scope,
      operationKey: 'accepted',
      endpoint: '/mutation',
      immutableBody: '{}',
    );
    controller.markAccepted(
      token,
      optimisticValue: 'cid-accepted',
      agrees: (value) => value == 'cid-accepted',
    );

    expect(controller.reconcile(scope, 'cid-older'), isFalse);
    expect(controller.reconcile(scope, 'cid-accepted'), isTrue);
    expect(controller.overlayFor(scope), isNull);
  });
}
