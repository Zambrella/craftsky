import 'package:craftsky_app/notifications/data/notification_api_client.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:craftsky_app/notifications/models/notification_resolution.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';

class ApiNotificationRepository
    implements
        NotificationRepository,
        NotificationDeviceRepository,
        NotificationResolutionRepository,
        NotificationNewnessRepository,
        NotificationPreferencesRepository {
  const ApiNotificationRepository(this._api);

  final NotificationApiClient _api;

  @override
  Future<NotificationPage> list({String? cursor, int? limit}) =>
      _api.listNotifications(cursor: cursor, limit: limit);

  @override
  Future<AccountSubscriptionId> register({
    required NotificationPlatform platform,
    required String token,
  }) => _api.registerDevice(platform: platform, token: token);

  @override
  Future<NotificationResolution> resolve(NotificationId id) =>
      _api.resolveNotification(id);

  @override
  Future<int> count() => _api.getNewCount();

  @override
  Future<void> markSeen() => _api.markSeen();

  @override
  Future<NotificationPreferences> load() => _api.getPreferences();

  @override
  Future<NotificationPreferences> patch(NotificationPreferencePatch patch) =>
      _api.patchPreferences(patch);
}
