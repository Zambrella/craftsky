import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/settings/models/settings_identity.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('projectSettingsIdentity', () {
    final aliceLease = AccountSessionLease(
      account: AccountKey('did:plc:alice'),
      sessionGeneration: 3,
    );
    final aliceSession = StoredSession(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
      sessionGeneration: 3,
      cachedDisplayName: 'Cached Alice',
      cachedAvatarUrl: 'https://cdn.test/alice-cached.jpg',
    );

    test(
      'uses the current loaded profile and normalizes the visible handle',
      () {
        final identity = projectSettingsIdentity(
          lease: aliceLease,
          session: aliceSession,
          loaded: ActiveAccountIdentity(
            lease: aliceLease,
            profile: Profile(
              did: 'did:plc:alice',
              handle: 'alice.test',
              displayName: ' Alice ',
              avatar: 'https://cdn.test/alice.jpg',
              crafts: const [],
            ),
          ),
        );

        expect(identity.primaryLabel, 'Alice');
        expect(identity.secondaryLabel, '@alice.test');
        expect(identity.handleLabel, '@alice.test');
        expect(identity.avatarUrl, 'https://cdn.test/alice.jpg');
        expect(identity.avatarSeed, 'did:plc:alice');
      },
    );

    test('shows the handle once when the display name is blank', () {
      final identity = projectSettingsIdentity(
        lease: aliceLease,
        session: aliceSession,
        loaded: ActiveAccountIdentity(
          lease: aliceLease,
          profile: Profile(
            did: 'did:plc:alice',
            handle: 'alice.test',
            displayName: '   ',
            crafts: const [],
          ),
        ),
      );

      expect(identity.primaryLabel, '@alice.test');
      expect(identity.secondaryLabel, isNull);
      expect(identity.primaryLabel, isNot(contains('did:')));
      expect(identity.primaryLabel, isNot('No username'));
    });

    test('discards a loaded profile from a stale account lease', () {
      final identity = projectSettingsIdentity(
        lease: aliceLease,
        session: aliceSession,
        loaded: ActiveAccountIdentity(
          lease: AccountSessionLease(
            account: AccountKey('did:plc:bob'),
            sessionGeneration: 7,
          ),
          profile: Profile(
            did: 'did:plc:bob',
            handle: 'bob.test',
            displayName: 'Bob',
            avatar: 'https://cdn.test/bob.jpg',
            crafts: const [],
          ),
        ),
      );

      expect(identity.primaryLabel, 'Cached Alice');
      expect(identity.secondaryLabel, '@alice.test');
      expect(identity.avatarUrl, 'https://cdn.test/alice-cached.jpg');
      expect(identity.primaryLabel, isNot('Bob'));
      expect(identity.avatarUrl, isNot('https://cdn.test/bob.jpg'));
    });
  });
}
