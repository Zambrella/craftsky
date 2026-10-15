import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/profile/data/profile_api_client.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);
  test('IT-006 decodes recent public summaries '
      'with present and absent optional fields', () async {
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
    DioAdapter(dio: dio).onGet(
      '/v1/profiles/recent',
      (server) => server.reply(200, {
        'items': [
          {
            'did': 'did:plc:aaaaaaaaaaaaaaaaaaaaaaaa',
            'handle': 'alice.test',
            'isCraftskyProfile': true,
            'displayName': 'Alice',
            'avatar': 'https://cdn.example.com/avatar.jpg',
          },
          {
            'did': 'did:plc:bbbbbbbbbbbbbbbbbbbbbbbb',
            'handle': 'handle.invalid',
            'isCraftskyProfile': true,
          },
        ],
      }),
    );
    final rows = await ProfileApiClient(dio).getRecentMembers();
    expect(rows.map((row) => row.handle.value), [
      'alice.test',
      'handle.invalid',
    ]);
    expect(rows.first.displayName, 'Alice');
    expect(rows.first.avatar, 'https://cdn.example.com/avatar.jpg');
    expect(rows.last.displayName, isNull);
    expect(rows.last.avatar, isNull);
    expect(rows.last.crafts, isEmpty);
    expect(
      rows.last.muted || rows.last.blocking || rows.last.blockedBy,
      isFalse,
    );
  });
  test(
    'IT-006 empty list succeeds and API errors retain normal mapping',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
        ..interceptors.add(const ErrorMappingInterceptor());
      final adapter = DioAdapter(dio: dio)
        ..onGet(
          '/v1/profiles/recent',
          (server) => server.reply(200, {'items': <Map<String, dynamic>>[]}),
        );
      expect(await ProfileApiClient(dio).getRecentMembers(), isEmpty);
      adapter.onGet(
        '/v1/profiles/recent',
        (server) => server.reply(500, {
          'error': 'internal_error',
          'message': 'Unable to load recent members',
          'requestId': 'recent-request',
        }),
      );
      await expectLater(
        ProfileApiClient(dio).getRecentMembers(),
        throwsA(isA<ApiException>()),
      );
    },
  );
}
