import 'package:craftsky_app/l10n/generated/app_localizations.dart';

enum ReportReason {
  harassment('harassment'),
  hate('hate'),
  spam('spam'),
  misleading('misleading'),
  suspectedAiGenerated('suspected_ai_generated'),
  adultOrGraphic('adult_or_graphic'),
  impersonation('impersonation'),
  offTopic('off_topic'),
  intellectualProperty('intellectual_property'),
  other('other');

  const ReportReason(this.reasonType);

  final String reasonType;

  String label(AppLocalizations l10n) => switch (this) {
    ReportReason.harassment => l10n.reportReasonHarassment,
    ReportReason.hate => l10n.reportReasonHate,
    ReportReason.spam => l10n.reportReasonSpam,
    ReportReason.misleading => l10n.reportReasonMisleading,
    ReportReason.suspectedAiGenerated => l10n.reportReasonSuspectedAiGenerated,
    ReportReason.adultOrGraphic => l10n.reportReasonAdultOrGraphic,
    ReportReason.impersonation => l10n.reportReasonImpersonation,
    ReportReason.offTopic => l10n.reportReasonOffTopic,
    ReportReason.intellectualProperty => l10n.reportReasonIntellectualProperty,
    ReportReason.other => l10n.reportReasonOther,
  };
}
