import 'package:craftsky_app/feed/composer/link_preview_controller.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class ComposerLinkPreviewCarousel extends StatelessWidget {
  const ComposerLinkPreviewCarousel({
    required this.selected,
    required this.current,
    required this.total,
    required this.loading,
    required this.onPrevious,
    required this.onNext,
    required this.onDismiss,
    super.key,
  });

  final SelectedLinkPreview? selected;
  final int current;
  final int total;
  final bool loading;
  final VoidCallback onPrevious;
  final VoidCallback onNext;
  final VoidCallback onDismiss;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final preview = selected;
    if (preview == null) {
      if (!loading) return const SizedBox.shrink();
      return Semantics(
        label: l10n.linkPreviewLoading,
        child: const LinearProgressIndicator(
          key: Key('link-preview-loading'),
        ),
      );
    }
    final theme = Theme.of(context);
    final radii = theme.extension<RadiusTheme>()!;
    final thumbnail = preview.preview.thumbnail;
    return Material(
      key: const Key('link-preview-carousel'),
      color: Colors.transparent,
      clipBehavior: Clip.antiAlias,
      borderRadius: BorderRadius.circular(radii.r2),
      child: Ink(
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(radii.r2),
          color: theme.colorScheme.surfaceContainerLow,
        ),
        child: DecoratedBox(
          key: const Key('link-preview-carousel-outline'),
          position: DecorationPosition.foreground,
          decoration: BoxDecoration(
            border: Border.all(color: theme.colorScheme.outline),
            borderRadius: BorderRadius.circular(radii.r2),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (thumbnail != null)
                LayoutBuilder(
                  builder: (context, constraints) => SizedBox(
                    height: (constraints.maxWidth / (16 / 9)).clamp(0, 180),
                    child: Image.memory(
                      thumbnail.bytes,
                      key: const Key('link-preview-thumbnail'),
                      fit: BoxFit.cover,
                      semanticLabel: l10n.externalCardThumbnail(
                        preview.preview.title,
                      ),
                      errorBuilder: (_, _, _) => ColoredBox(
                        color: theme.colorScheme.surfaceContainerHighest,
                      ),
                    ),
                  ),
                ),
              Padding(
                padding: const EdgeInsets.fromLTRB(12, 8, 4, 0),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        preview.preview.title,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: theme.textTheme.titleMedium,
                      ),
                    ),
                    IconButton(
                      tooltip: l10n.linkPreviewDismiss,
                      onPressed: onDismiss,
                      icon: const Icon(CraftskyIconsBold.close),
                    ),
                  ],
                ),
              ),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 12),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    if (preview.preview.description.isNotEmpty)
                      Text(
                        preview.preview.description,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                      ),
                    Text(
                      preview.navigationUri.host,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: theme.textTheme.labelLarge?.copyWith(
                        color: theme.colorScheme.outline,
                      ),
                    ),
                  ],
                ),
              ),
              Row(
                children: [
                  IconButton(
                    tooltip: l10n.linkPreviewPrevious,
                    onPressed: total > 1 ? onPrevious : null,
                    icon: const Icon(CraftskyIconsBold.previous),
                  ),
                  Expanded(
                    child: Text(
                      l10n.linkPreviewPosition(current, total),
                      textAlign: TextAlign.center,
                    ),
                  ),
                  IconButton(
                    tooltip: l10n.linkPreviewNext,
                    onPressed: total > 1 ? onNext : null,
                    icon: const Icon(CraftskyIconsBold.next),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}
