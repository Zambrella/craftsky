import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/onboarding/models/onboarding_completion.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';

class SignedOutAuthSession extends AuthSession {
  @override
  Future<AuthState> build() async => const SignedOut();
}

class SignedInAuthSession extends AuthSession {
  SignedInAuthSession({this.did = 'did:plc:test'});
  final String did;

  @override
  Future<AuthState> build() async =>
      SignedIn(did: did, handle: 'test.bsky.social');
}

class PendingOnboardingStatus extends OnboardingStatus {
  @override
  Future<OnboardingCompletion> build(AccountSessionLease lease) async =>
      const OnboardingCompletion(completed: false);
}

class CompletedOnboardingStatus extends OnboardingStatus {
  @override
  Future<OnboardingCompletion> build(AccountSessionLease lease) async =>
      const OnboardingCompletion(completed: true);
}

ActiveAccountInitialization completedActiveAccountInitialization({
  String did = 'did:plc:test',
}) => ActiveAccountInitialization(
  lease: ActiveAccountLease(
    session: AccountSessionLease(
      account: AccountKey(did),
      sessionGeneration: 1,
    ),
    activationGeneration: 1,
  ),
  languagePreferences: const LanguagePreferences(
    primaryLanguage: 'en',
    contentLanguages: ['en'],
  ),
  onboardingComplete: true,
);
