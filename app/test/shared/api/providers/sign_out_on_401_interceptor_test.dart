import 'dart:io';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/shared/api/providers/sign_out_on_401_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../../service_status/announcement_dismissal_test.dart'
    show FakeDismissalStore;
import '../../../service_status/service_status_controller_test.dart'
    show FakeStatusRepository, maintenance;

class _CapturingHandler extends ErrorInterceptorHandler {
  DioException? error;
  @override
  void next(DioException err) => error = err;
}

DioException _exWithStatus(int status) {
  final req = RequestOptions(path: '/v1/whoami');
  return DioException(
    requestOptions: req,
    response: Response(requestOptions: req, statusCode: status),
    type: DioExceptionType.badResponse,
  );
}

DioException _pendingDeletion() {
  final req = RequestOptions(path: '/v1/whoami');
  return DioException(
    requestOptions: req,
    response: Response<Map<String, Object?>>(
      requestOptions: req,
      statusCode: 401,
      data: const {'error': 'account_deletion_pending'},
    ),
    type: DioExceptionType.badResponse,
  );
}

void main() {
  test('401 invalidates only the captured account session lease', () async {
    final leaseA = AccountSessionLease(
      account: AccountKey('did:plc:alice'),
      sessionGeneration: 3,
    );
    final leaseB = AccountSessionLease(
      account: AccountKey('did:plc:bob'),
      sessionGeneration: 7,
    );
    final invalidated = <AccountSessionLease>[];
    final interceptorA = SignOutOn401Interceptor.withLease(
      lease: leaseA,
      invalidate: (lease) async => invalidated.add(lease),
    );
    final interceptorB = SignOutOn401Interceptor.withLease(
      lease: leaseB,
      invalidate: (lease) async => invalidated.add(lease),
    );

    final error401 = _exWithStatus(401);
    final error500 = _exWithStatus(500);
    final handlerA = _CapturingHandler();
    final handlerB = _CapturingHandler();
    interceptorA.onError(error401, handlerA);
    interceptorB.onError(error500, handlerB);
    await Future<void>.delayed(Duration.zero);

    expect(invalidated, [leaseA]);
    expect(handlerA.error, same(error401));
    expect(handlerB.error, same(error500));
  });

  test(
    'any 401, including stale pending-deletion responses, invalidates normally',
    () async {
      final lease = AccountSessionLease(
        account: AccountKey('did:plc:alice'),
        sessionGeneration: 3,
      );
      var invalidated = 0;
      final interceptor = SignOutOn401Interceptor.withLease(
        lease: lease,
        invalidate: (_) async => invalidated++,
      );
      final handler = _CapturingHandler();

      interceptor.onError(_pendingDeletion(), handler);
      await Future<void>.delayed(Duration.zero);

      expect(invalidated, 1);
    },
  );

  test(
    '503 infrastructure failure preserves the captured session lease',
    () async {
      final lease = AccountSessionLease(
        account: AccountKey('did:plc:alice'),
        sessionGeneration: 3,
      );
      final invalidated = <AccountSessionLease>[];
      final interceptor = SignOutOn401Interceptor.withLease(
        lease: lease,
        invalidate: (captured) async => invalidated.add(captured),
      );
      final error503 = _exWithStatus(503);
      final handler = _CapturingHandler();

      interceptor.onError(error503, handler);
      await Future<void>.delayed(Duration.zero);

      expect(invalidated, isEmpty);
      expect(handler.error, same(error503));
    },
  );
  // REG-001 / FR-007, RULE-004 / AC-010, AC-012.
  test(
    'status maintenance/offline/invalid results never invalidate the session; true 401 does',
    () async {
      final lease = AccountSessionLease(
        account: AccountKey('did:plc:alice'),
        sessionGeneration: 3,
      );
      final invalidated = <AccountSessionLease>[];
      final status = FakeStatusRepository();
      final container = ProviderContainer(
        overrides: [
          serviceStatusRepositoryProvider.overrideWithValue(status),
          announcementDismissalStoreProvider.overrideWithValue(
            FakeDismissalStore(),
          ),
          accountSessionInvalidatorProvider.overrideWithValue(
            (lease) async => invalidated.add(lease),
          ),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      final accepted = controller.refresh();
      status.requests.last.complete(maintenance);
      await accepted;
      for (final error in [
        const SocketException('Offline'),
        const FormatException('Invalid'),
        _exWithStatus(503),
      ]) {
        final refreshed = controller.refresh();
        status.requests.last.completeError(error);
        await refreshed;
      }
      expect(invalidated, isEmpty);
      final interceptor = SignOutOn401Interceptor.withLease(
        lease: lease,
        invalidate: container.read(accountSessionInvalidatorProvider),
      );
      // Separate invocation makes the genuine-401 control explicit.
      // ignore: cascade_invocations
      interceptor.onError(_exWithStatus(401), _CapturingHandler());
      await Future<void>.delayed(Duration.zero);
      expect(invalidated, [lease]);
      container.dispose();
    },
  );
}
