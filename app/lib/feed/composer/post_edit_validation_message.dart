import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';

/// Use localized control names, never server prose or raw schema paths.
String? postEditValidationMessage(AppLocalizations l10n, Object? error) {
  if (error is! ApiBadRequest || error.details.fields.isEmpty) return null;
  final labels = <String>{
    for (final field in error.details.fields.keys)
      switch (field) {
        'text' || 'facets' => l10n.postEditCaptionLabel,
        'sponsored' => l10n.postSponsoredToggleTitle,
        'langs' => l10n.languagesTitle,
        final value when value.startsWith('images') =>
          l10n.postComposePhotosTitle,
        final value when value.startsWith('embed.external') =>
          l10n.postEditContentLabel,
        final value when value.startsWith('embed.video') =>
          l10n.postComposeVideoSelected,
        'project.common.title' => l10n.projectComposerProjectTitleLabel,
        final value when value.startsWith('project.common.pattern') =>
          l10n.projectComposerPatternDetailsTitle,
        final value when value.startsWith('project') =>
          l10n.projectComposerMoreDetailsLabel,
        _ => l10n.postEditContentLabel,
      },
  };
  return l10n.postEditValidationFields(labels.join(', '));
}
