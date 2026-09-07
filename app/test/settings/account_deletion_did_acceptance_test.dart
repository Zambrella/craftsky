import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/settings/models/delete_account_confirmation.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'AT-007 persists the full server DID regardless of local handle state',
    () {
      const ownerDid = 'did:plc:alicefullidentifier';

      for (final handleState in {
        'valid': 'alice.test',
        'stale': 'old-alice.test',
        'invalid': 'handle.invalid',
        'absent': 'handle.invalid',
      }.entries) {
        final registry = SessionRegistry.empty().upsertAndActivate(
          token: 'craftsky-token',
          did: ownerDid,
          handle: handleState.value,
        );
        final pending = AccountDeletionLeaseFence.capture(registry).pending(
          jobId: '10000000-0000-4000-8000-000000000007',
          confirmationDid: ownerDid,
          expiresAt: DateTime.utc(2027),
        );
        final restored = SessionRegistry.fromJson(
          registry.stageAccountDeletion(pending).toJson(),
        ).pendingAccountDeletion!;

        expect(restored.confirmationDid, ownerDid, reason: handleState.key);
        expect(
          matchesDeletionConfirmationDid(
            confirmationDid: restored.confirmationDid,
            input: ownerDid,
          ),
          isTrue,
        );
        expect(
          matchesDeletionConfirmationDid(
            confirmationDid: restored.confirmationDid,
            input: 'did:plc:alicefullidentifieq',
          ),
          isFalse,
        );
      }
    },
  );
}
