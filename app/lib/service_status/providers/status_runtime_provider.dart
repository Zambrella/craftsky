import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/models/status_app_version.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:package_info_plus/package_info_plus.dart';

enum StatusPlatform { ios, android, web }

final statusPlatformProvider = Provider<StatusPlatform?>((ref) {
  if (kIsWeb) return StatusPlatform.web;
  return switch (defaultTargetPlatform) {
    TargetPlatform.iOS => StatusPlatform.ios,
    TargetPlatform.android => StatusPlatform.android,
    _ => null,
  };
});

// Independent of account, preferences and the general app dependency graph.
// Async failure is observed as unknown by status consumers, never a lockout.
final statusRuntimeVersionProvider = FutureProvider<StatusAppVersion?>((
  ref,
) async {
  final info = await PackageInfo.fromPlatform();
  try {
    return StatusAppVersion.parse(info.version);
  } on FormatException {
    return null;
  }
});

bool requiresStatusUpdate(
  StatusMinimumVersions? minima,
  StatusPlatform? platform,
  StatusAppVersion? current,
) {
  if (minima == null || current == null || platform == null) return false;
  final minimum = switch (platform) {
    StatusPlatform.ios => minima.ios,
    StatusPlatform.android => minima.android,
    StatusPlatform.web => minima.web,
  };
  return minimum != null &&
      current.compareTo(StatusAppVersion.parse(minimum)) < 0;
}

bool statusMessageEligible(StatusMessage? message, StatusAppVersion? current) =>
    message != null &&
    current != null &&
    current.compareTo(StatusAppVersion.parse(message.minVersion)) >= 0 &&
    (message.maxVersion == null ||
        current.compareTo(StatusAppVersion.parse(message.maxVersion!)) <= 0);
