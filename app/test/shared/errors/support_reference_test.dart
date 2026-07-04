import 'dart:ui';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/errors/support_reference.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('SupportReference', () {
    test('is created only from non-empty captured event IDs', () {
      const eventId = '0123456789abcdef0123456789abcdef';

      expect(
        SupportReference.fromReportResult(
          const ReportResult.captured(eventId: eventId),
        )?.value,
        eventId,
      );
      expect(
        SupportReference.fromReportResult(
          const ReportResult.captured(eventId: ''),
        ),
        isNull,
      );
      expect(
        SupportReference.fromReportResult(const ReportResult.disabled()),
        isNull,
      );
      expect(
        SupportReference.fromReportResult(const ReportResult.failed()),
        isNull,
      );
    });

    test('formats localized copy without backend identifiers', () {
      final l10n = lookupAppLocalizations(const Locale('en'));
      const reference = SupportReference('0123456789abcdef0123456789abcdef');
      final label = reference.format(l10n);

      expect(label, contains(reference.value));
      expect(label, isNot(contains('requestId')));
      expect(label, isNot(contains('req_123')));
      expect(label, isNot(contains('HTTP 500')));
      expect(label, isNot(contains('internal_error')));
    });
  });
}
