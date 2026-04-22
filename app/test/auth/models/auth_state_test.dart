import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('SignedOut is const-equal', () {
    expect(const SignedOut(), const SignedOut());
  });

  test('SignedIn carries did, handle, token', () {
    const s = SignedIn(did: 'did:plc:a', handle: 'a.bsky.social', token: 'tok');
    expect(s.did, 'did:plc:a');
    expect(s.handle, 'a.bsky.social');
    expect(s.token, 'tok');
  });

  test('AuthState pattern-matches exhaustively', () {
    const values = <AuthState>[
      SignedOut(),
      SignedIn(did: 'd', handle: 'h', token: 't'),
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
