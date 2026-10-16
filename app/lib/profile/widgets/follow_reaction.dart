import 'dart:async';

import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';
import 'package:rive/rive.dart' as rive;

final FutureProvider<rive.File> followAnimationFileProvider =
    FutureProvider.autoDispose<rive.File>((
      ref,
    ) async {
      rive.File? file;
      ref.onDispose(() => file?.dispose());
      await rive.RiveNative.init();
      final loaded = await rive.File.asset(
        'assets/animations/follow.riv',
        riveFactory: rive.Factory.flutter,
      );
      if (loaded == null) {
        throw const FormatException('Could not decode follow animation');
      }
      if (ref.mounted) {
        file = loaded;
      } else {
        loaded.dispose();
      }
      return loaded;
    }, name: 'followAnimationFileProvider');

/// Feedback belongs to one initiating surface and one active account lease.
/// The action returns true only for a confirmed new follow.
class FollowReactionScope extends ConsumerWidget {
  const new({required this.identity, required this.builder, super.key});

  final Object identity;
  final WidgetBuilder builder;

  static Future<void> run(
    BuildContext context,
    Future<bool> Function() action,
  ) async {
    final host = context.getInheritedWidgetOfExactType<_FollowReactionHost>();
    if (host == null) {
      await action();
    } else {
      await host.state.run(action);
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final lease = ref.watch(
      sessionRegistryProvider.select((state) => state.value?.activeLease),
    );
    return _FollowReactionSurface(
      key: ValueKey((identity, lease)),
      builder: builder,
    );
  }
}

class _FollowReactionSurface extends ConsumerStatefulWidget {
  const new({required this.builder, super.key});

  final WidgetBuilder builder;

  @override
  ConsumerState<_FollowReactionSurface> createState() =>
      _FollowReactionSurfaceState();
}

class _FollowReactionSurfaceState
    extends ConsumerState<_FollowReactionSurface> {
  final playback = ValueNotifier<_FollowPlayback?>(null);
  Timer? _timer;
  var _sequence = 0;

  Future<void> run(Future<bool> Function() action) async {
    final sequence = ++_sequence;
    _timer?.cancel();
    playback.value = null;
    final followed = await action();
    if (!mounted || sequence != _sequence || !followed) return;
    final file = ref.read(followAnimationFileProvider).asData?.value;
    // Feedback must never delay the action or start late after asset loading.
    if (file == null) return;
    final reduced = MediaQuery.disableAnimationsOf(context);
    playback.value = _FollowPlayback(
      file: file,
      reduced: reduced,
      sequence: sequence,
    );
    _timer = Timer(
      reduced
          ? const Duration(milliseconds: 400)
          : const Duration(milliseconds: 1400),
      () => playback.value = null,
    );
  }

  @override
  void dispose() {
    _timer?.cancel();
    playback.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    ref.watch(followAnimationFileProvider);
    return _FollowReactionHost(
      state: this,
      child: Builder(builder: widget.builder),
    );
  }
}

class _FollowReactionHost extends InheritedWidget {
  const new({required this.state, required super.child});

  final _FollowReactionSurfaceState state;

  @override
  bool updateShouldNotify(_FollowReactionHost oldWidget) =>
      state != oldWidget.state;
}

class _FollowPlayback {
  const new({
    required this.file,
    required this.reduced,
    required this.sequence,
  });

  final rive.File file;
  final bool reduced;
  final int sequence;
}

/// Keeps the avatar's layout, semantics and touch target unchanged.
class FollowReactionAvatar extends StatelessWidget {
  const new({required this.child, this.compact = false, super.key});

  final Widget child;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final host = context
        .dependOnInheritedWidgetOfExactType<_FollowReactionHost>();
    if (host == null) return child;
    return ValueListenableBuilder<_FollowPlayback?>(
      valueListenable: host.state.playback,
      child: child,
      builder: (context, playback, child) => Stack(
        clipBehavior: Clip.none,
        children: [
          child!,
          if (playback != null)
            Positioned.fill(
              key: const Key('follow-reaction'),
              child: IgnorePointer(
                child: ExcludeSemantics(
                  child: Center(
                    child: OverflowBox(
                      maxWidth: compact ? 112 : 160,
                      minWidth: compact ? 112 : 160,
                      maxHeight: compact ? 84 : 120,
                      minHeight: compact ? 84 : 120,
                      child: _KnotAnimation(
                        key: ValueKey(playback.sequence),
                        playback: playback,
                      ),
                    ),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _KnotAnimation extends StatefulWidget {
  const new({required this.playback, super.key});

  final _FollowPlayback playback;

  @override
  State<_KnotAnimation> createState() => _KnotAnimationState();
}

class _KnotAnimationState extends State<_KnotAnimation> {
  late final rive.RiveWidgetController _controller;

  @override
  void initState() {
    super.initState();
    _controller = rive.RiveWidgetController(widget.playback.file);
    if (widget.playback.reduced) {
      _controller.stateMachine.advanceAndApply(0.8);
      _controller.active = false;
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => TweenAnimationBuilder<double>(
    tween: Tween(begin: 0, end: 1),
    duration: widget.playback.reduced
        ? Duration.zero
        : const Duration(milliseconds: 1400),
    builder: (context, value, child) => Opacity(
      opacity: widget.playback.reduced || value < 6 / 7
          ? 1
          : ((1 - value) * 7).clamp(0.0, 1.0),
      child: child,
    ),
    child: TickerMode(
      enabled: !widget.playback.reduced,
      child: rive.RiveWidget(controller: _controller),
    ),
  );
}
