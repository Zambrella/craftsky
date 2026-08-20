import 'package:flutter/foundation.dart';

const _devSchemeRequested = bool.fromEnvironment(
  'CRAFTSKY_DEV_OAUTH_SCHEME',
);

String selectOAuthHandoffMode({
  required bool isDebug,
  required bool devSchemeRequested,
}) => isDebug && devSchemeRequested ? 'dev_scheme' : 'verified_link';

String oauthHandoffModeForCurrentBuild() => selectOAuthHandoffMode(
  isDebug: kDebugMode,
  devSchemeRequested: _devSchemeRequested,
);
