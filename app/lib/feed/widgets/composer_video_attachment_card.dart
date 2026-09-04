import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

final class ComposerVideoAttachmentCard extends StatelessWidget {
  const ComposerVideoAttachmentCard({
    required this.selection,
    required this.enabled,
    required this.onAltTextChanged,
    required this.onReplace,
    required this.onRemove,
    super.key,
  });

  final LocalVideoSelection selection;
  final bool enabled;
  final ValueChanged<String> onAltTextChanged;
  final VoidCallback onReplace;
  final VoidCallback onRemove;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final shadows = theme.extension<BrandShadowTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final semanticColors = theme.extension<SemanticColorsTheme>()!;
    final shadowOffset = shadows.dropSm.first.offset;

    return Padding(
      padding: EdgeInsets.only(
        right: shadowOffset.dx > 0 ? shadowOffset.dx : 0,
        bottom: shadowOffset.dy > 0 ? shadowOffset.dy : 0,
      ),
      child: Container(
        key: const Key('composer-video-attachment'),
        decoration: BoxDecoration(
          color: swatches.paper3,
          borderRadius: BorderRadius.circular(radii.r3),
          border: Border.all(color: theme.colorScheme.onSurface, width: 1.5),
          boxShadow: shadows.dropSm,
        ),
        padding: EdgeInsets.all(spacing.sp4),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Container(
                  width: 52,
                  height: 52,
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    color: swatches.butter,
                    shape: BoxShape.circle,
                    border: Border.all(
                      color: theme.colorScheme.onSurface,
                      width: 2,
                    ),
                  ),
                  child: const Icon(Icons.videocam_outlined, size: 30),
                ),
                SizedBox(width: spacing.sp3),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        l10n.postComposeVideoSelected,
                        style: theme.textTheme.titleMedium?.copyWith(
                          fontWeight: FontWeight.w900,
                        ),
                      ),
                      Text(
                        selection.displayName,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: theme.colorScheme.outline,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
            SizedBox(height: spacing.sp4),
            BrandTextField(
              textFieldKey: const Key('composer-video-alt-text'),
              label: l10n.postComposeAltTextLabel,
              initialValue: selection.altText,
              enabled: enabled,
              maxLength: 1000,
              maxLines: 3,
              minLines: 3,
              keyboardType: TextInputType.multiline,
              textInputAction: TextInputAction.newline,
              hintText: l10n.postComposeVideoAltHint,
              labelLeading: Icon(
                Icons.short_text_rounded,
                color: theme.colorScheme.onSurfaceVariant,
                size: 24,
              ),
              onChanged: onAltTextChanged,
            ),
            SizedBox(height: spacing.sp3),
            Wrap(
              alignment: WrapAlignment.end,
              spacing: spacing.sp2,
              runSpacing: spacing.sp2,
              children: [
                OutlinedButton.icon(
                  key: const Key('composer-replace-video'),
                  onPressed: enabled ? onReplace : null,
                  icon: const Icon(Icons.swap_horiz_rounded),
                  label: Text(l10n.postComposeReplaceVideo),
                ),
                OutlinedButton.icon(
                  key: const Key('composer-remove-video'),
                  onPressed: enabled ? onRemove : null,
                  style: OutlinedButton.styleFrom(
                    foregroundColor: semanticColors.error,
                  ),
                  icon: Icon(
                    Icons.delete_outline_rounded,
                    color: semanticColors.error,
                  ),
                  label: Text(l10n.postComposeRemoveVideo),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
