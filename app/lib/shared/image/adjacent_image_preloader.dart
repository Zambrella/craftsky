import 'dart:async';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';

/// Warms only the immediate neighbors, after the visible image has decoded.
class AdjacentImagePreloader extends StatefulWidget {
  const new({
    required this.urls,
    required this.currentIndex,
    required this.cacheManager,
    required this.child,
    super.key,
  });

  final List<String?> urls;
  final int currentIndex;
  final BaseCacheManager cacheManager;
  final Widget child;

  @override
  State<AdjacentImagePreloader> createState() => _AdjacentImagePreloaderState();
}

class _AdjacentImagePreloaderState extends State<AdjacentImagePreloader> {
  ImageStream? _stream;
  ImageStreamListener? _listener;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _watchCurrentImage();
  }

  @override
  void didUpdateWidget(covariant AdjacentImagePreloader oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.currentIndex != widget.currentIndex ||
        oldWidget.cacheManager != widget.cacheManager ||
        !listEquals(oldWidget.urls, widget.urls)) {
      _watchCurrentImage();
    }
  }

  void _stopWatching() {
    if (_listener case final listener?) {
      _stream?.removeListener(listener);
    }
    _stream = null;
    _listener = null;
  }

  void _watchCurrentImage() {
    _stopWatching();
    if (widget.urls.length < 2) return;
    final index = widget.currentIndex;
    final url = widget.urls[index];
    if (url == null) return;

    final stream = _provider(url)
        .resolve(createLocalImageConfiguration(context));
    var loaded = false;
    _stream = stream;
    _listener = ImageStreamListener((image, _) {
      image.dispose();
      if (loaded || !mounted || _stream != stream) return;
      loaded = true;
      for (final neighbor in [index - 1, index + 1]) {
        if (neighbor < 0 || neighbor >= widget.urls.length) continue;
        final neighborUrl = widget.urls[neighbor];
        if (neighborUrl == null) continue;
        // Best effort: the displayed image owns user-facing load errors.
        unawaited(
          precacheImage(_provider(neighborUrl), context, onError: (_, _) {}),
        );
      }
    }, onError: (_, _) {});
    stream.addListener(_listener!);
  }

  CachedNetworkImageProvider _provider(String url) =>
      CachedNetworkImageProvider(url, cacheManager: widget.cacheManager);

  @override
  void dispose() {
    _stopWatching();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => widget.child;
}
