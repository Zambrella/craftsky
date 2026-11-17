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

abstract interface class SubscriptionManagementApi {
  Future<Uri> managementURL();
}

abstract interface class PendingCheckoutCancellationApi {
  Future<void> cancelPendingCheckout();
}

final class SubscriptionApiClient
    implements
        SubscriptionApi,
        PendingCheckoutCancellationApi,
        SubscriptionManagementApi {
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
  Future<void> cancelPendingCheckout() => unwrapApi(() async {
    await _dio.post<Map<String, dynamic>>('/v1/billing/web-checkout/cancel');
  });

  @override
  Future<Uri> managementURL() => unwrapApi(() async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/v1/billing/web-management',
    );
    final raw = response.data?['url'];
    if (raw is! String ||
        raw.contains(RegExp(r'[\x00-\x20\x7f]')) ||
        RegExp(
              r'^https://billing\.stripe\.com/p/session(?:/[A-Za-z0-9_-]+(?:#[^\s]+)?|\?secret=[A-Za-z0-9._~-]+)$',
            ).stringMatch(raw) !=
            raw) {
      throw const FormatException('Subscription management unavailable');
    }
    return Uri.parse(raw);
  });

  @override
  Future<void> requestReconciliation() => unwrapApi(() async {
    await _dio.post<Map<String, dynamic>>('/v1/billing/reconciliation');
  });
}
