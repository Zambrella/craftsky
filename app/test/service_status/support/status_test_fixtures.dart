import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/models/status_app_version.dart';
import 'package:craftsky_app/service_status/providers/status_runtime_provider.dart';
import 'package:flutter_riverpod/misc.dart';

// Shared schema-2 fixtures for the pre-existing maintenance/message regression
// matrix. Modes here choose fixture data; wire documents have independent
// controls.
ServiceStatusDocument statusDocument({
  required ServiceStatusMode mode,
  required String revision,
  String? title,
  String? message,
  DateTime? estimatedRecoveryAt,
}) => ServiceStatusDocument(
  revision: revision,
  maintenance: mode == ServiceStatusMode.maintenance
      ? StatusMaintenance(
          title: title ?? 'Maintenance',
          description: message ?? 'Back soon',
          estimatedRecoveryAt: estimatedRecoveryAt,
        )
      : null,
  message: mode == ServiceStatusMode.announcement
      ? StatusMessage(
          revision: revision,
          minVersion: '0.0.0',
          title: title ?? 'News',
          description: message ?? 'Public information',
        )
      : null,
);

final List<Override> statusRuntimeOverrides = [
  statusPlatformProvider.overrideWithValue(StatusPlatform.ios),
  statusRuntimeVersionProvider.overrideWith(
    (ref) async => StatusAppVersion.parse('1.8.0+13'),
  ),
];
