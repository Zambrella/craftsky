import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  test('IT-001 sends and decodes every subscription API contract', () async {
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
      ..interceptors.add(const ErrorMappingInterceptor());
    final adapter = DioAdapter(dio: dio);
    final client = SubscriptionApiClient(dio);
    const licenseId = '40000000-0000-4000-8000-000000000001';
    final owner = _ownerFixture();

    adapter
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
        (server) => server.reply(201, owner),
      )
      ..onGet(
        '/v1/billing/account',
        (server) => server.reply(200, owner),
      )
      ..onPost(
        '/v1/billing/reconciliation',
        (server) => server.reply(202, null),
      )
      ..onPut(
        '/v1/billing/licenses/$licenseId/assignment',
        (server) => server.reply(200, {
          'licenseId': licenseId,
          'targetDid': 'did:plc:bob',
          'assignedAt': '2026-09-10T12:00:00Z',
        }),
        data: {'targetDid': 'did:plc:bob'},
      )
      ..onDelete(
        '/v1/billing/licenses/$licenseId/assignment',
        (server) => server.reply(204, null),
      );

    expect(
      (await client.getAccess()).effectiveTier,
      SubscriptionTier.plus,
    );
    expect((await client.ensureBillingAccount()).billingAccountId, isNotEmpty);
    expect((await client.getBillingAccount()).licenses, isEmpty);
    await client.requestReconciliation();
    expect(
      (await client.assign(licenseId, 'did:plc:bob')).targetDid.value,
      'did:plc:bob',
    );
    await client.unassign(licenseId);
  });

  for (final code in const [
    'billing_license_not_found',
    'assignment_target_ineligible',
    'assignment_conflict',
    'assignment_cooldown',
  ]) {
    test('IT-001 preserves exact AppView assignment error $code', () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
        ..interceptors.add(const ErrorMappingInterceptor());
      final adapter = DioAdapter(dio: dio);
      final client = SubscriptionApiClient(dio);
      const licenseId = '40000000-0000-4000-8000-000000000001';

      adapter.onPut(
        '/v1/billing/licenses/$licenseId/assignment',
        (server) => server.reply(422, {
          'error': code,
          'message': 'Safe assignment failure.',
          'requestId': 'request-canary',
        }),
        data: {'targetDid': 'did:plc:bob'},
      );

      await expectLater(
        client.assign(licenseId, 'did:plc:bob'),
        throwsA(
          isA<ApiBadRequest>()
              .having((error) => error.code, 'code', code)
              .having(
                (error) => error.details.requestId,
                'requestId',
                'request-canary',
              ),
        ),
      );
    });
  }

  test(
    'IT-001 preserves DELETE missing-license envelope and request ID',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
        ..interceptors.add(const ErrorMappingInterceptor());
      final adapter = DioAdapter(dio: dio);
      final client = SubscriptionApiClient(dio);
      const licenseId = '40000000-0000-4000-8000-000000000001';

      adapter.onDelete(
        '/v1/billing/licenses/$licenseId/assignment',
        (server) => server.reply(422, {
          'error': 'billing_license_not_found',
          'message': 'License unavailable.',
          'requestId': 'request-delete-missing',
        }),
      );

      await expectLater(
        client.unassign(licenseId),
        throwsA(
          isA<ApiBadRequest>()
              .having(
                (error) => error.code,
                'code',
                'billing_license_not_found',
              )
              .having(
                (error) => error.details.requestId,
                'requestId',
                'request-delete-missing',
              ),
        ),
      );
    },
  );

  for (final operation in const ['assign', 'unassign']) {
    test(
      'IT-001 $operation mutation preserves unauthorized envelope',
      () async {
        final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
          ..interceptors.add(const ErrorMappingInterceptor());
        final adapter = DioAdapter(dio: dio);
        final client = SubscriptionApiClient(dio);
        const licenseId = '40000000-0000-4000-8000-000000000001';
        const path = '/v1/billing/licenses/$licenseId/assignment';
        if (operation == 'assign') {
          adapter.onPut(
            path,
            (server) => server.reply(401, {
              'error': 'unauthorized',
              'message': 'Sign in again.',
              'requestId': 'request-$operation-401',
            }),
            data: {'targetDid': 'did:plc:bob'},
          );
        } else {
          adapter.onDelete(
            path,
            (server) => server.reply(401, {
              'error': 'unauthorized',
              'message': 'Sign in again.',
              'requestId': 'request-$operation-401',
            }),
          );
        }

        await expectLater(
          operation == 'assign'
              ? client.assign(licenseId, 'did:plc:bob')
              : client.unassign(licenseId),
          throwsA(
            isA<ApiUnauthorized>().having(
              (error) => error.details.requestId,
              'requestId',
              'request-$operation-401',
            ),
          ),
        );
      },
    );

    test('IT-001 $operation mutation redacts 5xx envelope', () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
        ..interceptors.add(const ErrorMappingInterceptor());
      final adapter = DioAdapter(dio: dio);
      final client = SubscriptionApiClient(dio);
      const licenseId = '40000000-0000-4000-8000-000000000001';
      const path = '/v1/billing/licenses/$licenseId/assignment';
      if (operation == 'assign') {
        adapter.onPut(
          path,
          (server) => server.reply(500, {
            'error': 'internal_error',
            'message': 'sensitive mutation details',
            'requestId': 'request-$operation-500',
          }),
          data: {'targetDid': 'did:plc:bob'},
        );
      } else {
        adapter.onDelete(
          path,
          (server) => server.reply(500, {
            'error': 'internal_error',
            'message': 'sensitive mutation details',
            'requestId': 'request-$operation-500',
          }),
        );
      }

      await expectLater(
        operation == 'assign'
            ? client.assign(licenseId, 'did:plc:bob')
            : client.unassign(licenseId),
        throwsA(
          isA<ApiServerError>().having(
            (error) => error.details.requestId,
            'requestId',
            'request-$operation-500',
          ),
        ),
      );
    });
  }

  test('IT-001 owner 401 uses standard unauthorized mapping', () async {
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
      ..interceptors.add(const ErrorMappingInterceptor());
    DioAdapter(dio: dio).onGet(
      '/v1/billing/account',
      (server) => server.reply(401, {
        'error': 'unauthorized',
        'message': 'Sign in again.',
        'requestId': 'request-401',
      }),
    );

    await expectLater(
      SubscriptionApiClient(dio).getBillingAccount(),
      throwsA(
        isA<ApiUnauthorized>().having(
          (error) => error.details.requestId,
          'requestId',
          'request-401',
        ),
      ),
    );
  });

  test('IT-001 owner 5xx uses redacted server-error mapping', () async {
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
  });
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
