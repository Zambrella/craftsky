import 'package:craftsky_app/notifications/models/notification_page.dart';

abstract interface class NotificationRepository {
  Future<NotificationPage> list({String? cursor, int? limit});
}
