import 'package:craftsky_app/settings/models/account_deletion_local_cleanup.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('cleanup erases product data and ordinary credentials only', () {
    final plan = AccountDeletionLocalCleanupPlan.forAcceptedDeletion();

    expect(
      plan.erase,
      containsAll(<AccountDeletionLocalArtifact>{
        AccountDeletionLocalArtifact.draftsAndStagedMedia,
        AccountDeletionLocalArtifact.instagramVerificationSnapshot,
        AccountDeletionLocalArtifact.imageCaches,
        AccountDeletionLocalArtifact.accountScopedProviderState,
        AccountDeletionLocalArtifact.ordinarySession,
      }),
    );
    expect(
      plan.preserve,
      containsAll(<AccountDeletionLocalArtifact>{
        AccountDeletionLocalArtifact.deletionJobId,
        AccountDeletionLocalArtifact.deletionStatusCredential,
        AccountDeletionLocalArtifact.deletionDisplayIdentity,
      }),
    );
    expect(plan.erase.intersection(plan.preserve), isEmpty);
  });
}
