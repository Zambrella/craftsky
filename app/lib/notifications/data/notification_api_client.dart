import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:craftsky_app/notifications/models/notification_resolution.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

class NotificationApiClient {
  const NotificationApiClient(this._dio);

  final Dio _dio;

  Future<NotificationPage> listNotifications({String? cursor, int? limit}) =>
      unwrapApi(() async {
        final res = await _dio.get<Map<String, dynamic>>(
          '/v1/notifications',
          queryParameters: {
            'cursor': ?cursor,
            'limit': ?limit?.toString(),
          },
        );
        return NotificationPage.fromMap(res.data!);
      });

  Future<AccountSubscriptionId> registerDevice({
    required NotificationPlatform platform,
    required String token,
  }) => unwrapApi(() async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/v1/notifications/devices',
      data: {'platform': platform.name, 'token': token},
    );
    return AccountSubscriptionId.parse(
      res.data!['accountSubscriptionId'] as String,
    );
  });

  Future<NotificationResolution> resolveNotification(NotificationId id) =>
      unwrapApi(() async {
        final res = await _dio.get<Map<String, dynamic>>(
          '/v1/notifications/${id.wireValue}',
        );
        return NotificationResolution.fromMap(res.data!);
      });

  Future<int> getNewCount() => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/notifications/new-count',
    );
    return res.data!['newCount'] as int;
  });

  Future<void> markSeen() => unwrapApi(() async {
    await _dio.post<void>('/v1/notifications/seen');
  });

  Future<NotificationPreferences> getPreferences() => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/notifications/preferences',
    );
    return NotificationPreferences.fromMap(res.data!);
  });

  Future<NotificationPreferences> patchPreferences(
    NotificationPreferencePatch patch,
  ) => unwrapApi(() async {
    final res = await _dio.patch<Map<String, dynamic>>(
      '/v1/notifications/preferences',
      data: patch.toMap(),
    );
    return NotificationPreferences.fromMap(res.data!);
  });
}
