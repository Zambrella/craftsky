import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';

final class SupportReference {
  const SupportReference(this.value);

  final String value;

  static SupportReference? fromReportResult(ReportResult result) {
    final eventId = result.eventId;
    if (result.status != ReportStatus.captured ||
        eventId == null ||
        eventId.isEmpty ||
        eventId == '00000000000000000000000000000000') {
      return null;
    }
    return SupportReference(eventId);
  }

  String format(AppLocalizations l10n) {
    return l10n.supportReferenceLabel(value);
  }
}
