import 'package:craftsky_app/feed/models/post.dart';
import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const _defaultFallbackHeight = 320.0;
const _defaultMinHeight = 160.0;
const _defaultMaxHeight = 420.0;

double computeBoundedImageHeight({
  required double availableWidth,
  required PostImageAspectRatio? aspectRatio,
  double minHeight = _defaultMinHeight,
  double maxHeight = _defaultMaxHeight,
  double fallbackHeight = _defaultFallbackHeight,
}) {
  if (aspectRatio == null ||
      aspectRatio.width <= 0 ||
      aspectRatio.height <= 0) {
    return fallbackHeight.clamp(minHeight, maxHeight);
  }

  final ratio = aspectRatio.width / aspectRatio.height;
  final rawHeight = availableWidth / ratio;
  return rawHeight.clamp(minHeight, maxHeight);
}

class PostImageCarousel extends ConsumerStatefulWidget {
  const PostImageCarousel({
    required this.images,
    this.onImageTap,
    this.heroTagBuilder,
    super.key,
  });

  final List<PostImage> images;
  final ValueChanged<int>? onImageTap;
  final String Function(int index)? heroTagBuilder;

  @override
  ConsumerState<PostImageCarousel> createState() => _PostImageCarouselState();
}

class _PostImageCarouselState extends ConsumerState<PostImageCarousel> {
  final _pageController = PageController();
  var _page = 0;

  @override
  void dispose() {
    _pageController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final current = widget.images[_page.clamp(0, widget.images.length - 1)];
        final height = computeBoundedImageHeight(
          availableWidth: constraints.maxWidth,
          aspectRatio: current.aspectRatio,
        );

        return Stack(
          key: const Key('post-image-carousel'),
          children: [
            DecoratedBox(
              position: DecorationPosition.foreground,
              decoration: const BoxDecoration(
                border: Border.fromBorderSide(BorderSide(color: Colors.black)),
              ),
              child: SizedBox(
                height: height,
                child: PageView.builder(
                  controller: _pageController,
                  itemCount: widget.images.length,
                  onPageChanged: (value) => setState(() => _page = value),
                  itemBuilder: (context, index) {
                    final image = widget.images[index];
                    final url = image.thumb ?? image.fullsize;
                    final heroTag = widget.heroTagBuilder?.call(index);
                    if (url == null) {
                      final child = InteractiveViewer(
                        minScale: 1,
                        maxScale: 4,
                        panEnabled: false,
                        child: Semantics(
                          label: image.alt,
                          child: const DecoratedBox(
                            decoration: BoxDecoration(
                              color: Color(0xFFEAEAEA),
                            ),
                          ),
                        ),
                      );

                      return GestureDetector(
                        behavior: HitTestBehavior.opaque,
                        onTap: () => widget.onImageTap?.call(index),
                        child: heroTag == null
                            ? child
                            : Hero(tag: heroTag, child: child),
                      );
                    }

                    final child = InteractiveViewer(
                      minScale: 1,
                      maxScale: 4,
                      panEnabled: false,
                      child: Semantics(
                        label: image.alt,
                        child: CachedNetworkImage(
                          imageUrl: url,
                          cacheManager: ref.watch(
                            feedImageCacheManagerProvider,
                          ),
                          fit: BoxFit.cover,
                          width: double.infinity,
                          height: height,
                        ),
                      ),
                    );

                    return GestureDetector(
                      behavior: HitTestBehavior.opaque,
                      onTap: () => widget.onImageTap?.call(index),
                      child: heroTag == null
                          ? child
                          : Hero(tag: heroTag, child: child),
                    );
                  },
                ),
              ),
            ),
            if (widget.images.length > 1)
              Positioned(
                right: 8,
                top: 8,
                child: DecoratedBox(
                  key: const Key('post-image-count'),
                  decoration: BoxDecoration(
                    color: Colors.black.withValues(alpha: 0.45),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Padding(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 8,
                      vertical: 2,
                    ),
                    child: Text(
                      '${_page + 1}/${widget.images.length}',
                      style: const TextStyle(color: Colors.white, fontSize: 12),
                    ),
                  ),
                ),
              ),
            if (widget.images.length > 1)
              Positioned(
                left: 0,
                right: 0,
                bottom: 8,
                child: Row(
                  key: const Key('post-image-dots'),
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: List.generate(widget.images.length, (index) {
                    final isActive = index == _page;
                    return Container(
                      width: 6,
                      height: 6,
                      margin: const EdgeInsets.symmetric(horizontal: 2),
                      decoration: BoxDecoration(
                        color: (isActive ? Colors.white : Colors.white70)
                            .withValues(
                              alpha: 0.95,
                            ),
                        shape: BoxShape.circle,
                      ),
                    );
                  }),
                ),
              ),
          ],
        );
      },
    );
  }
}
