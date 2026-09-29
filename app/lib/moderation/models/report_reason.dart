import 'package:craftsky_app/l10n/generated/app_localizations.dart';

enum ReportDestination { inApp, intellectualPropertyEmail }

enum ReportGroup {
  childSafety,
  sexualIntimate,
  danger,
  harassmentPrivacy,
  fraudIdentity,
  contentIntegrity,
  intellectualProperty,
  other;

  String label(AppLocalizations l10n) => switch (this) {
    ReportGroup.childSafety => l10n.reportGroupChildSafety,
    ReportGroup.sexualIntimate => l10n.reportGroupSexualIntimate,
    ReportGroup.danger => l10n.reportGroupDanger,
    ReportGroup.harassmentPrivacy => l10n.reportGroupHarassmentPrivacy,
    ReportGroup.fraudIdentity => l10n.reportGroupFraudIdentity,
    ReportGroup.contentIntegrity => l10n.reportGroupContentIntegrity,
    ReportGroup.intellectualProperty => l10n.reportGroupIntellectualProperty,
    ReportGroup.other => l10n.reportGroupOther,
  };

  List<ReportReason> get reasons => ReportReason.values
      .where((reason) => reason.group == this)
      .toList(growable: false);
}

enum ReportReason {
  childSexualExploitation(
    'childSexualExploitation',
    ReportGroup.childSafety,
  ),
  grooming('grooming', ReportGroup.childSafety),
  childAbuseMaterial('childAbuseMaterial', ReportGroup.childSafety),
  intimateImageAbuse('intimateImageAbuse', ReportGroup.sexualIntimate),
  sexualExploitation('sexualExploitation', ReportGroup.sexualIntimate),
  adultContent('adultContent', ReportGroup.sexualIntimate),
  graphicContent('graphicContent', ReportGroup.sexualIntimate),
  immediateDanger('immediateDanger', ReportGroup.danger),
  credibleThreat('credibleThreat', ReportGroup.danger),
  terrorism('terrorism', ReportGroup.danger),
  violence('violence', ReportGroup.danger),
  selfHarm('selfHarm', ReportGroup.danger),
  harassment('harassment', ReportGroup.harassmentPrivacy),
  hate('hate', ReportGroup.harassmentPrivacy),
  stalking('stalking', ReportGroup.harassmentPrivacy),
  doxxing('doxxing', ReportGroup.harassmentPrivacy),
  blockEvasion('blockEvasion', ReportGroup.harassmentPrivacy),
  privacy('privacy', ReportGroup.harassmentPrivacy),
  fraud('fraud', ReportGroup.fraudIdentity),
  scam('scam', ReportGroup.fraudIdentity),
  phishing('phishing', ReportGroup.fraudIdentity),
  impersonation('impersonation', ReportGroup.fraudIdentity),
  counterfeit('counterfeit', ReportGroup.fraudIdentity),
  spam('spam', ReportGroup.contentIntegrity),
  platformManipulation('platformManipulation', ReportGroup.contentIntegrity),
  misleading('misleading', ReportGroup.contentIntegrity),
  suspectedAiGenerated('suspectedAiGenerated', ReportGroup.contentIntegrity),
  offTopic('offTopic', ReportGroup.contentIntegrity),
  intellectualProperty(
    'intellectualProperty',
    ReportGroup.intellectualProperty,
    destination: ReportDestination.intellectualPropertyEmail,
  ),
  other('other', ReportGroup.other);

  const ReportReason(
    this.reasonType,
    this.group, {
    this.destination = ReportDestination.inApp,
  });

  final String reasonType;
  final ReportGroup group;
  final ReportDestination destination;

  Uri get externalUri => Uri(
    scheme: 'mailto',
    path: 'moderation@craftsky.social',
    queryParameters: {
      'subject': 'CraftSky copyright or trade mark report',
      'body':
          'Please identify the protected work, your authority, and the '
          'CraftSky URL or AT URI. Do not attach media.',
    },
  );

  String label(AppLocalizations l10n) => switch (this) {
    ReportReason.childSexualExploitation =>
      l10n.reportReasonChildSexualExploitation,
    ReportReason.grooming => l10n.reportReasonGrooming,
    ReportReason.childAbuseMaterial => l10n.reportReasonChildAbuseMaterial,
    ReportReason.intimateImageAbuse => l10n.reportReasonIntimateImageAbuse,
    ReportReason.sexualExploitation => l10n.reportReasonSexualExploitation,
    ReportReason.adultContent => l10n.reportReasonAdultContent,
    ReportReason.graphicContent => l10n.reportReasonGraphicContent,
    ReportReason.immediateDanger => l10n.reportReasonImmediateDanger,
    ReportReason.credibleThreat => l10n.reportReasonCredibleThreat,
    ReportReason.terrorism => l10n.reportReasonTerrorism,
    ReportReason.violence => l10n.reportReasonViolence,
    ReportReason.selfHarm => l10n.reportReasonSelfHarm,
    ReportReason.harassment => l10n.reportReasonHarassment,
    ReportReason.hate => l10n.reportReasonHate,
    ReportReason.stalking => l10n.reportReasonStalking,
    ReportReason.doxxing => l10n.reportReasonDoxxing,
    ReportReason.blockEvasion => l10n.reportReasonBlockEvasion,
    ReportReason.privacy => l10n.reportReasonPrivacy,
    ReportReason.fraud => l10n.reportReasonFraud,
    ReportReason.scam => l10n.reportReasonScam,
    ReportReason.phishing => l10n.reportReasonPhishing,
    ReportReason.impersonation => l10n.reportReasonImpersonation,
    ReportReason.counterfeit => l10n.reportReasonCounterfeit,
    ReportReason.spam => l10n.reportReasonSpam,
    ReportReason.platformManipulation => l10n.reportReasonPlatformManipulation,
    ReportReason.misleading => l10n.reportReasonMisleading,
    ReportReason.suspectedAiGenerated => l10n.reportReasonSuspectedAiGenerated,
    ReportReason.offTopic => l10n.reportReasonOffTopic,
    ReportReason.intellectualProperty => l10n.reportReasonIntellectualProperty,
    ReportReason.other => l10n.reportReasonOther,
  };
}
