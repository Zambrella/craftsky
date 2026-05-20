import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class PostImageGallery extends ConsumerStatefulWidget {
  const PostImageGallery({
    required this.images,
    this.initialIndex = 0,
    this.heroTagBuilder,
    super.key,
  });

  final List<PostImage> images;
  final int initialIndex;
  final String Function(int index)? heroTagBuilder;

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
    final current = widget.images[_currentIndex];
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
            final heroTag = widget.heroTagBuilder?.call(index);
            if (url == null) {
              final child = InteractiveViewer(
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
              return heroTag == null ? child : Hero(tag: heroTag, child: child);
            }
            final child = InteractiveViewer(
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
            return heroTag == null ? child : Hero(tag: heroTag, child: child);
          },
        ),
        Positioned(
          left: 0,
          right: 0,
          bottom: 0,
          child: DecoratedBox(
            decoration: BoxDecoration(
              color: Colors.black.withValues(alpha: 0.55),
            ),
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Text(
                current.alt,
                style: const TextStyle(color: Colors.white),
              ),
            ),
          ),
        ),
      ],
    );
  }
}
