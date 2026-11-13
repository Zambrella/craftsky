import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);
  test('malformed pending checkout state fails closed', () {
    for (final value in [null, 'true', 1]) {
      expect(
        () => BillingState.fromMap({
          ..._ownerFixture(),
          'webCheckoutPending': value,
        }),
        throwsFormatException,
      );
    }
  });
  test(
    'reads active account access and idempotently creates its customer',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
        ..interceptors.add(const ErrorMappingInterceptor());
      DioAdapter(dio: dio)
        ..onGet(
          '/v1/subscriptions/access',
          (server) => server.reply(200, {
            'did': 'did:plc:alice',
            'effectiveTier': 'plus',
            'givesAccess': true,
            'assignedTier': 'plus',
          }),
        )
        ..onPut(
          '/v1/billing/account',
          (server) => server.reply(200, _ownerFixture()),
        )
        ..onGet(
          '/v1/billing/account',
          (server) => server.reply(200, _ownerFixture()),
        )
        ..onPost(
          '/v1/billing/reconciliation',
          (server) => server.reply(202, null),
        );
      final api = SubscriptionApiClient(dio);
      expect((await api.getAccess()).effectiveTier, SubscriptionTier.plus);
      expect(
        (await api.ensureBillingAccount()).revenueCatAppUserId,
        '20000000-0000-4000-8000-000000000001',
      );
      final account = await api.getBillingAccount();
      expect(account.licenses, isEmpty);
      expect(account.webCheckoutPending, isFalse);
      await api.requestReconciliation();
    },
  );
  test(
    'retains a pending web checkout before a subscription is indexed',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
      DioAdapter(dio: dio).onGet(
        '/v1/billing/account',
        (server) => server.reply(200, {
          ..._ownerFixture(),
          'webCheckoutPending': true,
        }),
      );
      final account = await SubscriptionApiClient(dio).getBillingAccount();
      expect(account.webCheckoutPending, isTrue);
      expect(account.subscriptions, isEmpty);
    },
  );
  test(
    'billing errors retain request reference without exposing server details',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
        ..interceptors.add(const ErrorMappingInterceptor());
      DioAdapter(dio: dio).onGet(
        '/v1/billing/account',
        (server) => server.reply(500, {
          'error': 'internal_error',
          'message': 'sensitive database details',
          'requestId': 'request-500',
        }),
      );
      await expectLater(
        SubscriptionApiClient(dio).getBillingAccount(),
        throwsA(
          isA<ApiServerError>().having(
            (error) => error.details.requestId,
            'requestId',
            'request-500',
          ),
        ),
      );
    },
  );
}

Map<String, dynamic> _ownerFixture() => {
  'billingAccountId': '10000000-0000-4000-8000-000000000001',
  'revenueCatAppUserId': '20000000-0000-4000-8000-000000000001',
  'requestedGeneration': 0,
  'reconciledGeneration': 0,
  'reconciliationStale': false,
  'subscriptions': <Object>[],
  'licenses': <Object>[],
};
