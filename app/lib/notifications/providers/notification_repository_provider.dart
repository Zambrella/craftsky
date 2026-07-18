import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/notifications/data/api_notification_repository.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notification_repository_provider.g.dart';

@Riverpod(keepAlive: true)
ApiNotificationRepository notificationApiRepository(Ref ref) =>
    ApiNotificationRepository(ref.watch(dioProvider));

@Riverpod(keepAlive: true)
NotificationRepository notificationRepository(Ref ref) =>
    ref.watch(notificationApiRepositoryProvider);

@Riverpod(keepAlive: true)
NotificationNewnessRepository notificationNewnessRepository(Ref ref) =>
    ref.watch(notificationApiRepositoryProvider);

@Riverpod(keepAlive: true)
NotificationPreferencesRepository notificationPreferencesRepository(Ref ref) =>
    ref.watch(notificationApiRepositoryProvider);

@Riverpod(keepAlive: true)
NotificationDeviceRepository notificationDeviceRepository(Ref ref) =>
    ref.watch(notificationApiRepositoryProvider);

@riverpod
Future<NotificationDeviceRepository> accountNotificationDeviceRepository(
  Ref ref,
  AccountKey account,
) => ref.watch(accountNotificationApiRepositoryProvider(account).future);

@riverpod
Future<NotificationRepository> accountNotificationRepository(
  Ref ref,
  AccountKey account,
) => ref.watch(accountNotificationApiRepositoryProvider(account).future);

@riverpod
Future<NotificationNewnessRepository> accountNotificationNewnessRepository(
  Ref ref,
  AccountKey account,
) => ref.watch(accountNotificationApiRepositoryProvider(account).future);

@riverpod
Future<NotificationPreferencesRepository>
accountNotificationPreferencesRepository(
  Ref ref,
  AccountKey account,
) => ref.watch(accountNotificationApiRepositoryProvider(account).future);

@riverpod
Future<ApiNotificationRepository> accountNotificationApiRepository(
  Ref ref,
  AccountKey account,
) async => ApiNotificationRepository(
  await ref.watch(accountDioProvider(account).future),
);
