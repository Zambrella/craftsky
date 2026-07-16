import 'package:craftsky_app/notifications/services/firebase_notification_background_handler.dart';
import 'package:craftsky_app/notifications/services/firebase_notification_service.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';

Future<NotificationService> bootstrapFirebaseNotificationService() async {
  try {
    await Firebase.initializeApp();
    FirebaseMessaging.onBackgroundMessage(firebaseMessagingBackgroundHandler);
    return FirebaseNotificationService(FirebaseMessaging.instance);
  } on Object {
    return const UnavailableNotificationService();
  }
}
