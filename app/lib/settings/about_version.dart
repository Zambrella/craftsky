import 'package:craftsky_app/l10n/generated/app_localizations.dart';

String? buildVersionLabel(
  AppLocalizations l10n, {
  required String? version,
  required String? buildNumber,
}) {
  final normalizedVersion = version?.trim();
  if (normalizedVersion == null || normalizedVersion.isEmpty) return null;

  final normalizedBuild = buildNumber?.trim();
  return normalizedBuild == null || normalizedBuild.isEmpty
      ? normalizedVersion
      : l10n.navigationBuildVersion(normalizedVersion, normalizedBuild);
}
