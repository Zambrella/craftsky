import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';

/// Test-only [AuthStatus] that always starts unauthenticated, regardless of
/// the production build's dev-gated default. Pass
/// `UnauthenticatedAuthStatus.new` to `authStatusProvider.overrideWith(...)`
/// in `ProviderScope.overrides`.
class UnauthenticatedAuthStatus extends AuthStatus {
  @override
  bool build() => false;
}

/// Test-only [AuthStatus] that always starts authenticated. Pair with
/// [CompletedOnboardingStatus] or [PendingOnboardingStatus] when setting up
/// router-redirect scenarios.
class AuthenticatedAuthStatus extends AuthStatus {
  @override
  bool build() => true;
}

/// Test-only [OnboardingStatus] that starts pre-onboarding.
class PendingOnboardingStatus extends OnboardingStatus {
  @override
  bool build() => false;
}

/// Test-only [OnboardingStatus] that starts onboarded.
class CompletedOnboardingStatus extends OnboardingStatus {
  @override
  bool build() => true;
}
