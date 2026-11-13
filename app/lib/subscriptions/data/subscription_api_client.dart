import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:dio/dio.dart';

abstract interface class SubscriptionApi {
  Future<SubscriptionAccess> getAccess();
  Future<BillingState> ensureBillingAccount();
  Future<BillingState> getBillingAccount();
  Future<void> requestReconciliation();
}

final class SubscriptionApiClient implements SubscriptionApi {
  const new(this._dio);

  final Dio _dio;

  @override
  Future<SubscriptionAccess> getAccess() => unwrapApi(() async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/v1/subscriptions/access',
    );
    return SubscriptionAccess.fromMap(response.data!);
  });

  @override
  Future<BillingState> ensureBillingAccount() => unwrapApi(() async {
    final response = await _dio.put<Map<String, dynamic>>(
      '/v1/billing/account',
    );
    return BillingState.fromMap(response.data!);
  });

  @override
  Future<BillingState> getBillingAccount() => unwrapApi(() async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/v1/billing/account',
    );
    return BillingState.fromMap(response.data!);
  });

  @override
  Future<void> requestReconciliation() => unwrapApi(() async {
    await _dio.post<Map<String, dynamic>>('/v1/billing/reconciliation');
  });
}
