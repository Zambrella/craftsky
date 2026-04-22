import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
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
      SignedIn(did: did, handle: 'test.bsky.social', token: 'tok');
}

class PendingOnboardingStatus extends OnboardingStatus {
  @override
  bool build(String did) => false;
}

class CompletedOnboardingStatus extends OnboardingStatus {
  @override
  bool build(String did) => true;
}
