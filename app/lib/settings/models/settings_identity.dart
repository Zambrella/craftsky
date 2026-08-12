import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';

final class SettingsIdentity {
  const SettingsIdentity({
    required this.primaryLabel,
    required this.handleLabel,
    required this.avatarSeed,
    required this.customisation,
    this.secondaryLabel,
    this.avatarUrl,
  });

  final String primaryLabel;
  final String? secondaryLabel;
  final String handleLabel;
  final String? avatarUrl;
  final String avatarSeed;
  final ProfileCustomisation customisation;

  @override
  String toString() => 'SettingsIdentity(<redacted>)';
}

SettingsIdentity projectSettingsIdentity({
  required AccountSessionLease lease,
  required StoredSession session,
  ActiveAccountIdentity? loaded,
}) {
  final profile = loaded?.lease == lease ? loaded?.profile : null;
  final handle = profile?.handle.value ?? session.handle.value;
  final handleLabel = '@$handle';
  final displayName = (profile?.displayName ?? session.cachedDisplayName)
      ?.trim();
  final hasDisplayName = displayName != null && displayName.isNotEmpty;

  return SettingsIdentity(
    primaryLabel: hasDisplayName ? displayName : handleLabel,
    secondaryLabel: hasDisplayName ? handleLabel : null,
    handleLabel: handleLabel,
    avatarUrl: profile?.avatar ?? session.cachedAvatarUrl,
    avatarSeed: session.did.value,
    customisation: profile?.customisation ?? session.cachedCustomisation,
  );
}
