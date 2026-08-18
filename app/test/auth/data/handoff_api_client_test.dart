import 'package:craftsky_app/auth/data/handoff_api_client.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  Dio buildDio() {
    return Dio(
      BaseOptions(
        baseUrl: 'https://appview.example.com',
        headers: {'Content-Type': 'application/json'},
      ),
    )..interceptors.add(const ErrorMappingInterceptor());
  }

  test(
    'exchange POSTs only the code and parses the pending receipt',
    () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onPost(
        '/v1/auth/handoffs/exchange',
        (s) => s.reply(200, {
          'token': 'pending-bearer',
          'did': 'did:plc:a',
          'handle': 'a.bsky.social',
          'receiptId': '00000000-0000-4000-8000-000000000811',
          'confirmBy': '2026-08-14T12:05:00Z',
        }),
        data: {'code': 'browser-handoff-code'},
      );

      final result = await HandoffApiClient(
        dio,
      ).exchange(code: 'browser-handoff-code');

      expect(result.token, 'pending-bearer');
      expect(result.did, 'did:plc:a');
      expect(result.handle, 'a.bsky.social');
      expect(result.receiptId, '00000000-0000-4000-8000-000000000811');
      expect(result.confirmBy, DateTime.utc(2026, 8, 14, 12, 5));
      expect(result.toString(), isNot(contains('pending-bearer')));
      expect(result.toString(), isNot(contains('browser-handoff-code')));
    },
  );

  test('invalid exchange surfaces the safe AppView error code', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onPost(
      '/v1/auth/handoffs/exchange',
      (s) => s.reply(400, {'error': 'invalid_handoff'}),
      data: {'code': 'expired-code'},
    );

    await expectLater(
      () => HandoffApiClient(dio).exchange(code: 'expired-code'),
      throwsA(
        isA<ApiBadRequest>().having(
          (error) => error.code,
          'code',
          'invalid_handoff',
        ),
      ),
    );
  });

  test(
    'confirm sends the pending bearer only in the Authorization header',
    () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onPost(
        '/v1/auth/handoffs/confirm',
        (server) => server.reply(204, null),
        data: {'receiptId': '00000000-0000-4000-8000-000000000811'},
        headers: {'Authorization': 'Bearer pending-bearer'},
      );

      await HandoffApiClient(dio).confirm(
        token: 'pending-bearer',
        receiptId: '00000000-0000-4000-8000-000000000811',
      );
    },
  );
}
