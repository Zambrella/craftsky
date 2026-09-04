import 'dart:async';

import 'package:animated_list_plus/animated_list_plus.dart';
import 'package:animated_list_plus/transitions.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/image/craftsky_image_attachment_preview.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class ComposerImageAttachmentSection extends StatelessWidget {
  const ComposerImageAttachmentSection({
    required this.imagesState,
    required this.enabled,
    required this.onAddImages,
    required this.onAltTextChanged,
    required this.onRemove,
    required this.onReplaceUnavailable,
    required this.onReorder,
    super.key,
    this.validationErrorText,
    this.required = false,
    this.requiredLabel = 'required',
    this.maxImages = 4,
    this.imageUrlFor,
    this.onReplace,
    this.keyPrefix = 'composer',
  });

  static const _imageListAnimationDuration = Duration(milliseconds: 220);

  final ComposerImagesState imagesState;
  final bool enabled;
  final Future<void> Function()? onAddImages;
  final void Function(String imageId, String value) onAltTextChanged;
  final void Function(String imageId) onRemove;
  final Future<void> Function(String imageId) onReplaceUnavailable;
  final void Function(int fromIndex, int toIndex) onReorder;
  final String? validationErrorText;
  final bool required;
  final String requiredLabel;
  final int maxImages;
  final String? Function(ComposerImageDraft image)? imageUrlFor;
  final Future<void> Function(String imageId)? onReplace;
  final String keyPrefix;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final describedImageCount = imagesState.images
        .where((image) => image.altText.trim().isNotEmpty)
        .length;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _PhotosHeader(
          imageCount: imagesState.images.length,
          describedImageCount: describedImageCount,
          required: required,
          requiredLabel: requiredLabel,
          maxImages: maxImages,
        ),
        SizedBox(height: spacing.sp3),
        if (imagesState.images.isNotEmpty)
          ImplicitlyAnimatedReorderableList<ComposerImageDraft>(
            items: imagesState.images,
            shrinkWrap: true,
            clipBehavior: Clip.none,
            physics: const NeverScrollableScrollPhysics(),
            padding: EdgeInsets.zero,
            reorderDuration: _imageListAnimationDuration,
            insertDuration: _imageListAnimationDuration,
            removeDuration: _imageListAnimationDuration,
            areItemsTheSame: (oldItem, newItem) => oldItem.id == newItem.id,
            onReorderFinished: (image, oldIndex, newIndex, newItems) {
              if (!enabled || oldIndex == newIndex) return;
              onReorder(oldIndex, newIndex);
            },
            itemBuilder: (context, animation, image, index) {
              return Reorderable(
                key: ValueKey('$keyPrefix-image-${image.id}'),
                child: SizeFadeTransition(
                  animation: animation,
                  curve: Curves.easeOutCubic,
                  sizeFraction: 0.85,
                  child: _DraftImageTile(
                    image: image,
                    index: index,
                    enabled: enabled,
                    altTextKey: Key('$keyPrefix-alt-${image.id}'),
                    removeKey: Key('$keyPrefix-remove-${image.id}'),
                    moveUpKey: Key('$keyPrefix-move-up-${image.id}'),
                    moveDownKey: Key('$keyPrefix-move-down-${image.id}'),
                    previewKey: Key('$keyPrefix-preview-${image.id}'),
                    previewUrl: imageUrlFor?.call(image),
                    canMoveUp: index > 0,
                    canMoveDown: index < imagesState.images.length - 1,
                    onAltChanged: (value) => onAltTextChanged(image.id, value),
                    onRemove: () => onRemove(image.id),
                    onReplaceUnavailable: () => onReplaceUnavailable(image.id),
                    onReplace: onReplace == null
                        ? null
                        : () => onReplace!(image.id),
                    onMoveUp: () => onReorder(index, index - 1),
                    onMoveDown: () => onReorder(index, index + 1),
                  ),
                ),
              );
            },
            removeItemBuilder: (context, animation, image) {
              return Reorderable(
                key: ValueKey('$keyPrefix-image-${image.id}'),
                child: SizeFadeTransition(
                  animation: animation,
                  curve: Curves.easeOutCubic,
                  sizeFraction: 0.85,
                  child: _DraftImageTile(
                    image: image,
                    index: 0,
                    enabled: false,
                    altTextKey: Key('$keyPrefix-alt-${image.id}'),
                    removeKey: Key('$keyPrefix-remove-${image.id}'),
                    moveUpKey: Key('$keyPrefix-move-up-${image.id}'),
                    moveDownKey: Key('$keyPrefix-move-down-${image.id}'),
                    previewKey: Key('$keyPrefix-preview-${image.id}'),
                    previewUrl: imageUrlFor?.call(image),
                    canMoveUp: false,
                    canMoveDown: false,
                    onAltChanged: (_) {},
                    onRemove: () {},
                    onReplaceUnavailable: () async {},
                    onReplace: null,
                    onMoveUp: () {},
                    onMoveDown: () {},
                  ),
                ),
              );
            },
          ),
        if (imagesState.images.length < maxImages)
          _AddPhotoCard(
            remainingCount: maxImages - imagesState.images.length,
            maxImages: maxImages,
            addKey: Key('$keyPrefix-add-image'),
            hasImages: imagesState.images.isNotEmpty,
            onPressed: enabled && onAddImages != null
                ? () => unawaited(onAddImages!())
                : null,
          ),
        if (validationErrorText != null) ...[
          SizedBox(height: spacing.sp2),
          Text(
            validationErrorText!,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.error,
            ),
          ),
        ],
      ],
    );
  }
}

class _PhotosHeader extends StatelessWidget {
  const _PhotosHeader({
    required this.imageCount,
    required this.describedImageCount,
    required this.required,
    required this.requiredLabel,
    required this.maxImages,
  });

  final int imageCount;
  final int describedImageCount;
  final bool required;
  final String requiredLabel;
  final int maxImages;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final colors = theme.colorScheme;
    final l10n = AppLocalizations.of(context);
    final titleStyle = theme.textTheme.titleMedium?.copyWith(
      fontWeight: FontWeight.w800,
    );
    final requiredLabelStyle = theme.textTheme.labelSmall?.copyWith(
      color: colors.onSurfaceVariant,
      fontWeight: FontWeight.w600,
    );
    final describedLabel = imageCount == 0
        ? l10n.postComposeNoImagesDescribed
        : l10n.postComposeImagesDescribed(describedImageCount, imageCount);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text.rich(
          TextSpan(
            children: [
              TextSpan(text: l10n.postComposePhotosTitle),
              if (required)
                TextSpan(
                  text: '  $requiredLabel',
                  style: requiredLabelStyle,
                ),
            ],
          ),
          style: titleStyle,
          textAlign: TextAlign.start,
        ),
        SizedBox(height: spacing.sp1),
        Row(
          children: [
            Icon(CraftskyIcons.altText, color: colors.outline, size: 22),
            SizedBox(width: spacing.sp1),
            Expanded(
              child: Text(
                describedLabel,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: colors.outline,
                  fontWeight: FontWeight.w800,
                ),
              ),
            ),
          ],
        ),
        SizedBox(height: spacing.sp1),
        Text(
          imageCount == 0
              ? l10n.postComposePhotosLimitHelper(maxImages)
              : l10n.postComposePhotosReorderHelper(
                  imageCount,
                  maxImages,
                ),
          style: theme.textTheme.bodyMedium?.copyWith(
            color: colors.outline,
            fontWeight: FontWeight.w700,
          ),
        ),
      ],
    );
  }
}

class _DraftImageTile extends StatelessWidget {
  const _DraftImageTile({
    required this.image,
    required this.index,
    required this.enabled,
    required this.altTextKey,
    required this.removeKey,
    required this.moveUpKey,
    required this.moveDownKey,
    required this.canMoveUp,
    required this.canMoveDown,
    required this.onAltChanged,
    required this.onRemove,
    required this.onReplaceUnavailable,
    required this.onReplace,
    required this.onMoveUp,
    required this.onMoveDown,
    required this.previewKey,
    required this.previewUrl,
  });

  final ComposerImageDraft image;
  final int index;
  final bool enabled;
  final Key altTextKey;
  final Key removeKey;
  final Key moveUpKey;
  final Key moveDownKey;
  final bool canMoveUp;
  final bool canMoveDown;
  final ValueChanged<String> onAltChanged;
  final VoidCallback onRemove;
  final Future<void> Function() onReplaceUnavailable;
  final Future<void> Function()? onReplace;
  final VoidCallback onMoveUp;
  final VoidCallback onMoveDown;
  final Key previewKey;
  final String? previewUrl;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final shadows = theme.extension<BrandShadowTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final semanticColors = theme.extension<SemanticColorsTheme>()!;
    final colors = theme.colorScheme;
    final l10n = AppLocalizations.of(context);
    final hasAltText = image.altText.trim().isNotEmpty;
    final shadowOffset = shadows.dropSm.first.offset;

    return Padding(
      padding: EdgeInsets.only(
        right: shadowOffset.dx > 0 ? shadowOffset.dx : 0,
        bottom: spacing.sp5 + (shadowOffset.dy > 0 ? shadowOffset.dy : 0),
      ),
      child: Container(
        decoration: BoxDecoration(
          color: swatches.paper3,
          borderRadius: BorderRadius.circular(radii.r3),
          border: Border.all(color: colors.onSurface, width: 1.5),
          boxShadow: shadows.dropSm,
        ),
        child: Padding(
          padding: EdgeInsets.all(spacing.sp4),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                children: [
                  _ImageNumberBadge(index: index),
                  const Spacer(),
                  _CircleIconButton(
                    key: moveUpKey,
                    icon: CraftskyIconsBold.moveUp,
                    tooltip: l10n.postComposeMoveImageUp,
                    onPressed: enabled && canMoveUp ? onMoveUp : null,
                  ),
                  SizedBox(width: spacing.sp2),
                  _CircleIconButton(
                    key: moveDownKey,
                    icon: CraftskyIconsBold.moveDown,
                    tooltip: l10n.postComposeMoveImageDown,
                    onPressed: enabled && canMoveDown ? onMoveDown : null,
                  ),
                  SizedBox(width: spacing.sp2),
                  _CircleIconButton(
                    key: removeKey,
                    icon: CraftskyIconsBold.delete,
                    tooltip: l10n.postComposeRemoveImage,
                    foregroundColor: semanticColors.error,
                    onPressed: enabled ? onRemove : null,
                  ),
                  SizedBox(width: spacing.sp2),
                  Handle(
                    child: Tooltip(
                      message: l10n.postComposeDragToReorder,
                      child: Icon(
                        CraftskyIconsBold.reorder,
                        color: colors.outline,
                        size: 34,
                      ),
                    ),
                  ),
                ],
              ),
              SizedBox(height: spacing.sp3),
              _composerImagePreview(
                context,
                image,
                imageKey: previewKey,
                imageUrl: previewUrl,
              ),
              if (onReplace != null && image.phase is! ImageUnavailable) ...[
                SizedBox(height: spacing.sp3),
                Align(
                  alignment: Alignment.centerLeft,
                  child: TextButton.icon(
                    key: Key('composer-replace-${image.id}'),
                    onPressed: enabled ? () => unawaited(onReplace!()) : null,
                    icon: const Icon(CraftskyIconsBold.chooseImage),
                    label: Text(l10n.draftsReplaceImageAction),
                  ),
                ),
              ],
              if (image.phase is ImageUnavailable) ...[
                SizedBox(height: spacing.sp3),
                _ImageStatus(image: image),
                SizedBox(height: spacing.sp1),
                Align(
                  alignment: Alignment.centerLeft,
                  child: TextButton.icon(
                    key: Key('composer-replace-${image.id}'),
                    onPressed: enabled
                        ? () => unawaited(onReplaceUnavailable())
                        : null,
                    icon: const Icon(CraftskyIconsBold.chooseImage),
                    label: Text(l10n.draftsReplaceImageAction),
                  ),
                ),
              ],
              if (image.phase is ImageFailed) ...[
                SizedBox(height: spacing.sp3),
                _ImageStatus(image: image),
              ],
              SizedBox(height: spacing.sp4),
              BrandTextField(
                key: altTextKey,
                label: l10n.postComposeAltTextLabel,
                initialValue: image.altText,
                hintText: l10n.postComposeAltTextHint,
                minLines: 2,
                maxLines: 4,
                keyboardType: TextInputType.multiline,
                textInputAction: TextInputAction.newline,
                enabled: enabled,
                onChanged: onAltChanged,
                labelLeading: Icon(
                  CraftskyIcons.altText,
                  color: colors.onSurfaceVariant,
                  size: 24,
                ),
                labelStyle: theme.textTheme.labelSmall?.copyWith(
                  color: colors.onSurfaceVariant,
                  letterSpacing: 1.5,
                  fontWeight: FontWeight.w900,
                ),
                labelTrailing: Text(
                  hasAltText
                      ? l10n.postComposeImageDescribed
                      : l10n.postComposeImageNeedsAltText,
                  style: theme.textTheme.labelMedium?.copyWith(
                    color: hasAltText ? semanticColors.success : colors.error,
                    fontWeight: FontWeight.w800,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ImageStatus extends StatelessWidget {
  const _ImageStatus({required this.image});

  final ComposerImageDraft image;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final colors = theme.colorScheme;
    final failed = image.phase is ImageFailed;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          _statusLabel(context, image),
          style: theme.textTheme.bodyMedium?.copyWith(
            color: failed ? colors.error : colors.outline,
            fontWeight: FontWeight.w700,
          ),
        ),
        if (image.phase case ImageFailed(:final failure)) ...[
          SizedBox(height: spacing.sp1),
          Text(
            failure.message,
            style: theme.textTheme.bodyMedium?.copyWith(color: colors.error),
          ),
        ],
      ],
    );
  }
}

class _ImageNumberBadge extends StatelessWidget {
  const _ImageNumberBadge({required this.index});

  final int index;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isCover = index == 0;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    return Container(
      width: 50,
      height: 50,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: isCover ? theme.colorScheme.primary : swatches.paper2,
        shape: BoxShape.circle,
        border: Border.all(color: theme.colorScheme.onSurface, width: 2),
      ),
      child: Text(
        '${index + 1}',
        style: theme.textTheme.titleLarge?.copyWith(
          color: isCover
              ? theme.colorScheme.onPrimary
              : theme.colorScheme.onSurface,
          fontWeight: FontWeight.w900,
        ),
      ),
    );
  }
}

class _CircleIconButton extends StatelessWidget {
  const _CircleIconButton({
    required this.icon,
    required this.tooltip,
    required this.onPressed,
    this.foregroundColor,
    super.key,
  });

  final IconData icon;
  final String tooltip;
  final VoidCallback? onPressed;
  final Color? foregroundColor;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final enabled = onPressed != null;

    return IconButton(
      tooltip: tooltip,
      onPressed: onPressed,
      icon: Icon(icon),
      style: IconButton.styleFrom(
        fixedSize: const Size(52, 52),
        backgroundColor: swatches.paper3,
        foregroundColor: enabled
            ? foregroundColor ?? colors.onSurface
            : colors.outlineVariant,
        disabledForegroundColor: colors.outlineVariant,
        side: BorderSide(
          color: enabled ? colors.onSurface : colors.outlineVariant,
          width: 2,
        ),
        shape: const CircleBorder(),
      ),
    );
  }
}

class _AddPhotoCard extends StatelessWidget {
  const _AddPhotoCard({
    required this.remainingCount,
    required this.hasImages,
    required this.onPressed,
    required this.maxImages,
    required this.addKey,
  });

  final int remainingCount;
  final bool hasImages;
  final VoidCallback? onPressed;
  final int maxImages;
  final Key addKey;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final colors = theme.colorScheme;
    final l10n = AppLocalizations.of(context);
    final label = hasImages
        ? l10n.postComposeAddAnotherPhoto
        : l10n.postComposeAddPhoto;
    final subtitle = hasImages
        ? l10n.postComposePhotosRemaining(remainingCount)
        : l10n.postComposePhotosLimitHelper(maxImages);

    return Padding(
      padding: EdgeInsets.only(bottom: spacing.sp2),
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          key: addKey,
          borderRadius: BorderRadius.circular(24),
          onTap: onPressed,
          child: CustomPaint(
            painter: _DashedBorderPainter(
              color: colors.onSurface,
              radius: radii.r4,
              strokeWidth: 1.8,
            ),
            child: Padding(
              padding: EdgeInsets.symmetric(
                horizontal: spacing.sp4,
                vertical: spacing.sp4,
              ),
              child: Row(
                children: [
                  Container(
                    width: 56,
                    height: 56,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      color: onPressed == null
                          ? colors.surfaceContainerHighest
                          : swatches.butter,
                      shape: BoxShape.circle,
                      border: Border.all(color: colors.onSurface, width: 2),
                    ),
                    child: const Icon(CraftskyIconsBold.add, size: 34),
                  ),
                  SizedBox(width: spacing.sp4),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          label,
                          style: theme.textTheme.titleMedium?.copyWith(
                            fontWeight: FontWeight.w900,
                          ),
                        ),
                        Text(
                          subtitle,
                          style: theme.textTheme.bodyMedium?.copyWith(
                            color: colors.outline,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _DashedBorderPainter extends CustomPainter {
  const _DashedBorderPainter({
    required this.color,
    required this.radius,
    required this.strokeWidth,
  });

  final Color color;
  final double radius;
  final double strokeWidth;

  @override
  void paint(Canvas canvas, Size size) {
    const dashLength = 6.0;
    const gapLength = 6.0;
    final paint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = strokeWidth;
    final rect = Offset.zero & size;
    final rrect = RRect.fromRectAndRadius(
      rect.deflate(strokeWidth / 2),
      Radius.circular(radius),
    );
    final path = Path()..addRRect(rrect);

    for (final metric in path.computeMetrics()) {
      var distance = 0.0;
      while (distance < metric.length) {
        final end = (distance + dashLength).clamp(0, metric.length).toDouble();
        canvas.drawPath(metric.extractPath(distance, end), paint);
        distance = end + gapLength;
      }
    }
  }

  @override
  bool shouldRepaint(covariant _DashedBorderPainter oldDelegate) {
    return oldDelegate.color != color ||
        oldDelegate.radius != radius ||
        oldDelegate.strokeWidth != strokeWidth;
  }
}

double _previewAspectRatio(ComposerImageDraft image) {
  final aspectRatio = switch (image.phase) {
    ImageUploaded(:final uploaded) =>
      uploaded.aspectRatio ?? image.previewAspectRatio,
    _ => image.previewAspectRatio,
  };
  if (aspectRatio != null && aspectRatio.height > 0) {
    return (aspectRatio.width / aspectRatio.height).clamp(0.7, 1.6);
  }

  return 1;
}

Widget _composerImagePreview(
  BuildContext context,
  ComposerImageDraft image, {
  required Key imageKey,
  required String? imageUrl,
}) {
  final overlay = _previewLoadingOverlay(context, image);
  return CraftskyImageAttachmentPreview(
    bytes: image.previewBytes,
    imageUrl: imageUrl,
    aspectRatio: _previewAspectRatio(image),
    unavailable: image.phase is ImageUnavailable,
    busyLabel: overlay?.label,
    busyProgress: overlay?.value,
    imageKey: imageKey,
    overlayKey: Key('composer-preview-overlay-${image.id}'),
    progressKey: Key('composer-upload-progress-${image.id}'),
    labelKey: Key('composer-upload-label-${image.id}'),
  );
}

String _statusLabel(BuildContext context, ComposerImageDraft image) {
  final l10n = AppLocalizations.of(context);
  return switch (image.phase) {
    ImageQueued() || ImageReading() => l10n.postComposeReadingImage,
    ImagePreparing() => l10n.postComposePreparingImage,
    ImageUploading() => l10n.postComposeUploadingImage,
    ImageReady() => l10n.postComposeUploadedImage,
    ImageUploaded() => l10n.postComposeUploadedImage,
    ScheduledImageReady() => l10n.postComposeUploadedImage,
    ImageUnavailable() => l10n.draftsImageUnavailable,
    ImageFailed() => l10n.postComposeImageFailed,
  };
}

_PreviewLoadingOverlay? _previewLoadingOverlay(
  BuildContext context,
  ComposerImageDraft image,
) {
  final l10n = AppLocalizations.of(context);
  return switch (image.phase) {
    ImageQueued() || ImageReading() => _PreviewLoadingOverlay(
      label: l10n.postComposeReadingImage,
    ),
    ImagePreparing() => _PreviewLoadingOverlay(
      label: l10n.postComposePreparingImage,
    ),
    ImageUploading(:final progress) => _uploadLoadingOverlay(l10n, progress),
    ImageReady() ||
    ImageUploaded() ||
    ScheduledImageReady() ||
    ImageUnavailable() ||
    ImageFailed() => null,
  };
}

_PreviewLoadingOverlay _uploadLoadingOverlay(
  AppLocalizations l10n,
  ImageTransferProgress progress,
) {
  if (_isProcessingUpload(progress)) {
    return _PreviewLoadingOverlay(label: l10n.postComposeProcessingImage);
  }

  final value = progress.indicatorValue;
  if (value == null) {
    return _PreviewLoadingOverlay(label: l10n.postComposeUploadingImage);
  }

  final percent = (value * 100).round().clamp(0, 99);
  return _PreviewLoadingOverlay(
    label: l10n.postComposeUploadingProgress(percent),
    value: value,
  );
}

bool _isProcessingUpload(ImageTransferProgress progress) {
  return switch (progress) {
    TransferFinalizing() => true,
    TransferBytes(:final sent, :final sendTotal) =>
      sendTotal > 0 && sent >= sendTotal,
    TransferStarting() => false,
  };
}

class _PreviewLoadingOverlay {
  const _PreviewLoadingOverlay({required this.label, this.value});

  final String label;
  final double? value;
}
