import 'package:craftsky_app/settings/models/settings_action_policy.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('Sign out and Delete account have distinct action policies', () {
    expect(settingsSignOutPolicy.requiresConfirmation, isFalse);
    expect(settingsSignOutPolicy.invokesSessionSignOut, isTrue);
    expect(settingsSignOutPolicy.startsAccountDeletion, isFalse);

    expect(deleteAccountPolicy.requiresConfirmation, isTrue);
    expect(deleteAccountPolicy.invokesSessionSignOut, isFalse);
    expect(deleteAccountPolicy.startsAccountDeletion, isTrue);
  });
}
