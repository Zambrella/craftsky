import 'dart:async';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_image_page_indicator.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pinch_zoom/pinch_zoom.dart';

const _defaultFallbackHeight = 320.0;
const _defaultMinHeight = 160.0;
const _defaultMaxHeight = 420.0;

typedef PostImageTapCallback = void Function(int index, List<Object> heroTags);

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
    this.onImageDoubleTap,
    super.key,
  });

  final List<PostImage> images;
  final PostImageTapCallback? onImageTap;
  final VoidCallback? onImageDoubleTap;

  @override
  ConsumerState<PostImageCarousel> createState() => _PostImageCarouselState();
}

class _PostImageCarouselState extends ConsumerState<PostImageCarousel> {
  final _pageController = PageController(keepPage: false);
  Timer? _singleTapTimer;
  int? _pointer;
  Offset? _downPosition;
  Offset? _lastTapPosition;
  Duration? _lastTapTime;
  var _isTap = false;
  var _page = 0;
  late List<Object> _heroTags;

  @override
  void initState() {
    super.initState();
    _heroTags = List<Object>.generate(widget.images.length, (_) => Object());
  }

  @override
  void didUpdateWidget(covariant PostImageCarousel oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.images.length == widget.images.length) return;
    _heroTags = List<Object>.generate(
      widget.images.length,
      (index) => index < _heroTags.length ? _heroTags[index] : Object(),
    );
    _page = _page.clamp(0, widget.images.length - 1);
  }

  @override
  void dispose() {
    _singleTapTimer?.cancel();
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

        return Listener(
          behavior: HitTestBehavior.opaque,
          onPointerDown: _handlePointerDown,
          onPointerMove: _handlePointerMove,
          onPointerUp: _handlePointerUp,
          onPointerCancel: (_) => _resetPointer(),
          child: Stack(
            key: const Key('post-image-carousel'),
            children: [
              DecoratedBox(
                position: DecorationPosition.foreground,
                decoration: const BoxDecoration(
                  border: Border.fromBorderSide(BorderSide()),
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
                      if (url == null) {
                        final child = PinchZoom(
                          maxScale: 4,
                          child: Semantics(
                            label: image.alt,
                            child: DecoratedBox(
                              decoration: BoxDecoration(
                                color:
                                    Theme.of(context).brightness ==
                                        Brightness.dark
                                    ? Theme.of(
                                        context,
                                      ).colorScheme.surfaceContainerHighest
                                    : const Color(0xFFEAEAEA),
                              ),
                            ),
                          ),
                        );

                        return Hero(tag: _heroTags[index], child: child);
                      }

                      final child = PinchZoom(
                        maxScale: 4,
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

                      return Hero(tag: _heroTags[index], child: child);
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
                        style: const TextStyle(
                          color: Colors.white,
                          fontSize: 12,
                        ),
                      ),
                    ),
                  ),
                ),
              if (widget.images.length > 1)
                Positioned(
                  left: 0,
                  right: 0,
                  bottom: 8,
                  child: PostImagePageIndicator(
                    indicatorKey: const Key('post-image-dots'),
                    controller: _pageController,
                    count: widget.images.length,
                  ),
                ),
            ],
          ),
        );
      },
    );
  }

  void _handlePointerDown(PointerDownEvent event) {
    if (_pointer != null) {
      _isTap = false;
      return;
    }
    _pointer = event.pointer;
    _downPosition = event.localPosition;
    _isTap = true;
  }

  void _handlePointerMove(PointerMoveEvent event) {
    if (event.pointer != _pointer || !_isTap) return;
    if ((event.localPosition - _downPosition!).distance > kTouchSlop) {
      _isTap = false;
    }
  }

  void _handlePointerUp(PointerUpEvent event) {
    if (event.pointer != _pointer) return;
    final isTap = _isTap;
    _resetPointer();
    if (!isTap) return;

    final lastTapTime = _lastTapTime;
    final lastTapPosition = _lastTapPosition;
    final isDoubleTap =
        lastTapTime != null &&
        event.timeStamp - lastTapTime <= kDoubleTapTimeout &&
        lastTapPosition != null &&
        (event.localPosition - lastTapPosition).distance <= kDoubleTapSlop;
    if (isDoubleTap) {
      _singleTapTimer?.cancel();
      _lastTapTime = null;
      _lastTapPosition = null;
      widget.onImageDoubleTap?.call();
      return;
    }

    _lastTapTime = event.timeStamp;
    _lastTapPosition = event.localPosition;
    _singleTapTimer?.cancel();
    _singleTapTimer = Timer(kDoubleTapTimeout, () {
      _lastTapTime = null;
      _lastTapPosition = null;
      widget.onImageTap?.call(
        _page,
        List<Object>.unmodifiable(_heroTags),
      );
    });
  }

  void _resetPointer() {
    _pointer = null;
    _downPosition = null;
    _isTap = false;
  }
}
