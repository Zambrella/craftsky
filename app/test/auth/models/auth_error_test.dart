import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('AuthError pattern-matches exhaustively', () {
    final values = <AuthError>[
      const HandleRequired(),
      const InvalidHandle(),
      const ServerUnavailable(),
      const BrowserLaunchFailed(),
      const SignInTimedOut(),
      StorageFailure(Exception('oops')),
      RegistrationFailure.canceled,
    ];
    for (final e in values) {
      final label = switch (e) {
        HandleRequired() => 'handle_required',
        InvalidHandle() => 'invalid_handle',
        ServerUnavailable() => 'server_unavailable',
        BrowserLaunchFailed() => 'browser_launch_failed',
        SignInTimedOut() => 'timed_out',
        StorageFailure() => 'storage',
        RegistrationFailure() => e.name,
      };
      expect(label, isNotEmpty);
    }
  });

  test('StorageFailure preserves its cause', () {
    final cause = Exception('keystore down');
    expect(StorageFailure(cause).cause, same(cause));
  });

  test('UT-007 maps only approved registration failure values', () {
    for (final failure in RegistrationFailure.values) {
      expect(RegistrationFailure.fromCallback(failure.name), failure);
    }

    expect(
      RegistrationFailure.fromStartError(
        const ApiServerError(
          'http_502',
          details: ApiFailureDetails(
            statusCode: 502,
            appViewError: 'registration_provider_unavailable',
          ),
        ),
      ),
      RegistrationFailure.providerUnavailable,
    );
    expect(
      RegistrationFailure.fromStartError(
        const ApiServerError(
          'http_502',
          details: ApiFailureDetails(
            statusCode: 502,
            appViewError: 'registration_incomplete',
          ),
        ),
      ),
      RegistrationFailure.registrationIncomplete,
    );
    expect(
      RegistrationFailure.fromStartError(const ApiNetworkError('secret')),
      RegistrationFailure.providerUnavailable,
    );
    expect(
      RegistrationFailure.fromStartError(const ApiServerError('secret')),
      RegistrationFailure.providerUnavailable,
    );
    expect(
      RegistrationFailure.fromStartError(const BrowserLaunchFailed()),
      RegistrationFailure.registrationIncomplete,
    );

    for (final untrusted in <Object?>[
      null,
      '',
      'access_denied',
      'registration_incomplete',
      'providerUnavailable: issuer secret',
      const ApiBadRequest('provider_error_with_internal_detail'),
      StateError('issuer/token/lifecycle detail'),
    ]) {
      final mapped = untrusted == null || untrusted is String
          ? RegistrationFailure.fromCallback(untrusted as String?)
          : RegistrationFailure.fromStartError(untrusted);
      expect(mapped, isNull, reason: '$untrusted');
    }
  });
}
