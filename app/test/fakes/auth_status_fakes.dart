import 'package:craftsky_app/auth/providers/auth_status_provider.dart';

/// Test-only [AuthStatus] that always starts unauthenticated, regardless of
/// the production build's dev-gated default. Pass
/// `UnauthenticatedAuthStatus.new` to `authStatusProvider.overrideWith(...)`
/// in `ProviderScope.overrides`.
class UnauthenticatedAuthStatus extends AuthStatus {
  @override
  bool build() => false;
}
