import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:flutter/foundation.dart';

/// Account-critical state that is ready for one exact active session lease.
@immutable
final class ActiveAccountInitialization {
  const ActiveAccountInitialization({
    required this.lease,
    required this.languagePreferences,
  });

  final ActiveAccountLease lease;
  final LanguagePreferences languagePreferences;

  @override
  bool operator ==(Object other) =>
      other is ActiveAccountInitialization &&
      other.lease == lease &&
      other.languagePreferences == languagePreferences;

  @override
  int get hashCode => Object.hash(lease, languagePreferences);

  @override
  String toString() => 'ActiveAccountInitialization(<redacted>)';
}
