import 'package:craftsky_app/notifications/services/notification_delivery_dedupe_store.dart';
import 'package:craftsky_app/notifications/services/notification_local_presenter.dart';
import 'package:craftsky_app/notifications/services/notification_presentation_eligibility.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';

final class FlutterLocalNotificationGateway
    implements NotificationPresentationGateway {
  FlutterLocalNotificationGateway({FlutterLocalNotificationsPlugin? plugin})
    : _plugin = plugin ?? FlutterLocalNotificationsPlugin();

  static const String channelId = 'craftsky_notifications';
  static const String channelName = 'CraftSky notifications';

  final FlutterLocalNotificationsPlugin _plugin;

  @override
  Future<void> initialize({
    required void Function(String? payload) onOpen,
  }) async {
    await _plugin.initialize(
      settings: const InitializationSettings(
        android: AndroidInitializationSettings(
          'ic_stat_craftsky_notification',
        ),
        iOS: DarwinInitializationSettings(
          requestAlertPermission: false,
          requestBadgePermission: false,
          requestSoundPermission: false,
          defaultPresentAlert: false,
          defaultPresentBadge: false,
          defaultPresentBanner: false,
          defaultPresentList: false,
          defaultPresentSound: false,
        ),
      ),
      onDidReceiveNotificationResponse: (response) => onOpen(response.payload),
    );
  }

  @override
  Future<void> present(NotificationPresentation presentation) => _plugin.show(
    id: presentation.id,
    title: presentation.title,
    body: presentation.body,
    notificationDetails: NotificationDetails(
      android: AndroidNotificationDetails(
        channelId,
        channelName,
        tag: presentation.tag,
      ),
    ),
    payload: presentation.payload,
  );

  @override
  Future<String?> takeInitialOpenPayload() async {
    final details = await _plugin.getNotificationAppLaunchDetails();
    if (details?.didNotificationLaunchApp != true) return null;
    return details?.notificationResponse?.payload;
  }
}

NotificationLocalPresenter createDefaultNotificationLocalPresenter() =>
    NotificationLocalPresenter(
      gateway: FlutterLocalNotificationGateway(),
      dedupe: SqliteNotificationDeliveryDedupeStore(),
      eligibility: createDefaultNotificationPresentationEligibility(),
    );
