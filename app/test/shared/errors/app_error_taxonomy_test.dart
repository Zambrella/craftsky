import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('AppError taxonomy', () {
    test('is finite and covers the intended user-facing outcomes', () {
      expect(AppErrorKind.values, [
        AppErrorKind.networkUnavailable,
        AppErrorKind.serviceUnavailable,
        AppErrorKind.sessionExpired,
        AppErrorKind.permissionDenied,
        AppErrorKind.contentUnavailable,
        AppErrorKind.storageUnavailable,
        AppErrorKind.initializationFailed,
        AppErrorKind.navigationFailed,
        AppErrorKind.actionFailed,
        AppErrorKind.backgroundLoadFailed,
        AppErrorKind.unexpected,
      ]);
    });

    test('every case has complete metadata', () {
      for (final kind in AppErrorKind.values) {
        final metadata = kind.metadata;

        expect(metadata.sentryClassification, isNotEmpty, reason: kind.name);
        expect(metadata.severity, isA<AppErrorSeverity>(), reason: kind.name);
      }
    });

    test('expected routine states are not reportable by default', () {
      for (final kind in [
        AppErrorKind.networkUnavailable,
        AppErrorKind.sessionExpired,
        AppErrorKind.permissionDenied,
        AppErrorKind.contentUnavailable,
      ]) {
        expect(kind.metadata.reportableByDefault, isFalse, reason: kind.name);
      }
    });

    test('defect and degraded-state fallbacks are reportable by default', () {
      for (final kind in [
        AppErrorKind.storageUnavailable,
        AppErrorKind.initializationFailed,
        AppErrorKind.navigationFailed,
        AppErrorKind.actionFailed,
        AppErrorKind.backgroundLoadFailed,
        AppErrorKind.unexpected,
      ]) {
        expect(kind.metadata.reportableByDefault, isTrue, reason: kind.name);
      }
    });
  });
}
