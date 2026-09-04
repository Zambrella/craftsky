import 'dart:async';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/caption_uri_resource.dart';
import 'package:craftsky_app/feed/widgets/native_video_controller.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:visibility_detector/visibility_detector.dart';

const _playbackStartupGracePeriod = Duration(seconds: 1);

double nativeVideoAspectRatio({required int? width, required int? height}) {
  if (width == null || height == null || width <= 0 || height <= 0) {
    return 16 / 9;
  }
  return (width / height).clamp(0.5, 2.4);
}

final class NativeVideoPlayer extends StatefulWidget {
  const NativeVideoPlayer({
    required this.video,
    super.key,
    this.loadCaption,
    this.createCaptionResource = createCaptionUriResource,
    this.createAdapter = MediaKitNativeVideoAdapter.new,
    this.playbackCoordinator,
  });

  final PostVideo video;
  final Future<String> Function(String route)? loadCaption;
  final Future<CaptionUriResource> Function(String data) createCaptionResource;
  final NativeVideoViewAdapter Function() createAdapter;
  final NativeVideoPlaybackCoordinator? playbackCoordinator;

  @override
  State<NativeVideoPlayer> createState() => _NativeVideoPlayerState();
}

final class _NativeVideoPlayerState extends State<NativeVideoPlayer>
    with WidgetsBindingObserver {
  NativeVideoController? _lifecycle;
  NativeVideoViewAdapter? _adapter;
  StreamSubscription<Object>? _errorSubscription;
  StreamSubscription<bool>? _readySubscription;
  StreamSubscription<bool>? _playingSubscription;
  Timer? _failureTimer;
  Uri? _playlist;
  bool _ready = false;
  bool _failed = false;
  bool _disposed = false;
  final List<CaptionUriResource> _captionResources = [];
  List<NativeVideoCaptionTrack> _captions = const [];

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    final playlist = Uri.tryParse(widget.video.playlist ?? '');
    if (playlist == null || playlist.scheme != 'https') {
      _failed = true;
      return;
    }
    _playlist = playlist;
  }

  void _startPlayback() {
    final playlist = _playlist;
    if (playlist == null || _adapter != null || _failed) return;
    final adapter = widget.createAdapter();
    _adapter = adapter;
    _lifecycle = NativeVideoController(
      adapter,
      playbackCoordinator: widget.playbackCoordinator,
    );
    _errorSubscription = adapter.playbackErrors.listen((_) {
      if (_ready) {
        _fail();
        return;
      }
      _failureTimer ??= Timer(_playbackStartupGracePeriod, () {
        _failureTimer = null;
        if (!_ready) _fail();
      });
    });
    _readySubscription = adapter.playbackReadyChanges.listen((ready) {
      if (!ready) return;
      _ready = true;
      _failureTimer?.cancel();
      _failureTimer = null;
      if (mounted) setState(() {});
    });
    _playingSubscription = adapter.playingChanges.listen((playing) {
      if (playing) unawaited(_lifecycle?.didStartPlayback());
    });
    setState(() {});
    unawaited(_loadCaptions());
    unawaited(_open(playlist));
  }

  void _fail() {
    if (mounted) setState(() => _failed = true);
  }

  Future<void> _open(Uri playlist) async {
    try {
      await _lifecycle!.open(playlist, play: true);
    } on Object {
      _fail();
    }
  }

  Future<void> _loadCaptions() async {
    final load = widget.loadCaption;
    if (load == null) return;
    final captions = <NativeVideoCaptionTrack>[];
    final resources = <CaptionUriResource>[];
    for (final caption in widget.video.captions) {
      try {
        final resource = await widget.createCaptionResource(
          await load(caption.uri),
        );
        resources.add(resource);
        captions.add(
          NativeVideoCaptionTrack(
            language: caption.lang,
            label: caption.name,
            uri: resource.uri,
          ),
        );
      } on Object {
        // A bad optional caption must not make the video unavailable.
      }
    }
    if (_disposed) {
      await Future.wait(resources.map((resource) => resource.dispose()));
      return;
    }
    _captionResources.addAll(resources);
    if (mounted) setState(() => _captions = captions);
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state != AppLifecycleState.resumed) {
      unawaited(_lifecycle?.didEnterBackground());
    }
  }

  @override
  void dispose() {
    _disposed = true;
    WidgetsBinding.instance.removeObserver(this);
    _failureTimer?.cancel();
    unawaited(_errorSubscription?.cancel());
    unawaited(_readySubscription?.cancel());
    unawaited(_playingSubscription?.cancel());
    unawaited(_lifecycle?.dispose());
    for (final resource in _captionResources) {
      unawaited(resource.dispose());
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final radius = theme.extension<RadiusTheme>()?.r2 ?? const RadiusTheme().r2;
    final ratio = nativeVideoAspectRatio(
      width: widget.video.aspectRatio?.width,
      height: widget.video.aspectRatio?.height,
    );
    return Semantics(
      label: widget.video.alt?.trim().isNotEmpty ?? false
          ? widget.video.alt
          : 'Video',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Align(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 640),
              child: DecoratedBox(
                key: const Key('native-video-outline'),
                position: DecorationPosition.foreground,
                decoration: BoxDecoration(
                  border: Border.all(color: theme.colorScheme.outlineVariant),
                  borderRadius: BorderRadius.circular(radius),
                ),
                child: ClipRRect(
                  key: const Key('native-video-clip'),
                  borderRadius: BorderRadius.circular(radius),
                  clipBehavior: Clip.antiAlias,
                  child: AspectRatio(
                    aspectRatio: ratio,
                    child: _failed
                        ? ColoredBox(
                            color: Colors.black12,
                            child: Center(
                              child: Text(
                                AppLocalizations.of(
                                  context,
                                ).postVideoUnavailable,
                              ),
                            ),
                          )
                        : Stack(
                            fit: StackFit.expand,
                            children: [
                              if (_adapter case final adapter?)
                                VisibilityDetector(
                                  key: ValueKey(
                                    'native-video-${widget.video.cid}',
                                  ),
                                  onVisibilityChanged: (info) => unawaited(
                                    _lifecycle?.setVisible(
                                      visible: info.visibleFraction > 0.5,
                                    ),
                                  ),
                                  child: adapter.buildView(aspectRatio: ratio),
                                ),
                              if (!_ready)
                                _NativeVideoThumbnail(
                                  thumbnail: widget.video.thumbnail,
                                  loading: _adapter != null,
                                  onPlay: _startPlayback,
                                ),
                            ],
                          ),
                  ),
                ),
              ),
            ),
          ),
          if (widget.video.alt?.trim().isNotEmpty ?? false)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(widget.video.alt!),
            ),
          if (_captions.isNotEmpty)
            Align(
              alignment: AlignmentDirectional.centerEnd,
              child: PopupMenuButton<int>(
                key: const Key('native-video-caption-menu'),
                tooltip: AppLocalizations.of(context).postVideoCaptions,
                icon: const Icon(Icons.closed_caption_outlined),
                onSelected: (index) => unawaited(
                  _lifecycle?.selectCaption(
                    index < 0 ? null : _captions[index],
                  ),
                ),
                itemBuilder: (context) => [
                  PopupMenuItem<int>(
                    value: -1,
                    child: Text(
                      AppLocalizations.of(context).postVideoCaptionsOff,
                    ),
                  ),
                  for (var i = 0; i < _captions.length; i++)
                    PopupMenuItem<int>(
                      value: i,
                      child: Text(_captions[i].label),
                    ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}

class _NativeVideoThumbnail extends ConsumerWidget {
  const _NativeVideoThumbnail({
    required this.thumbnail,
    required this.loading,
    required this.onPlay,
  });

  final String? thumbnail;
  final bool loading;
  final VoidCallback onPlay;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final uri = Uri.tryParse(thumbnail ?? '');
    return ColoredBox(
      key: const Key('native-video-thumbnail'),
      color: Colors.black12,
      child: Stack(
        fit: StackFit.expand,
        children: [
          if (uri != null && uri.scheme == 'https')
            CachedNetworkImage(
              imageUrl: uri.toString(),
              cacheManager: ref.watch(feedImageCacheManagerProvider),
              fit: BoxFit.cover,
            ),
          Center(
            child: loading
                ? const CircularProgressIndicator()
                : IconButton.filledTonal(
                    key: const Key('native-video-thumbnail-play'),
                    tooltip: AppLocalizations.of(context).nativeVideoPlay,
                    onPressed: onPlay,
                    icon: const Icon(Icons.play_arrow),
                  ),
          ),
        ],
      ),
    );
  }
}
