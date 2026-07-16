import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/notification_id.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:craftsky_app/notifications/models/notification_resolution.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';

// Repository interfaces are intentionally one-method seams for AppView-backed
// notification data sources and test fakes.
// Ignored because this boundary intentionally remains a one-method seam.
// ignore: one_member_abstracts
abstract interface class NotificationRepository {
  Future<NotificationPage> list({String? cursor, int? limit});
}

// Ignored because registration is a deliberately narrow adapter boundary.
// ignore: one_member_abstracts
abstract interface class NotificationDeviceRepository {
  Future<AccountSubscriptionId> register({
    required NotificationPlatform platform,
    required String token,
  });
}

// Ignored because resolution is a deliberately narrow adapter boundary.
// ignore: one_member_abstracts
abstract interface class NotificationResolutionRepository {
  Future<NotificationResolution> resolve(NotificationId id);
}

abstract interface class NotificationNewnessRepository {
  Future<int> count();
  Future<void> markSeen();
}

abstract interface class NotificationPreferencesRepository {
  Future<NotificationPreferences> load();
  Future<NotificationPreferences> patch(NotificationPreferencePatch patch);
}
