import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/handoff_api_client.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  Dio buildDio(String token) {
    return Dio(
      BaseOptions(
        baseUrl: 'https://appview.example.com',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $token',
        },
      ),
    )..interceptors.add(const ErrorMappingInterceptor());
  }

  test(
    'whoami sends the baked-in Bearer token and parses the response',
    () async {
      final dio = buildDio('tok-handoff');
      DioAdapter(dio: dio).onGet(
        '/v1/whoami',
        (s) => s.reply(200, {'did': 'did:plc:a', 'handle': 'a.bsky.social'}),
        headers: {'Authorization': 'Bearer tok-handoff'},
      );

      final who = await HandoffApiClient(dio).whoami();

      expect(who.did, 'did:plc:a');
      expect(who.handle, 'a.bsky.social');
    },
  );

  test('401 surfaces as ApiUnauthorized (no side effects)', () async {
    final dio = buildDio('tok-rejected');
    DioAdapter(dio: dio).onGet(
      '/v1/whoami',
      (s) => s.reply(401, <String, dynamic>{}),
    );

    await expectLater(
      () => HandoffApiClient(dio).whoami(),
      throwsA(isA<ApiUnauthorized>()),
    );
  });
}
