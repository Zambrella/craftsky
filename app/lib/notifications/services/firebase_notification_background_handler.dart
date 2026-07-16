import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';

@pragma('vm:entry-point')
Future<void> firebaseMessagingBackgroundHandler(
  RemoteMessage message, {
  Future<void> Function()? initializeFirebase,
}) async {
  if (initializeFirebase != null) {
    await initializeFirebase();
  } else {
    await Firebase.initializeApp();
  }
}
