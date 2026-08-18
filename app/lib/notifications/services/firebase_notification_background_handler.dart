import 'package:craftsky_app/notifications/services/flutter_local_notification_gateway.dart';
import 'package:craftsky_app/notifications/services/notification_delivery_envelope.dart';
import 'package:craftsky_app/notifications/services/notification_local_presenter.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';

@pragma('vm:entry-point')
Future<void> firebaseMessagingBackgroundHandler(
  RemoteMessage message, {
  Future<void> Function()? initializeFirebase,
  Future<void> Function(Map<String, Object?> data)? presentData,
}) async {
  if (initializeFirebase != null) {
    await initializeFirebase();
  } else {
    await Firebase.initializeApp();
  }

  if (presentData != null) {
    await presentData(message.data);
    return;
  }
  final presenter = createDefaultNotificationLocalPresenter();
  try {
    await presentBackgroundNotificationData(message.data, presenter);
  } finally {
    await presenter.dispose();
  }
}

Future<void> presentBackgroundNotificationData(
  Map<String, Object?> data,
  NotificationLocalPresenter presenter,
) async {
  final envelope = NotificationDeliveryEnvelope.tryParse(data);
  if (envelope == null) return;
  await presenter.initialize();
  await presenter.present(envelope);
}
