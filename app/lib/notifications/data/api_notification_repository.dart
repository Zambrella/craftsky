import 'package:craftsky_app/notifications/data/notification_api_client.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';

class ApiNotificationRepository implements NotificationRepository {
  const ApiNotificationRepository(this._api);

  final NotificationApiClient _api;

  @override
  Future<NotificationPage> list({String? cursor, int? limit}) =>
      _api.listNotifications(cursor: cursor, limit: limit);
}
