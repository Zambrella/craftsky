import 'package:craftsky_app/auth/data/auth_api_client.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_runtime_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_service_provider.dart';
import 'package:craftsky_app/notifications/services/notification_sign_out_recovery.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/shared/api/providers/session_auth_interceptor.dart';
import 'package:craftsky_app/shared/device/device_id_provider.dart';
import 'package:dio/dio.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notification_sign_out_recovery_provider.g.dart';

@Riverpod(keepAlive: true)
NotificationSignOutRecovery notificationSignOutRecovery(Ref ref) {
  final registryNotifier = ref.read(sessionRegistryProvider.notifier);
  return NotificationSignOutRecovery(
    readRegistry: () => ref.read(sessionRegistryProvider).requireValue,
    quarantineAndRemove: (lease) async {
      await registryNotifier.quarantineAndRemove(lease);
      // Clear the coordinator's retained eligible set before invalidating the
      // installation-wide provider token.
      await ref.read(notificationRuntimeProvider).resume();
    },
    deleteCleanupCredential: registryNotifier.deletePendingCleanup,
    deleteProviderToken: ref.watch(notificationServiceProvider).deleteToken,
    logoutCleanup: (cleanup) async {
      final deviceId = await ref.read(deviceIdProvider.future);
      final client = Dio(baseDioOptions());
      client.interceptors.addAll([
        SessionAuthInterceptor.fixed(
          token: cleanup.token,
          readDeviceId: () async => deviceId,
        ),
        const ErrorMappingInterceptor(),
      ]);
      try {
        await AuthApiClient(client).logout();
        return NotificationCleanupResult.complete;
      } on ApiUnauthorized {
        return NotificationCleanupResult.alreadyComplete;
      } on ApiException {
        return NotificationCleanupResult.retryable;
      } finally {
        client.close(force: true);
      }
    },
    resumeRegistration: ref.read(notificationRuntimeProvider).resume,
  );
}
