import 'package:craftsky_app/notifications/data/notification_api_client.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  Dio buildDio() => Dio(BaseOptions(baseUrl: 'https://appview.example.com'));

  test('IT-006 GET preserves future preference categories', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onGet(
      '/v1/notifications/preferences',
      (server) => server.reply(200, {
        'preferences': {
          'like': {'scope': 'everyone', 'pushEnabled': true},
          'futureCategory': {
            'scope': 'futureScope',
            'pushEnabled': false,
            'futureField': 'retained',
          },
        },
      }),
    );

    final preferences = await NotificationApiClient(dio).getPreferences();

    expect(
      preferences.known[NotificationCategory.like]?.scope,
      NotificationPreferenceScope.everyone,
    );
    expect(preferences.unknown, {
      'futureCategory': {
        'scope': 'futureScope',
        'pushEnabled': false,
        'futureField': 'retained',
      },
    });
  });

  test('IT-006 PATCH changes exactly one known category and field', () async {
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
    DioAdapter(dio: dio).onPatch(
      '/v1/notifications/preferences',
      (server) => server.reply(200, {
        'preferences': {
          'quote': {'scope': 'peopleIFollow', 'pushEnabled': true},
          'futureCategory': {'scope': 'futureScope', 'pushEnabled': false},
        },
      }),
      data: {
        'preferences': {
          'quote': {'scope': 'peopleIFollow'},
        },
      },
    );

    final preferences = await NotificationApiClient(dio).patchPreferences(
      NotificationPreferencePatch.scope(
        NotificationCategory.quote,
        value: NotificationPreferenceScope.peopleIFollow,
      ),
    );

    expect(captured?.method, 'PATCH');
    expect(captured?.path, '/v1/notifications/preferences');
    expect(captured?.data, {
      'preferences': {
        'quote': {'scope': 'peopleIFollow'},
      },
    });
    expect(
      preferences.known[NotificationCategory.quote]?.scope,
      NotificationPreferenceScope.peopleIFollow,
    );
    expect(preferences.unknown, contains('futureCategory'));
  });
}
