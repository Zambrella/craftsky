import 'dart:ui';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('AppError presentation', () {
    final l10n = lookupAppLocalizations(const Locale('en'));

    test('renders localized safe copy for every app error case', () {
      for (final kind in AppErrorKind.values) {
        final message = AppError(kind).message(l10n);

        expect(message, isNotEmpty, reason: kind.name);
        expect(message, isNot(kind.name));
      }
    });

    test('does not expose forbidden diagnostic text in user copy', () {
      const forbidden = [
        'Exception:',
        'StateError',
        'requestId',
        'req_123',
        'HTTP 500',
        'internal_error',
        'did:plc:alice',
        'alice.craftsky.social',
        'Bearer',
        'https://',
        '{"text"',
      ];

      final message = const AppError(
        AppErrorKind.initializationFailed,
        safeDiagnostics: {
          'raw': 'Exception: boot failed requestId=req_123 did:plc:alice',
        },
      ).message(l10n);

      for (final token in forbidden) {
        expect(message, isNot(contains(token)), reason: token);
      }
    });

    test('renders localized action labels from app error kind', () {
      expect(
        const AppError(AppErrorKind.initializationFailed).actionLabel(l10n),
        l10n.retryButton,
      );
      expect(
        const AppError(AppErrorKind.navigationFailed).actionLabel(l10n),
        l10n.goHomeButton,
      );
      expect(
        const AppError(AppErrorKind.sessionExpired).actionLabel(l10n),
        l10n.errorActionSignIn,
      );
      expect(
        const AppError(AppErrorKind.contentUnavailable).actionLabel(l10n),
        isNull,
      );
    });
  });
}
