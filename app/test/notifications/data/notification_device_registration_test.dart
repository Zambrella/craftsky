import 'package:craftsky_app/notifications/data/notification_api_client.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  test(
    'IT-001 registers a native device with the exact camelCase body',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
      DioAdapter(dio: dio).onPost(
        '/v1/notifications/devices',
        (server) => server.reply(200, {
          'accountSubscriptionId': 'registered_binding',
        }),
        data: {'platform': 'android', 'token': 'SENSITIVE_FAKE_TOKEN'},
      );

      final binding = await NotificationApiClient(dio).registerDevice(
        platform: NotificationPlatform.android,
        token: 'SENSITIVE_FAKE_TOKEN',
      );

      expect(binding, AccountSubscriptionId.parse('registered_binding'));
      expect(binding.toString(), isNot(contains('registered_binding')));
    },
  );
}
