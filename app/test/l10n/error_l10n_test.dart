import 'dart:ui';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_presenter.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('all finite error cases have generated English localizations', () {
    final l10n = lookupAppLocalizations(const Locale('en'));

    for (final kind in AppErrorKind.values) {
      final message = AppErrorPresenter.message(l10n, AppError(kind));
      expect(message, isNotEmpty, reason: kind.name);
    }
  });
}
