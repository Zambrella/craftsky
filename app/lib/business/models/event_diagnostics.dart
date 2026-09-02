import 'package:craftsky_app/l10n/generated/app_localizations.dart';

abstract final class EventDiagnostics {
  static List<String> localized(
    Iterable<String> codes,
    AppLocalizations l10n,
  ) {
    final seen = <String>{};
    final labels = <String>[];
    for (final code in codes) {
      if (!seen.add(code)) continue;
      final label = _label(code, l10n);
      if (label != null) labels.add(label);
    }
    return labels;
  }

  static String? _label(String code, AppLocalizations l10n) => switch (code) {
    'owner-not-business' => l10n.businessEventDiagnosticOwnerNotBusiness,
    'invalid-time-range' => l10n.businessEventDiagnosticInvalidTimeRange,
    'duration-exceeds-limit' =>
      l10n.businessEventDiagnosticDurationExceedsLimit,
    'record-moderated' => l10n.businessEventDiagnosticRecordModerated,
    'ended' => l10n.businessEventDiagnosticEnded,
    'cancelled' => l10n.businessEventDiagnosticCancelled,
    'postponed' => l10n.businessEventDiagnosticPostponed,
    _ => null,
  };
}
