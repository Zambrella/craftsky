import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/shared/api/providers/sign_out_on_401_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

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
}
