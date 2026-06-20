import 'package:craftsky_app/search/data/search_api_client.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  Dio buildDio() {
    return Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
      ..interceptors.add(const ErrorMappingInterceptor());
  }

  test('IT-010 maps unauthorized AppView errors to ApiUnauthorized', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onGet(
      '/v1/search/recent',
      (server) => server.reply(401, {
        'error': 'unauthorized',
        'message': 'missing session',
        'requestId': 'req_auth',
      }),
    );

    await expectLater(
      () => SearchApiClient(dio).listRecentSearches(),
      throwsA(isA<ApiUnauthorized>()),
    );
  });

  test('IT-010 maps server errors without cursor recovery', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onGet(
      '/v1/search/posts',
      (server) => server.reply(500, {
        'error': 'server_error',
        'message': 'boom',
        'requestId': 'req_server',
      }),
      queryParameters: {'q': 'alpaca', 'cursor': 'opaque:bad'},
    );

    await expectLater(
      () => SearchApiClient(dio).searchPosts(q: 'alpaca', cursor: 'opaque:bad'),
      throwsA(isA<ApiServerError>()),
    );
  });
}
