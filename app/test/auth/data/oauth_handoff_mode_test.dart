import 'package:craftsky_app/auth/data/oauth_handoff_mode.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('dev OAuth scheme requires both debug build and explicit opt-in', () {
    expect(
      selectOAuthHandoffMode(isDebug: false, devSchemeRequested: false),
      'verified_link',
    );
    expect(
      selectOAuthHandoffMode(isDebug: false, devSchemeRequested: true),
      'verified_link',
    );
    expect(
      selectOAuthHandoffMode(isDebug: true, devSchemeRequested: false),
      'verified_link',
    );
    expect(
      selectOAuthHandoffMode(isDebug: true, devSchemeRequested: true),
      'dev_scheme',
    );
  });

  test('current build policy follows the compile-time opt-in', () {
    const requested = bool.fromEnvironment('CRAFTSKY_DEV_OAUTH_SCHEME');
    expect(
      oauthHandoffModeForCurrentBuild(),
      requested ? 'dev_scheme' : 'verified_link',
    );
  });
}
