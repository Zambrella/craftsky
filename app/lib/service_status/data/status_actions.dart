import 'package:craftsky_app/service_status/data/status_reload_stub.dart'
    if (dart.library.js_interop) 'status_reload_web.dart';
import 'package:craftsky_app/service_status/providers/status_runtime_provider.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final statusReloadProvider = Provider<void Function()>(
  (ref) => reloadCurrentStatusApp,
);

final statusExternalLauncherProvider = Provider<ExternalLinkLauncher>(
  (ref) => launchExternalLink,
);

Uri officialStatusStore(StatusPlatform platform) => switch (platform) {
  StatusPlatform.ios => Uri.parse('https://apps.apple.com/app/id6812428632'),
  StatusPlatform.android => Uri.parse(
    'https://play.google.com/store/apps/details?id=social.craftsky.app',
  ),
  StatusPlatform.web => throw const FormatException('No web store action'),
};
