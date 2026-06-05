import 'package:craftsky_app/notifications/models/notification_page.dart';

// Repository interfaces are intentionally one-method seams for AppView-backed
// notification data sources and test fakes.
// ignore: one_member_abstracts
abstract interface class NotificationRepository {
  Future<NotificationPage> list({String? cursor, int? limit});
}
