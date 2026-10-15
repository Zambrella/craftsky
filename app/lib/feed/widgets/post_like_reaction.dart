import 'dart:async';

import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';
import 'package:rive/rive.dart' as rive;

/// Decoded once for the mounted post cards, before a like needs to play it.
final FutureProvider<rive.File> likeAnimationFileProvider =
    FutureProvider.autoDispose<rive.File>((ref) async {
      rive.File? file;
      ref.onDispose(() => file?.dispose());
      await rive.RiveNative.init();
      final loaded = await rive.File.asset(
        'assets/animations/like.riv',
        riveFactory: rive.Factory.flutter,
      );
      if (loaded == null) {
        throw const FormatException('Could not decode like animation');
      }
      if (ref.mounted) {
        file = loaded;
      } else {
        loaded.dispose();
      }
      return loaded;
    }, name: 'likeAnimationFileProvider');

/// Local feedback for the initiating card, independent of mutation completion.
/// The caller's key identifies both the post and the active account session.
class PostLikeReaction extends ConsumerStatefulWidget {
  const new({
    required this.liked,
    required this.onLike,
    required this.builder,
    this.cancelled = false,
    super.key,
  });

  final bool liked;
  final bool cancelled;
  final VoidCallback? onLike;
  final Widget Function(BuildContext context, VoidCallback? onLike) builder;

  @override
  ConsumerState<PostLikeReaction> createState() => _PostLikeReactionState();
}

class _PostLikeReactionState extends ConsumerState<PostLikeReaction> {
  Timer? _timer;
  var _visible = false;
  var _sequence = 0;
  rive.File? _file;

  @override
  void didUpdateWidget(PostLikeReaction oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.cancelled || (oldWidget.liked && !widget.liked)) {
      _timer?.cancel();
      _visible = false;
    }
  }

  void _like() {
    final wasLiked = widget.liked;
    widget.onLike?.call();
    if (!mounted) return;
    _timer?.cancel();
    if (wasLiked) {
      setState(() => _visible = false);
      return;
    }
    // Capture readiness for this playback. Loading must never delay a like or
    // cause an animation to begin late halfway through the feedback window.
    _file = ref.read(likeAnimationFileProvider).asData?.value;
    setState(() {
      _sequence++;
      _visible = true;
    });
    final reduced = MediaQuery.disableAnimationsOf(context);
    _timer = Timer(
      reduced
          ? const Duration(milliseconds: 400)
          : const Duration(milliseconds: 1400),
      () {
        if (mounted) setState(() => _visible = false);
      },
    );
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    ref.watch(likeAnimationFileProvider);
    final reduced = MediaQuery.disableAnimationsOf(context);
    return Stack(
      children: [
        widget.builder(context, widget.onLike == null ? null : _like),
        if (_visible)
          Positioned.fill(
            key: const Key('post-like-reaction'),
            child: IgnorePointer(
              child: ExcludeSemantics(
                child: Center(
                  child: SizedBox.square(
                    dimension: 160,
                    child: _file == null
                        ? const Center(
                            child: Icon(
                              CraftskyIcons.liked,
                              color: BrandColors.red,
                              size: 72,
                            ),
                          )
                        : _LikeAnimation(
                            key: ValueKey((_sequence, reduced)),
                            file: _file!,
                            reduced: reduced,
                          ),
                  ),
                ),
              ),
            ),
          ),
      ],
    );
  }
}

class _LikeAnimation extends StatefulWidget {
  const new({required this.file, required this.reduced, super.key});

  final rive.File file;
  final bool reduced;

  @override
  State<_LikeAnimation> createState() => _LikeAnimationState();
}

class _LikeAnimationState extends State<_LikeAnimation> {
  late final rive.RiveWidgetController _controller;

  @override
  void initState() {
    super.initState();
    _controller = rive.RiveWidgetController(widget.file);
    if (widget.reduced) {
      _controller.stateMachine.advanceAndApply(0.65);
      _controller.active = false;
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => TickerMode(
    enabled: !widget.reduced,
    child: rive.RiveWidget(controller: _controller),
  );
}
