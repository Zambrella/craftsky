import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('AuthError pattern-matches exhaustively', () {
    final values = <AuthError>[
      const HandleRequired(),
      const InvalidHandle(),
      const ServerUnavailable(),
      const BrowserLaunchFailed(),
      const NoPendingSignIn(),
      const SignInTimedOut(),
      StorageFailure(Exception('oops')),
    ];
    for (final e in values) {
      final label = switch (e) {
        HandleRequired() => 'handle_required',
        InvalidHandle() => 'invalid_handle',
        ServerUnavailable() => 'server_unavailable',
        BrowserLaunchFailed() => 'browser_launch_failed',
        NoPendingSignIn() => 'no_pending',
        SignInTimedOut() => 'timed_out',
        StorageFailure() => 'storage',
      };
      expect(label, isNotEmpty);
    }
  });

  test('StorageFailure preserves its cause', () {
    final cause = Exception('keystore down');
    expect(StorageFailure(cause).cause, same(cause));
  });
}
