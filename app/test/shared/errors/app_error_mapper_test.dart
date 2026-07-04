import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('AppErrorMapper', () {
    test('maps API failures to safe finite cases', () {
      expect(
        AppErrorMapper.map(const ApiUnauthorized()).kind,
        AppErrorKind.sessionExpired,
      );
      expect(
        AppErrorMapper.map(const ApiBadRequest('not_found')).kind,
        AppErrorKind.contentUnavailable,
      );
      expect(
        AppErrorMapper.map(const ApiNetworkError('offline')).kind,
        AppErrorKind.networkUnavailable,
      );
      expect(
        AppErrorMapper.map(const ApiServerError('http_500')).kind,
        AppErrorKind.serviceUnavailable,
      );
      expect(
        AppErrorMapper.map(const ApiCanceled()).kind,
        AppErrorKind.actionFailed,
      );
    });

    test('maps expected API failures as non-reportable', () {
      for (final error in [
        const ApiUnauthorized(),
        const ApiBadRequest('handle_required'),
        const ApiBadRequest('not_found'),
        const ApiNetworkError('offline'),
        const ApiCanceled(),
      ]) {
        expect(AppErrorMapper.map(error).reportable, isFalse);
      }
    });

    test('maps reportable failures with surface-specific fallbacks', () {
      expect(
        AppErrorMapper.map(
          StateError('secure storage failed'),
          source: AppErrorSource.storage,
        ).kind,
        AppErrorKind.storageUnavailable,
      );
      expect(
        AppErrorMapper.map(
          StateError('boot failed'),
          source: AppErrorSource.initialization,
        ).kind,
        AppErrorKind.initializationFailed,
      );
      expect(
        AppErrorMapper.map(
          const FormatException('bad json'),
          source: AppErrorSource.backgroundLoad,
        ).kind,
        AppErrorKind.backgroundLoadFailed,
      );
      expect(
        AppErrorMapper.map(
          StateError('bad route'),
          source: AppErrorSource.routing,
        ).kind,
        AppErrorKind.navigationFailed,
      );
    });

    test('keeps raw diagnostic text out of mapper diagnostics', () {
      final error = AppErrorMapper.map(
        StateError('token=secret did:plc:alice https://example.test/path?q=x'),
        source: AppErrorSource.action,
      );

      expect(error.safeDiagnostics.values.join(' '), isNot(contains('secret')));
      expect(
        error.safeDiagnostics.values.join(' '),
        isNot(contains('did:plc:alice')),
      );
      expect(
        error.safeDiagnostics.values.join(' '),
        isNot(contains('https://example.test')),
      );
    });
  });
}
