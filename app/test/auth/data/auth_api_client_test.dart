import 'package:craftsky_app/auth/data/auth_api_client.dart';
import 'package:craftsky_app/auth/data/oauth_handoff_mode.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  Dio buildDio() {
    return Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
      ..interceptors.add(const ErrorMappingInterceptor());
  }

  // `http_mock_adapter`'s default `FullHttpRequestMatcher` matches on
  // method + path + data + query, so POST tests must either pass `data:`
  // on the match OR use `UrlRequestMatcher`. The body the client sends
  // for login is selected once from the build-gated OAuth policy.
  final kLoginBody = {
    'handle': 'alice.bsky.social',
    'handoffMode': oauthHandoffModeForCurrentBuild(),
  };

  group('AuthApiClient.login', () {
    test('POSTs /v1/auth/login with the build-gated handoff', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onPost(
        '/v1/auth/login',
        (server) =>
            server.reply(200, {'authUrl': 'https://pds.example.com/auth?x=1'}),
        data: kLoginBody,
      );

      final res = await AuthApiClient(dio).login(handle: 'alice.bsky.social');

      expect(res.authUrl, 'https://pds.example.com/auth?x=1');
    });

    test(
      '400 with handle_required surfaces as ApiBadRequest(handle_required)',
      () async {
        final dio = buildDio();
        DioAdapter(dio: dio).onPost(
          '/v1/auth/login',
          (server) => server.reply(400, {'error': 'handle_required'}),
          data: kLoginBody,
        );

        await expectLater(
          () => AuthApiClient(dio).login(handle: 'alice.bsky.social'),
          throwsA(
            isA<ApiBadRequest>().having(
              (e) => e.code,
              'code',
              'handle_required',
            ),
          ),
        );
      },
    );
  });

  group('AuthApiClient.register', () {
    for (final testCase in <({String mode, String? loopbackUri})>[
      (mode: 'verified_link', loopbackUri: null),
      (mode: 'loopback', loopbackUri: 'http://127.0.0.1:43125/oauth/handoff'),
      (mode: 'dev_scheme', loopbackUri: null),
    ]) {
      test('POSTs the exact ${testCase.mode} handoff request', () async {
        final dio = buildDio();
        final expectedBody = <String, dynamic>{
          'handoffMode': testCase.mode,
          'loopbackRedirectUri': ?testCase.loopbackUri,
        };
        late Map<String, dynamic> capturedBody;
        dio.interceptors.add(
          InterceptorsWrapper(
            onRequest: (options, handler) {
              capturedBody = Map<String, dynamic>.from(options.data as Map);
              handler.next(options);
            },
          ),
        );
        DioAdapter(dio: dio).onPost(
          '/v1/auth/registrations',
          (server) => server.reply(200, {
            'authUrl': 'https://pds.example.com/auth?request_uri=urn:par',
          }),
          data: expectedBody,
        );

        final response = await AuthApiClient(dio).register(
          handoffMode: testCase.mode,
          loopbackRedirectUri: testCase.loopbackUri,
        );

        expect(response.authUrl, contains('request_uri=urn:par'));
        expect(capturedBody, expectedBody);
        for (final forbiddenKey in <String>[
          'handle',
          'did',
          'identity',
          'provider',
          'providerOrigin',
          'email',
          'password',
          'credentials',
          'accessToken',
          'refreshToken',
          'dpopKey',
          'code',
        ]) {
          expect(capturedBody, isNot(contains(forbiddenKey)));
        }
      });
    }
  });

  group('AuthApiClient.whoami', () {
    test('GETs /v1/whoami and parses did + handle', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/whoami',
        (server) => server.reply(200, {
          'did': 'did:plc:alice',
          'handle': 'alice.bsky.social',
        }),
      );

      final res = await AuthApiClient(dio).whoami();

      expect(res.did, 'did:plc:alice');
      expect(res.handle, 'alice.bsky.social');
    });

    test('401 surfaces as ApiUnauthorized', () async {
      final dio = buildDio();
      DioAdapter(
        dio: dio,
      ).onGet('/v1/whoami', (server) => server.reply(401, <String, dynamic>{}));

      await expectLater(
        () => AuthApiClient(dio).whoami(),
        throwsA(isA<ApiUnauthorized>()),
      );
    });
  });

  group('AuthApiClient.logout', () {
    test('POSTs /v1/auth/logout and returns on 204', () async {
      final dio = buildDio();
      DioAdapter(
        dio: dio,
      ).onPost('/v1/auth/logout', (server) => server.reply(204, null));

      await AuthApiClient(dio).logout();
    });
  });
}
