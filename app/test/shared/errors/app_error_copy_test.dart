import 'dart:ui';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_presenter.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('AppErrorPresenter', () {
    final l10n = lookupAppLocalizations(const Locale('en'));

    test('renders localized safe copy for every app error case', () {
      for (final kind in AppErrorKind.values) {
        final message = AppErrorPresenter.message(l10n, AppError(kind));

        expect(message, isNotEmpty, reason: kind.name);
        expect(message, isNot(kind.metadata.localizationKey));
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

      final message = AppErrorPresenter.message(
        l10n,
        const AppError(
          AppErrorKind.initializationFailed,
          safeDiagnostics: {
            'raw': 'Exception: boot failed requestId=req_123 did:plc:alice',
          },
        ),
      );

      for (final token in forbidden) {
        expect(message, isNot(contains(token)), reason: token);
      }
    });

    test('renders localized action labels from action policy', () {
      expect(
        AppErrorPresenter.actionLabel(
          l10n,
          const AppError(AppErrorKind.initializationFailed),
        ),
        l10n.retryButton,
      );
      expect(
        AppErrorPresenter.actionLabel(
          l10n,
          const AppError(AppErrorKind.navigationFailed),
        ),
        l10n.goHomeButton,
      );
      expect(
        AppErrorPresenter.actionLabel(
          l10n,
          const AppError(AppErrorKind.sessionExpired),
        ),
        l10n.errorActionSignIn,
      );
      expect(
        AppErrorPresenter.actionLabel(
          l10n,
          const AppError(AppErrorKind.contentUnavailable),
        ),
        isNull,
      );
    });
  });
}
