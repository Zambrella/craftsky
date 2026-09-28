import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:dio/dio.dart';

abstract interface class SubscriptionApi {
  Future<SubscriptionAccess> getAccess();
  Future<BillingState> ensureBillingAccount();
  Future<BillingState> getBillingAccount();
  Future<void> requestReconciliation();
  Future<BillingAssignment> assign(String licenseId, String targetDid);
  Future<void> unassign(String licenseId);
}

final class SubscriptionApiClient implements SubscriptionApi {
  const SubscriptionApiClient(this._dio);

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

  @override
  Future<BillingAssignment> assign(String licenseId, String targetDid) =>
      unwrapApi(() async {
        final response = await _dio.put<Map<String, dynamic>>(
          _assignmentPath(licenseId),
          data: {'targetDid': targetDid},
        );
        return BillingAssignment.fromMap(response.data!);
      });

  @override
  Future<void> unassign(String licenseId) => unwrapApi(() async {
    await _dio.delete<void>(_assignmentPath(licenseId));
  });

  String _assignmentPath(String licenseId) =>
      '/v1/billing/licenses/${Uri.encodeComponent(licenseId)}/assignment';
}
