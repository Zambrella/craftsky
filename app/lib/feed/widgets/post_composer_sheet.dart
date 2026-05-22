import 'package:animated_list_plus/animated_list_plus.dart';
import 'package:animated_list_plus/transitions.dart';
import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/create_post_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:uuid/uuid.dart';

Future<Post?> showPostComposerSheet(
  BuildContext context, {
  Post? replyTarget,
}) {
  return Navigator.of(context, rootNavigator: true).push<Post?>(
    MaterialPageRoute<Post?>(
      fullscreenDialog: true,
      builder: (_) => PostComposerSheet(replyTarget: replyTarget),
    ),
  );
}

class PostComposerSheet extends ConsumerStatefulWidget {
  const PostComposerSheet({super.key, this.replyTarget});

  static const maxCharacters = 2000;

  final Post? replyTarget;

  @override
  ConsumerState<PostComposerSheet> createState() => _PostComposerSheetState();
}

class _PostComposerSheetState extends ConsumerState<PostComposerSheet> {
  static const _imageListAnimationDuration = Duration(milliseconds: 220);

  final _controller = TextEditingController();
  final _focusNode = FocusNode(debugLabel: 'postComposerText');
  final String _composerId = const Uuid().v4();
  String _initialText = '';
  String _text = '';

  @override
  void initState() {
    super.initState();
    if (widget.replyTarget?.reply != null) {
      _text = '@${widget.replyTarget!.author.handle} ';
      _controller.text = _text;
      _controller.selection = TextSelection.collapsed(offset: _text.length);
    }
    _initialText = _text;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      _focusNode.requestFocus();
    });
  }

  @override
  void dispose() {
    _controller.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final createState = ref.watch(createPostProvider);
    final imagesState = ref.watch(composerImagesProvider(_composerId));
    final imagesNotifier = ref.read(
      composerImagesProvider(_composerId).notifier,
    );
    final isReply = widget.replyTarget != null;
    final trimmedText = _text.trim();
    final tooLong = _text.length > PostComposerSheet.maxCharacters;
    final canSubmit =
        !createState.isLoading &&
        trimmedText.isNotEmpty &&
        !tooLong &&
        imagesState.canSubmitImages();
    final submitLabel = isReply
        ? l10n.postComposeReplySubmit
        : l10n.postComposeSubmit;
    final describedImageCount = imagesState.images
        .where((image) => image.altText.trim().isNotEmpty)
        .length;
    final hasDraft = _hasDraft(imagesState);

    ref
      ..listen(createPostProvider, (previous, next) {
        switch ((previous, next)) {
          case (AsyncLoading(), AsyncData(:final value?)):
            if (Navigator.of(context).canPop()) {
              Navigator.of(context).pop(value);
            }
            context.showInfo(l10n.postCreateSuccess);
            ref.read(createPostProvider.notifier).reset();
          case (AsyncLoading(), AsyncError()):
            context.showError(l10n.postCreateError);
            ref.read(createPostProvider.notifier).reset();
          case _:
            break;
        }
      })
      ..listen(composerImagesProvider(_composerId), (previous, next) {
        final notice = next.notice;
        if (notice == null || previous?.notice?.id == notice.id) return;
        switch (notice) {
          case ImageSelectionLimitNotice(:final maxImages):
            context.showError('You can add up to $maxImages images');
          case UnsupportedImagesNotice(:final count):
            context.showError(
              count == 1
                  ? 'Unsupported image type'
                  : '$count unsupported images',
            );
          case ImagePickerFailedNotice():
            context.showError('Could not open image picker');
        }
        imagesNotifier.clearNotice(notice.id);
      });

    return PopScope<Post?>(
      canPop: !hasDraft || createState.isLoading,
      onPopInvokedWithResult: (didPop, _) async {
        if (didPop) return;
        final discard = await _confirmDiscard();
        if (!discard) return;
        if (!context.mounted) return;
        Navigator.of(context).pop();
      },
      child: Scaffold(
        backgroundColor: swatches.paper,
        appBar: AppBar(
          title: Text(
            isReply ? l10n.postComposeReplyTitle : l10n.postComposeTitle,
            style: theme.textTheme.titleLarge,
          ),
          actions: [
            Padding(
              padding: EdgeInsets.only(right: spacing.sp4),
              child: _PostAction(
                isSaving: createState.isLoading,
                label: submitLabel,
                onPressed: canSubmit
                    ? () => ref
                          .read(createPostProvider.notifier)
                          .create(
                            text: trimmedText,
                            reply: _replyFor(widget.replyTarget),
                            images: isReply
                                ? null
                                : ref
                                      .read(
                                        composerImagesProvider(_composerId),
                                      )
                                      .toCreatePostImages(),
                          )
                    : null,
              ),
            ),
          ],
        ),
        body: SafeArea(
          top: false,
          bottom: false,
          child: SingleChildScrollView(
            clipBehavior: Clip.none,
            padding: EdgeInsets.fromLTRB(
              spacing.sp4,
              spacing.sp5,
              spacing.sp4,
              spacing.sp7,
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                if (widget.replyTarget case final replyTarget?) ...[
                  _ReplyTargetPreview(post: replyTarget),
                  SizedBox(height: spacing.sp4),
                ],
                BrandTextField(
                  label: isReply
                      ? l10n.postComposeReplyHint
                      : l10n.postComposeHint,
                  hintText: isReply
                      ? null
                      : "Pattern, fabric, what went right, what didn't...",
                  controller: _controller,
                  focusNode: _focusNode,
                  minLines: isReply ? 5 : 3,
                  maxLines: 12,
                  textInputAction: TextInputAction.newline,
                  keyboardType: TextInputType.multiline,
                  enabled: !createState.isLoading,
                  errorText: tooLong ? l10n.postComposeTooLong : null,
                  helperText:
                      '${_text.length}/${PostComposerSheet.maxCharacters}',
                  helperAlignment: AlignmentDirectional.centerEnd,
                  onChanged: (value) => setState(() => _text = value),
                ),
                if (!isReply) ...[
                  SizedBox(height: spacing.sp6),
                  _PhotosHeader(
                    imageCount: imagesState.images.length,
                    describedImageCount: describedImageCount,
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
                      areItemsTheSame: (oldItem, newItem) =>
                          oldItem.id == newItem.id,
                      onReorderFinished: (image, oldIndex, newIndex, newItems) {
                        imagesNotifier.reorder(
                          fromIndex: oldIndex,
                          toIndex: newIndex,
                        );
                      },
                      itemBuilder: (context, animation, image, index) {
                        return Reorderable(
                          key: ValueKey('composer-image-${image.id}'),
                          child: SizeFadeTransition(
                            animation: animation,
                            curve: Curves.easeOutCubic,
                            sizeFraction: 0.85,
                            child: _DraftImageTile(
                              image: image,
                              index: index,
                              altTextKey: Key('composer-alt-${image.id}'),
                              removeKey: Key('composer-remove-${image.id}'),
                              moveUpKey: Key('composer-move-up-${image.id}'),
                              moveDownKey: Key(
                                'composer-move-down-${image.id}',
                              ),
                              canMoveUp: index > 0,
                              canMoveDown:
                                  index < imagesState.images.length - 1,
                              onAltChanged: (value) =>
                                  imagesNotifier.setAltText(
                                    image.id,
                                    value,
                                  ),
                              onRemove: () => imagesNotifier.remove(image.id),
                              onMoveUp: () => imagesNotifier.reorder(
                                fromIndex: index,
                                toIndex: index - 1,
                              ),
                              onMoveDown: () => imagesNotifier.reorder(
                                fromIndex: index,
                                toIndex: index + 1,
                              ),
                            ),
                          ),
                        );
                      },
                      removeItemBuilder: (context, animation, image) {
                        return Reorderable(
                          key: ValueKey('composer-image-${image.id}'),
                          child: SizeFadeTransition(
                            animation: animation,
                            curve: Curves.easeOutCubic,
                            sizeFraction: 0.85,
                            child: _DraftImageTile(
                              image: image,
                              index: 0,
                              altTextKey: Key('composer-alt-${image.id}'),
                              removeKey: Key('composer-remove-${image.id}'),
                              moveUpKey: Key('composer-move-up-${image.id}'),
                              moveDownKey: Key(
                                'composer-move-down-${image.id}',
                              ),
                              canMoveUp: false,
                              canMoveDown: false,
                              onAltChanged: (_) {},
                              onRemove: null,
                              onMoveUp: null,
                              onMoveDown: null,
                            ),
                          ),
                        );
                      },
                    ),
                  if (imagesState.images.length < mediaConfig.maxImages)
                    _AddPhotoCard(
                      remainingCount:
                          mediaConfig.maxImages - imagesState.images.length,
                      hasImages: imagesState.images.isNotEmpty,
                      onPressed: createState.isLoading
                          ? null
                          : () async {
                              if (imagesState.images.length >=
                                  mediaConfig.maxImages) {
                                context.showError(
                                  'You can add up to '
                                  '${mediaConfig.maxImages} images',
                                );
                                return;
                              }
                              await imagesNotifier.addImages();
                            },
                    ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }

  bool _hasDraft(ComposerImagesState imagesState) {
    return _text != _initialText || imagesState.images.isNotEmpty;
  }

  Future<bool> _confirmDiscard() {
    final l10n = AppLocalizations.of(context);
    return showCraftskyConfirmDialog(
      context,
      title: l10n.postComposeDiscardTitle,
      message: l10n.postComposeDiscardMessage,
      confirmLabel: l10n.postComposeDiscardConfirm,
      cancelLabel: l10n.postComposeDiscardCancel,
    );
  }
}

class _PhotosHeader extends StatelessWidget {
  const _PhotosHeader({
    required this.imageCount,
    required this.describedImageCount,
  });

  final int imageCount;
  final int describedImageCount;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final colors = theme.colorScheme;
    final describedLabel = imageCount == 0
        ? '0 described'
        : '$describedImageCount / $imageCount described';

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Expanded(
              child: Text(
                'Photos',
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w800,
                ),
              ),
            ),
            Icon(Icons.short_text_rounded, color: colors.outline, size: 22),
            SizedBox(width: spacing.sp1),
            Text(
              describedLabel,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: colors.outline,
                fontWeight: FontWeight.w800,
              ),
            ),
          ],
        ),
        SizedBox(height: spacing.sp1),
        Text(
          imageCount == 0
              ? 'Up to ${mediaConfig.maxImages} photos'
              : '$imageCount/${mediaConfig.maxImages} · drag to reorder · first is the cover',
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
    required this.altTextKey,
    required this.removeKey,
    required this.moveUpKey,
    required this.moveDownKey,
    required this.canMoveUp,
    required this.canMoveDown,
    required this.onAltChanged,
    required this.onRemove,
    required this.onMoveUp,
    required this.onMoveDown,
  });

  final ComposerImageDraft image;
  final int index;
  final Key altTextKey;
  final Key removeKey;
  final Key moveUpKey;
  final Key moveDownKey;
  final bool canMoveUp;
  final bool canMoveDown;
  final ValueChanged<String> onAltChanged;
  final VoidCallback? onRemove;
  final VoidCallback? onMoveUp;
  final VoidCallback? onMoveDown;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final shadows = theme.extension<BrandShadowTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final colors = theme.colorScheme;
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
                    icon: Icons.arrow_upward_rounded,
                    tooltip: 'Move image up',
                    onPressed: canMoveUp ? onMoveUp : null,
                  ),
                  SizedBox(width: spacing.sp2),
                  _CircleIconButton(
                    key: moveDownKey,
                    icon: Icons.arrow_downward_rounded,
                    tooltip: 'Move image down',
                    onPressed: canMoveDown ? onMoveDown : null,
                  ),
                  SizedBox(width: spacing.sp2),
                  _CircleIconButton(
                    key: removeKey,
                    icon: Icons.delete_outline_rounded,
                    tooltip: 'Remove image',
                    foregroundColor: BrandColors.red,
                    onPressed: onRemove,
                  ),
                  SizedBox(width: spacing.sp2),
                  Handle(
                    child: Tooltip(
                      message: 'Drag to reorder',
                      child: Icon(
                        Icons.drag_indicator_rounded,
                        color: colors.outline,
                        size: 34,
                      ),
                    ),
                  ),
                ],
              ),
              SizedBox(height: spacing.sp3),
              _DraftImagePreview(image: image),
              if (image.phase is ImageFailed) ...[
                SizedBox(height: spacing.sp3),
                _ImageStatus(image: image),
              ],
              SizedBox(height: spacing.sp4),
              BrandTextField(
                key: altTextKey,
                label: 'ALT TEXT',
                initialValue: image.altText,
                hintText: _altTextHint,
                minLines: 2,
                maxLines: 4,
                keyboardType: TextInputType.multiline,
                textInputAction: TextInputAction.newline,
                onChanged: onAltChanged,
                labelLeading: Icon(
                  Icons.short_text_rounded,
                  color: colors.onSurfaceVariant,
                  size: 24,
                ),
                labelStyle: theme.textTheme.labelSmall?.copyWith(
                  color: colors.onSurfaceVariant,
                  letterSpacing: 1.5,
                  fontWeight: FontWeight.w900,
                ),
                labelTrailing: Text(
                  hasAltText ? 'Described' : 'Help screen readers',
                  style: theme.textTheme.labelMedium?.copyWith(
                    color: hasAltText ? BrandColors.moss : colors.error,
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

class _DraftImagePreview extends StatelessWidget {
  const _DraftImagePreview({required this.image});

  final ComposerImageDraft image;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;

    return DecoratedBox(
      position: DecorationPosition.foreground,
      decoration: BoxDecoration(
        border: Border.all(color: colors.onSurface),
      ),
      child: AnimatedSize(
        duration: const Duration(milliseconds: 180),
        curve: Curves.easeOutCubic,
        alignment: Alignment.topCenter,
        child: AspectRatio(
          aspectRatio: _previewAspectRatio(image),
          child: Stack(
            fit: StackFit.expand,
            children: [
              switch (image.previewBytes) {
                final bytes? => Image.memory(
                  bytes,
                  key: Key('composer-preview-${image.id}'),
                  fit: BoxFit.cover,
                  width: double.infinity,
                ),
                null => const DecoratedBox(
                  decoration: BoxDecoration(color: Color(0xFFEAEAEA)),
                ),
              },
              if (_previewLoadingOverlay(image) case final overlay?)
                _ImageLoadingOverlay(imageId: image.id, overlay: overlay),
            ],
          ),
        ),
      ),
    );
  }
}

class _ImageLoadingOverlay extends StatelessWidget {
  const _ImageLoadingOverlay({required this.imageId, required this.overlay});

  final String imageId;
  final _PreviewLoadingOverlay overlay;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;

    return Semantics(
      label: overlay.label,
      liveRegion: true,
      child: DecoratedBox(
        key: Key('composer-preview-overlay-$imageId'),
        decoration: const BoxDecoration(color: Color(0x99000000)),
        child: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              SizedBox.square(
                dimension: 72,
                child: CircularProgressIndicator(
                  key: Key('composer-upload-progress-$imageId'),
                  value: overlay.value,
                  strokeWidth: 6,
                  backgroundColor: Colors.white24,
                  valueColor: const AlwaysStoppedAnimation<Color>(
                    Colors.white,
                  ),
                ),
              ),
              SizedBox(height: spacing.sp3),
              Text(
                overlay.label,
                key: Key('composer-upload-label-$imageId'),
                textAlign: TextAlign.center,
                style: theme.textTheme.titleMedium?.copyWith(
                  color: Colors.white,
                  fontWeight: FontWeight.w900,
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
          _statusLabel(image),
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
    return Container(
      width: 50,
      height: 50,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: isCover ? theme.colorScheme.primary : BrandColors.paper2,
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
    final enabled = onPressed != null;

    return IconButton(
      tooltip: tooltip,
      onPressed: onPressed,
      icon: Icon(icon),
      style: IconButton.styleFrom(
        fixedSize: const Size(52, 52),
        backgroundColor: BrandColors.paper3,
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
  });

  final int remainingCount;
  final bool hasImages;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final colors = theme.colorScheme;
    final label = hasImages ? 'Add another photo' : 'Add a photo';
    final subtitle = hasImages
        ? 'Up to $remainingCount more'
        : 'Up to ${mediaConfig.maxImages} photos';

    return Padding(
      padding: EdgeInsets.only(bottom: spacing.sp2),
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          key: const Key('composer-add-image'),
          borderRadius: BorderRadius.circular(24),
          onTap: onPressed,
          child: CustomPaint(
            painter: _DashedBorderPainter(
              color: colors.onSurface,
              radius: 24,
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
                      color: BrandColors.butter,
                      shape: BoxShape.circle,
                      border: Border.all(color: colors.onSurface, width: 2),
                    ),
                    child: const Icon(Icons.add_rounded, size: 34),
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

String _statusLabel(ComposerImageDraft image) => switch (image.phase) {
  ImageQueued() || ImageReading() => 'Reading image',
  ImagePreparing() => 'Preparing image',
  ImageUploading() => 'Uploading image',
  ImageUploaded() => 'Uploaded',
  ImageFailed() => 'Failed',
};

const _altTextHint =
    'Describe the image for someone who cannot see it, including the craft, '
    'materials, colors, and important details.';

_PreviewLoadingOverlay? _previewLoadingOverlay(ComposerImageDraft image) {
  return switch (image.phase) {
    ImageQueued() || ImageReading() => const _PreviewLoadingOverlay(
      label: 'Reading image',
    ),
    ImagePreparing() => const _PreviewLoadingOverlay(label: 'Preparing image'),
    ImageUploading(:final progress) => _uploadLoadingOverlay(progress),
    ImageUploaded() || ImageFailed() => null,
  };
}

_PreviewLoadingOverlay _uploadLoadingOverlay(ImageTransferProgress progress) {
  if (_isProcessingUpload(progress)) {
    return const _PreviewLoadingOverlay(label: 'Processing');
  }

  final value = progress.indicatorValue;
  if (value == null) {
    return const _PreviewLoadingOverlay(label: 'Uploading image');
  }

  final percent = (value * 100).round().clamp(0, 99);
  return _PreviewLoadingOverlay(label: 'Uploading $percent%', value: value);
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

class _ReplyTargetPreview extends StatelessWidget {
  const _ReplyTargetPreview({required this.post});

  final Post post;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final displayName = post.author.displayName;
    return DecoratedBox(
      decoration: BoxDecoration(
        color: swatches.paper2,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: swatches.borderHair),
      ),
      child: Padding(
        padding: EdgeInsets.all(spacing.sp3),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (displayName != null && displayName.trim().isNotEmpty)
              Text(displayName, style: theme.textTheme.titleSmall),
            Text(
              '@${post.author.handle}',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
            ),
            SizedBox(height: spacing.sp2),
            Text(
              post.text,
              maxLines: 3,
              overflow: TextOverflow.ellipsis,
              style: theme.textTheme.bodyMedium,
            ),
          ],
        ),
      ),
    );
  }
}

class _PostAction extends StatelessWidget {
  const _PostAction({
    required this.isSaving,
    required this.label,
    required this.onPressed,
  });

  final bool isSaving;
  final String label;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    return TextButton(
      onPressed: onPressed,
      style: TextButton.styleFrom(
        textStyle: const TextStyle(fontWeight: FontWeight.w800),
      ),
      child: isSaving ? const StitchProgressIndicator(size: 18) : Text(label),
    );
  }
}

PostReply? _replyFor(Post? target) {
  if (target == null) return null;

  return PostReply(
    root: target.reply?.root ?? PostRef(uri: target.uri, cid: target.cid),
    parent: PostRef(uri: target.uri, cid: target.cid),
  );
}
