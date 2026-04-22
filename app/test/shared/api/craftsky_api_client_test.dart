import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/craftsky_api_client.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  Dio buildDio() {
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
    dio.interceptors.add(const ErrorMappingInterceptor());
    return dio;
  }

  // `http_mock_adapter`'s default `FullHttpRequestMatcher` matches on
  // method + path + data + query, so POST tests must either pass `data:`
  // on the match OR use `UrlRequestMatcher`. The body the client sends
  // for login is always `{handle, handoff_mode: 'deep_link'}`.
  const kLoginBody = {
    'handle': 'alice.bsky.social',
    'handoff_mode': 'deep_link',
  };

  group('CraftskyApiClient.login', () {
    test('POSTs /v1/auth/login with handle + deep_link handoff', () async {
      final dio = buildDio();
      final adapter = DioAdapter(dio: dio);
      adapter.onPost(
        '/v1/auth/login',
        (server) =>
            server.reply(200, {'auth_url': 'https://pds.example.com/auth?x=1'}),
        data: kLoginBody,
      );

      final res =
          await CraftskyApiClient(dio).login(handle: 'alice.bsky.social');

      expect(res.authUrl, 'https://pds.example.com/auth?x=1');
    });

    test('400 with handle_required surfaces as ApiBadRequest(handle_required)',
        () async {
      final dio = buildDio();
      final adapter = DioAdapter(dio: dio);
      adapter.onPost(
        '/v1/auth/login',
        (server) => server.reply(400, {'error': 'handle_required'}),
        data: kLoginBody,
      );

      await expectLater(
        () => CraftskyApiClient(dio).login(handle: 'alice.bsky.social'),
        throwsA(isA<ApiBadRequest>()
            .having((e) => e.code, 'code', 'handle_required')),
      );
    });
  });

  group('CraftskyApiClient.whoami', () {
    test('GETs /v1/whoami and parses did + handle', () async {
      final dio = buildDio();
      final adapter = DioAdapter(dio: dio);
      adapter.onGet(
        '/v1/whoami',
        (server) => server.reply(
          200,
          {'did': 'did:plc:alice', 'handle': 'alice.bsky.social'},
        ),
      );

      final res = await CraftskyApiClient(dio).whoami();

      expect(res.did, 'did:plc:alice');
      expect(res.handle, 'alice.bsky.social');
    });

    test('401 surfaces as ApiUnauthorized', () async {
      final dio = buildDio();
      final adapter = DioAdapter(dio: dio);
      adapter.onGet(
        '/v1/whoami',
        (server) => server.reply(401, <String, dynamic>{}),
      );

      await expectLater(
        () => CraftskyApiClient(dio).whoami(),
        throwsA(isA<ApiUnauthorized>()),
      );
    });
  });

  group('CraftskyApiClient.logout', () {
    test('POSTs /v1/auth/logout and returns on 204', () async {
      final dio = buildDio();
      final adapter = DioAdapter(dio: dio);
      adapter.onPost('/v1/auth/logout', (server) => server.reply(204, null));

      await CraftskyApiClient(dio).logout();
    });
  });
}
