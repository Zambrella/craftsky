import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('pending and attention states route only to deletion status', () {
    for (final status in [
      AccountDeletionStatus.active,
      AccountDeletionStatus.retrying,
      AccountDeletionStatus.needsAttention,
    ]) {
      expect(
        DeletionLoginPolicy.destination(status: status),
        DeletionLoginDestination.status,
      );
    }
  });

  test('terminal success allows fresh onboarding and restores nothing', () {
    expect(
      DeletionLoginPolicy.destination(status: AccountDeletionStatus.deleted),
      DeletionLoginDestination.freshOnboarding,
    );
  });

  test('an absent deletion follows ordinary authentication', () {
    expect(
      DeletionLoginPolicy.destination(status: null),
      DeletionLoginDestination.ordinaryAuthentication,
    );
  });
}
