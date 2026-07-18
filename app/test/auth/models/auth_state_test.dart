import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('SignedOut is const-equal', () {
    expect(const SignedOut(), const SignedOut());
  });

  test('SignedIn carries only redacted UI identity', () {
    final s = SignedIn(did: 'did:plc:a', handle: 'a.bsky.social');
    expect(s.did, 'did:plc:a');
    expect(s.handle, 'a.bsky.social');
    expect('$s', 'SignedIn(<redacted>)');
  });

  test('AuthState pattern-matches exhaustively', () {
    final values = <AuthState>[
      const SignedOut(),
      SignedIn(did: 'did:plc:test', handle: 'h.test'),
    ];
    for (final v in values) {
      final label = switch (v) {
        SignedOut() => 'out',
        SignedIn() => 'in',
      };
      expect(label, anyOf('out', 'in'));
    }
  });
}
