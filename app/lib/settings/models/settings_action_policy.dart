final class SettingsActionPolicy {
  const SettingsActionPolicy({
    required this.requiresConfirmation,
    required this.invokesSessionSignOut,
    required this.startsAccountDeletion,
  });

  final bool requiresConfirmation;
  final bool invokesSessionSignOut;
  final bool startsAccountDeletion;
}

const settingsSignOutPolicy = SettingsActionPolicy(
  requiresConfirmation: false,
  invokesSessionSignOut: true,
  startsAccountDeletion: false,
);

const deleteAccountPolicy = SettingsActionPolicy(
  requiresConfirmation: true,
  invokesSessionSignOut: false,
  startsAccountDeletion: true,
);
