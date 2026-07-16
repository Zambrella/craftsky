import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/notifications/data/notification_api_client.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  Dio buildDio() => Dio(BaseOptions(baseUrl: 'https://appview.example.com'));

  test('requests notifications endpoint and passes cursor opaquely', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onGet(
      '/v1/notifications',
      (server) => server.reply(200, {'items': <Map<String, dynamic>>[]}),
      queryParameters: {'limit': '20', 'cursor': 'opaque:abc'},
    );

    final page = await NotificationApiClient(
      dio,
    ).listNotifications(limit: 20, cursor: 'opaque:abc');

    expect(page.items, isEmpty);
    expect(page.cursor, isNull);
  });

  test('IT-005 posts seen without a request body', () async {
    final dio = buildDio();
    RequestOptions? captured;
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          captured = options;
          handler.next(options);
        },
      ),
    );
    DioAdapter(dio: dio).onPost(
      '/v1/notifications/seen',
      (server) => server.reply(204, null),
    );

    await NotificationApiClient(dio).markSeen();

    expect(captured?.method, 'POST');
    expect(captured?.path, '/v1/notifications/seen');
    expect(captured?.data, isNull);
  });
}
