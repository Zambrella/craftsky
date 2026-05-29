import 'package:craftsky_app/notifications/data/api_notification_repository.dart';
import 'package:craftsky_app/notifications/data/notification_api_client.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final notificationApiClientProvider = Provider<NotificationApiClient>(
  (ref) => NotificationApiClient(ref.watch(dioProvider)),
);

final notificationRepositoryProvider = Provider<NotificationRepository>(
  (ref) => ApiNotificationRepository(ref.watch(notificationApiClientProvider)),
);
