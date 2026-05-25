import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_image_page_indicator.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const _galleryMinScale = 1.0;
const _galleryMaxScale = 4.0;
const _zoomedScaleEpsilon = 0.01;

class GalleryImage {
  const GalleryImage({required this.alt, this.thumb, this.fullsize});

  factory GalleryImage.fromPostImage(PostImage image) {
    return GalleryImage(
      alt: image.alt,
      thumb: image.thumb,
      fullsize: image.fullsize,
    );
  }

  final String alt;
  final String? thumb;
  final String? fullsize;
}

Future<void> showImageGallery(
  BuildContext context, {
  required List<GalleryImage> images,
  int initialIndex = 0,
}) {
  if (images.isEmpty) return Future<void>.value();

  return Navigator.of(context, rootNavigator: true).push<void>(
    MaterialPageRoute<void>(
      fullscreenDialog: true,
      builder: (context) {
        final viewPadding = MediaQuery.of(context).viewPadding;
        final spacing =
            Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
        return Scaffold(
          backgroundColor: Colors.black,
          body: Stack(
            children: [
              Positioned.fill(
                child: PostImageGallery.items(
                  galleryImages: images,
                  initialIndex: initialIndex,
                ),
              ),
              Positioned(
                left: viewPadding.left + spacing.sp2,
                top: viewPadding.top + spacing.sp2,
                child: DecoratedBox(
                  key: const Key('post-image-gallery-close-background'),
                  decoration: BoxDecoration(
                    color: Colors.black.withValues(alpha: 0.55),
                    shape: BoxShape.circle,
                  ),
                  child: const CloseButton(color: Colors.white),
                ),
              ),
            ],
          ),
        );
      },
    ),
  );
}

Future<void> showPostImageGallery(
  BuildContext context, {
  required List<PostImage> images,
  int initialIndex = 0,
}) {
  return showImageGallery(
    context,
    images: images.map(GalleryImage.fromPostImage).toList(),
    initialIndex: initialIndex,
  );
}

class PostImageGallery extends ConsumerStatefulWidget {
  PostImageGallery({
    required this.images,
    this.initialIndex = 0,
    super.key,
  }) : galleryImages = images.map(GalleryImage.fromPostImage).toList();

  const PostImageGallery.items({
    required this.galleryImages,
    this.initialIndex = 0,
    super.key,
  }) : images = const [];

  final List<PostImage> images;
  final List<GalleryImage> galleryImages;
  final int initialIndex;

  @override
  ConsumerState<PostImageGallery> createState() => _PostImageGalleryState();
}

class _PostImageGalleryState extends ConsumerState<PostImageGallery> {
  late final PageController _controller;
  late int _currentIndex;
  var _isCurrentPageZoomed = false;
  var _dismissDragOffset = 0.0;
  var _isDismissDragging = false;
  final Set<int> _activePointers = <int>{};
  int? _dismissPointer;
  Offset? _dismissStartPosition;

  @override
  void initState() {
    super.initState();
    _currentIndex = widget.initialIndex.clamp(
      0,
      widget.galleryImages.length - 1,
    );
    _controller = PageController(initialPage: _currentIndex);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final viewPadding = MediaQuery.of(context).viewPadding;
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>() ?? const SpacingTheme();
    final radii = theme.extension<RadiusTheme>() ?? const RadiusTheme();
    final current = widget.galleryImages[_currentIndex];
    final hasMultipleImages = widget.galleryImages.length > 1;
    final height = MediaQuery.sizeOf(context).height;
    final opacity = (1 - (_dismissDragOffset.abs() / height)).clamp(0.35, 1.0);
    final dragAnimationDuration = _isDismissDragging
        ? Duration.zero
        : const Duration(milliseconds: 180);
    final content = Stack(
      children: [
        PageView.builder(
          key: const Key('post-image-gallery-page-view'),
          controller: _controller,
          physics: _isCurrentPageZoomed
              ? const NeverScrollableScrollPhysics()
              : const PageScrollPhysics(),
          itemCount: widget.galleryImages.length,
          onPageChanged: (value) => setState(() {
            _currentIndex = value;
            _isCurrentPageZoomed = false;
          }),
          itemBuilder: (context, index) {
            final image = widget.galleryImages[index];
            final url = image.fullsize ?? image.thumb;
            return _ZoomableGalleryImage(
              image: image,
              imageUrl: url,
              onZoomChanged: index == _currentIndex
                  ? (isZoomed) => _setCurrentPageZoomed(isZoomed)
                  : null,
            );
          },
        ),
        if (hasMultipleImages)
          Positioned(
            right: viewPadding.right + spacing.sp4,
            top: viewPadding.top + spacing.sp4,
            child: DecoratedBox(
              key: const Key('post-image-gallery-count'),
              decoration: BoxDecoration(
                color: Colors.black.withValues(alpha: 0.45),
                borderRadius: BorderRadius.circular(radii.r2),
              ),
              child: Padding(
                padding: EdgeInsets.symmetric(
                  horizontal: spacing.sp3,
                  vertical: spacing.sp1,
                ),
                child: Text(
                  '${_currentIndex + 1}/${widget.galleryImages.length}',
                  style: const TextStyle(color: Colors.white, fontSize: 13),
                ),
              ),
            ),
          ),
        Positioned(
          left: 0,
          right: 0,
          bottom: 0,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (hasMultipleImages) ...[
                PostImagePageIndicator(
                  indicatorKey: const Key('post-image-gallery-dots'),
                  controller: _controller,
                  count: widget.galleryImages.length,
                ),
                SizedBox(height: spacing.sp2),
              ],
              DecoratedBox(
                decoration: BoxDecoration(
                  color: Colors.black.withValues(alpha: 0.55),
                ),
                child: Padding(
                  key: const Key('post-image-gallery-alt-text-padding'),
                  padding: EdgeInsets.fromLTRB(
                    viewPadding.left + spacing.sp3,
                    spacing.sp3,
                    viewPadding.right + spacing.sp3,
                    viewPadding.bottom + spacing.sp3,
                  ),
                  child: SizedBox(
                    width: double.infinity,
                    child: Text(
                      current.alt,
                      style: const TextStyle(color: Colors.white),
                    ),
                  ),
                ),
              ),
            ],
          ),
        ),
      ],
    );

    return Listener(
      behavior: HitTestBehavior.translucent,
      onPointerDown: (event) => _handlePointerDown(event),
      onPointerMove: (event) => _handlePointerMove(event, height),
      onPointerUp: (_) => _handlePointerEnd(context, height),
      onPointerCancel: (_) => _cancelDismissDrag(),
      child: AnimatedOpacity(
        duration: dragAnimationDuration,
        curve: Curves.easeOutCubic,
        opacity: opacity,
        child: AnimatedContainer(
          duration: dragAnimationDuration,
          curve: Curves.easeOutCubic,
          transform: Matrix4.translationValues(0, _dismissDragOffset, 0),
          child: content,
        ),
      ),
    );
  }

  void _setCurrentPageZoomed(bool isZoomed) {
    if (_isCurrentPageZoomed == isZoomed || !mounted) return;
    setState(() => _isCurrentPageZoomed = isZoomed);
  }

  void _handlePointerDown(PointerDownEvent event) {
    _activePointers.add(event.pointer);
    if (_isCurrentPageZoomed || _activePointers.length != 1) return;
    _dismissPointer = event.pointer;
    _dismissStartPosition = event.position;
  }

  void _handlePointerMove(PointerMoveEvent event, double height) {
    if (_isCurrentPageZoomed || event.pointer != _dismissPointer) return;
    final start = _dismissStartPosition;
    if (start == null || _activePointers.length != 1) return;

    final delta = event.position - start;
    final isDownwardDismiss = delta.dy > 0 && delta.dy > delta.dx.abs();
    if (!isDownwardDismiss && !_isDismissDragging) return;

    setState(() {
      _isDismissDragging = true;
      _dismissDragOffset = delta.dy.clamp(0.0, height);
    });
  }

  void _handlePointerEnd(BuildContext context, double height) {
    final shouldDismiss = _dismissDragOffset > height * 0.18;
    _activePointers.clear();
    _dismissPointer = null;
    _dismissStartPosition = null;

    if (shouldDismiss) {
      Navigator.of(context).maybePop();
      return;
    }
    _cancelDismissDrag();
  }

  void _cancelDismissDrag() {
    _activePointers.clear();
    _dismissPointer = null;
    _dismissStartPosition = null;
    setState(() {
      _isDismissDragging = false;
      _dismissDragOffset = 0;
    });
  }
}

class _ZoomableGalleryImage extends ConsumerStatefulWidget {
  const _ZoomableGalleryImage({
    required this.image,
    required this.imageUrl,
    required this.onZoomChanged,
  });

  final GalleryImage image;
  final String? imageUrl;
  final ValueChanged<bool>? onZoomChanged;

  @override
  ConsumerState<_ZoomableGalleryImage> createState() =>
      _ZoomableGalleryImageState();
}

class _ZoomableGalleryImageState extends ConsumerState<_ZoomableGalleryImage> {
  late final TransformationController _transformationController;
  var _isZoomed = false;

  @override
  void initState() {
    super.initState();
    _transformationController = TransformationController()
      ..addListener(_handleTransformChanged);
  }

  @override
  void dispose() {
    _transformationController
      ..removeListener(_handleTransformChanged)
      ..dispose();
    super.dispose();
  }

  void _handleTransformChanged() {
    final scale = _transformationController.value.getMaxScaleOnAxis();
    final isZoomed = scale > _galleryMinScale + _zoomedScaleEpsilon;
    if (_isZoomed == isZoomed) return;
    _isZoomed = isZoomed;
    widget.onZoomChanged?.call(isZoomed);
  }

  @override
  Widget build(BuildContext context) {
    return InteractiveViewer(
      minScale: _galleryMinScale,
      maxScale: _galleryMaxScale,
      panEnabled: true,
      transformationController: _transformationController,
      child: Semantics(
        label: widget.image.alt,
        child: widget.imageUrl == null
            ? const DecoratedBox(
                decoration: BoxDecoration(color: Color(0xFF111111)),
              )
            : CachedNetworkImage(
                imageUrl: widget.imageUrl!,
                cacheManager: ref.watch(feedImageCacheManagerProvider),
                fit: BoxFit.contain,
                progressIndicatorBuilder: (context, _, progress) => Center(
                  child: CircularProgressIndicator(
                    key: const Key('post-image-gallery-loading'),
                    value: progress.progress,
                    color: Colors.white,
                  ),
                ),
              ),
      ),
    );
  }
}
