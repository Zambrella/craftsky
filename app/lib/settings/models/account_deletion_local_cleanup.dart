enum AccountDeletionLocalArtifact {
  draftsAndStagedMedia,
  instagramVerificationSnapshot,
  imageCaches,
  accountScopedProviderState,
  ordinarySession,
  deletionJobId,
  deletionStatusCredential,
  deletionDisplayIdentity,
}

final class AccountDeletionLocalCleanupPlan {
  const AccountDeletionLocalCleanupPlan._({
    required this.erase,
    required this.preserve,
  });

  factory AccountDeletionLocalCleanupPlan.forAcceptedDeletion() =>
      const AccountDeletionLocalCleanupPlan._(
        erase: {
          AccountDeletionLocalArtifact.draftsAndStagedMedia,
          AccountDeletionLocalArtifact.instagramVerificationSnapshot,
          AccountDeletionLocalArtifact.imageCaches,
          AccountDeletionLocalArtifact.accountScopedProviderState,
          AccountDeletionLocalArtifact.ordinarySession,
        },
        preserve: {
          AccountDeletionLocalArtifact.deletionJobId,
          AccountDeletionLocalArtifact.deletionStatusCredential,
          AccountDeletionLocalArtifact.deletionDisplayIdentity,
        },
      );

  final Set<AccountDeletionLocalArtifact> erase;
  final Set<AccountDeletionLocalArtifact> preserve;
}
