import 'package:craftsky_app/notifications/models/notification_page.dart';
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
}
