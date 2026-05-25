import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('SignedOut is const-equal', () {
    expect(const SignedOut(), const SignedOut());
  });

  test('SignedIn carries did, handle, token', () {
    final s = SignedIn(did: 'did:plc:a', handle: 'a.bsky.social', token: 'tok');
    expect(s.did, 'did:plc:a');
    expect(s.handle, 'a.bsky.social');
    expect(s.token, 'tok');
  });

  test('AuthState pattern-matches exhaustively', () {
    final values = <AuthState>[
      const SignedOut(),
      SignedIn(did: 'did:plc:test', handle: 'h.test', token: 't'),
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
