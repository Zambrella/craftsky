import 'package:craftsky_app/account_eligibility/models/account_eligibility_status.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/foundation.dart';

part 'active_account_initialization.mapper.dart';

/// Account-critical state that is ready for one exact active session lease.
@immutable
@MappableClass(generateMethods: GenerateMethods.copy | GenerateMethods.equals)
final class ActiveAccountInitialization
    with ActiveAccountInitializationMappable {
  const ActiveAccountInitialization({
    required this.lease,
    required this.languagePreferences,
    required this.onboardingComplete,
    this.accountEligibility = const AccountEligibilityStatus(
      state: AccountEligibilityState.eligible,
      appealable: false,
    ),
  });

  final ActiveAccountLease lease;
  final LanguagePreferences languagePreferences;
  final bool onboardingComplete;
  final AccountEligibilityStatus accountEligibility;

  @override
  String toString() => 'ActiveAccountInitialization(<redacted>)';
}
