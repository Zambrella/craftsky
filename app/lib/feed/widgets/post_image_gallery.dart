import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class PostImageGallery extends ConsumerStatefulWidget {
  const PostImageGallery({
    required this.images,
    this.initialIndex = 0,
    super.key,
  });

  final List<PostImage> images;
  final int initialIndex;

  @override
  ConsumerState<PostImageGallery> createState() => _PostImageGalleryState();
}

class _PostImageGalleryState extends ConsumerState<PostImageGallery> {
  late final PageController _controller;
  late int _currentIndex;

  @override
  void initState() {
    super.initState();
    _currentIndex = widget.initialIndex.clamp(0, widget.images.length - 1);
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
    final current = widget.images[_currentIndex];
    final hasMultipleImages = widget.images.length > 1;
    return Stack(
      children: [
        PageView.builder(
          key: const Key('post-image-gallery-page-view'),
          controller: _controller,
          itemCount: widget.images.length,
          onPageChanged: (value) => setState(() => _currentIndex = value),
          itemBuilder: (context, index) {
            final image = widget.images[index];
            final url = image.fullsize ?? image.thumb;
            if (url == null) {
              return InteractiveViewer(
                minScale: 1,
                maxScale: 4,
                panEnabled: false,
                child: Semantics(
                  label: image.alt,
                  child: const DecoratedBox(
                    decoration: BoxDecoration(color: Color(0xFF111111)),
                  ),
                ),
              );
            }
            return InteractiveViewer(
              minScale: 1,
              maxScale: 4,
              panEnabled: false,
              child: Semantics(
                label: image.alt,
                child: CachedNetworkImage(
                  imageUrl: url,
                  cacheManager: ref.watch(feedImageCacheManagerProvider),
                  fit: BoxFit.contain,
                ),
              ),
            );
          },
        ),
        if (hasMultipleImages)
          Positioned(
            right: viewPadding.right + 16,
            top: viewPadding.top + 16,
            child: DecoratedBox(
              key: const Key('post-image-gallery-count'),
              decoration: BoxDecoration(
                color: Colors.black.withValues(alpha: 0.45),
                borderRadius: BorderRadius.circular(10),
              ),
              child: Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: 10,
                  vertical: 4,
                ),
                child: Text(
                  '${_currentIndex + 1}/${widget.images.length}',
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
                Row(
                  key: const Key('post-image-gallery-dots'),
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: List.generate(widget.images.length, (index) {
                    final isActive = index == _currentIndex;
                    return Container(
                      width: 6,
                      height: 6,
                      margin: const EdgeInsets.symmetric(horizontal: 2),
                      decoration: BoxDecoration(
                        color: (isActive ? Colors.white : Colors.white70)
                            .withValues(alpha: 0.95),
                        shape: BoxShape.circle,
                      ),
                    );
                  }),
                ),
                const SizedBox(height: 8),
              ],
              DecoratedBox(
                decoration: BoxDecoration(
                  color: Colors.black.withValues(alpha: 0.55),
                ),
                child: Padding(
                  key: const Key('post-image-gallery-alt-text-padding'),
                  padding: EdgeInsets.fromLTRB(
                    viewPadding.left + 12,
                    12,
                    viewPadding.right + 12,
                    viewPadding.bottom + 12,
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
  }
}
