import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/notification_id.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:craftsky_app/notifications/models/notification_resolution.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

class ApiNotificationRepository
    implements
        NotificationRepository,
        NotificationDeviceRepository,
        NotificationResolutionRepository,
        NotificationNewnessRepository,
        NotificationPreferencesRepository {
  const ApiNotificationRepository(this._dio);

  final Dio _dio;

  @override
  Future<NotificationPage> list({String? cursor, int? limit}) => unwrapApi(
    () async {
      final response = await _dio.get<Map<String, dynamic>>(
        '/v1/notifications',
        queryParameters: {'cursor': ?cursor, 'limit': ?limit?.toString()},
      );
      return NotificationPage.fromMap(response.data!);
    },
  );

  @override
  Future<AccountSubscriptionId> register({
    required NotificationPlatform platform,
    required String token,
  }) => unwrapApi(() async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/v1/notifications/devices',
      data: {'platform': platform.name, 'token': token},
    );
    return AccountSubscriptionId.parse(
      response.data!['accountSubscriptionId'] as String,
    );
  });

  @override
  Future<NotificationResolution> resolve(NotificationId id) => unwrapApi(
    () async {
      final response = await _dio.get<Map<String, dynamic>>(
        '/v1/notifications/${id.wireValue}',
      );
      return NotificationResolution.fromMap(response.data!);
    },
  );

  @override
  Future<int> count() => unwrapApi(() async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/v1/notifications/new-count',
    );
    return response.data!['newCount'] as int;
  });

  @override
  Future<void> markSeen() => unwrapApi(() async {
    await _dio.post<void>('/v1/notifications/seen');
  });

  @override
  Future<NotificationPreferences> load() => unwrapApi(() async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/v1/notifications/preferences',
    );
    return NotificationPreferences.fromMap(response.data!);
  });

  @override
  Future<NotificationPreferences> patch(NotificationPreferencePatch patch) =>
      unwrapApi(() async {
        final response = await _dio.patch<Map<String, dynamic>>(
          '/v1/notifications/preferences',
          data: patch.toMap(),
        );
        return NotificationPreferences.fromMap(response.data!);
      });
}
