import 'dart:async';
import 'dart:math' as math;

import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:cupertino_ui/cupertino_ui.dart'
    show CupertinoSliverRefreshControl, RefreshIndicatorMode;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';
import 'package:rive/rive.dart' as rive;

final FutureProvider<rive.File> yarnRefreshFileProvider =
    FutureProvider.autoDispose<rive.File>((ref) async {
      rive.File? file;
      ref.onDispose(() => file?.dispose());
      await rive.RiveNative.init();
      final loaded = await rive.File.asset(
        'assets/animations/yarn_refresh.riv',
        // Keep the Rive renderer on devices (including the iOS rendering fix).
        // Headless widget tests have no native render context; passing its
        // unsupported factory into File.asset aborts the process in C++.
        riveFactory: rive.Factory.rive.isSupported
            ? rive.Factory.rive
            : rive.Factory.flutter,
      );
      if (loaded == null) {
        throw const FormatException('Could not decode yarn refresh animation');
      }
      if (ref.mounted) {
        file = loaded;
      } else {
        loaded.dispose();
      }
      return loaded;
    }, name: 'yarnRefreshFileProvider');

/// A refresh sliver reserves real space between the header and scrolling rows.
/// Bouncing physics exposes it on every platform, even for empty lists.
class YarnRefreshScrollView extends ConsumerWidget {
  const new({
    required this.onRefresh,
    required this.slivers,
    this.controller,
    this.scrollViewKey,
    this.headerSliverCount = 0,
    this.notificationPredicate = defaultScrollNotificationPredicate,
    this.showSuccess = true,
    super.key,
  }) : children = null,
       padding = null;

  const new list({
    required this.onRefresh,
    required this.children,
    this.padding,
    this.controller,
    this.scrollViewKey,
    this.notificationPredicate = defaultScrollNotificationPredicate,
    this.showSuccess = true,
    super.key,
  }) : slivers = const [],
       headerSliverCount = 0;

  final Future<void> Function() onRefresh;
  final List<Widget> slivers;
  final List<Widget>? children;
  final EdgeInsetsGeometry? padding;
  final ScrollController? controller;
  final Key? scrollViewKey;
  final int headerSliverCount;
  final ScrollNotificationPredicate notificationPredicate;
  // Retained-error providers can complete normally after a failed attempt.
  // Their error UI remains authoritative; those callers omit the check.
  final bool showSuccess;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final lease = ref.watch(
      sessionRegistryProvider.select((state) => state.value?.activeLease),
    );
    final rows = children;
    final content = rows == null
        ? slivers
        : <Widget>[
            SliverPadding(
              padding: padding ?? MediaQuery.paddingOf(context),
              sliver: SliverList.list(children: rows),
            ),
          ];
    return _RefreshScrollSurface(
      key: ValueKey((lease, scrollViewKey)),
      onRefresh: onRefresh,
      controller: controller,
      scrollViewKey: scrollViewKey,
      headerSliverCount: headerSliverCount,
      notificationPredicate: notificationPredicate,
      showSuccess: showSuccess,
      slivers: content,
    );
  }
}

class _RefreshScrollSurface extends ConsumerStatefulWidget {
  const new({
    required this.onRefresh,
    required this.slivers,
    required this.controller,
    required this.scrollViewKey,
    required this.headerSliverCount,
    required this.notificationPredicate,
    required this.showSuccess,
    super.key,
  });

  final Future<void> Function() onRefresh;
  final List<Widget> slivers;
  final ScrollController? controller;
  final Key? scrollViewKey;
  final int headerSliverCount;
  final ScrollNotificationPredicate notificationPredicate;
  final bool showSuccess;

  @override
  ConsumerState<_RefreshScrollSurface> createState() =>
      _RefreshScrollSurfaceState();
}

class _RefreshScrollSurfaceState extends ConsumerState<_RefreshScrollSurface> {
  static const _trigger = 112.0;
  Completer<void>? _release;
  int? _pointer;
  double _extent = 0;
  bool _allowed = true;
  bool _cancelled = false;
  bool _succeeded = false;
  bool _performingRefresh = false;
  Timer? _ackTimer;
  Completer<void>? _acknowledgement;

  void _endPointer({bool cancelled = false}) {
    _cancelled = cancelled;
    final release = _release;
    if (release != null && !release.isCompleted) release.complete();
  }

  Future<void> _refresh() async {
    final release = _release;
    if (release != null) await release.future;
    if (!mounted || _cancelled || !_allowed || _extent < _trigger) return;
    setState(() => _performingRefresh = true);
    try {
      await widget.onRefresh();
      if (!mounted) return;
      if (!widget.showSuccess) return;
      setState(() => _succeeded = true);
      // A brief acknowledgement after the real future resolves, never a fake
      // minimum loading time. The sliver still owns retraction and re-entry.
      final acknowledgement = Completer<void>();
      _acknowledgement = acknowledgement;
      _ackTimer = Timer(
        const Duration(milliseconds: 220),
        acknowledgement.complete,
      );
      await acknowledgement.future;
    } on Object {
      // Refresh callbacks retain their existing provider-owned error UI.
      // A failed refresh retracts without claiming that the feed is up to date.
      if (mounted) setState(() => _succeeded = false);
    } finally {
      _ackTimer?.cancel();
      _ackTimer = null;
      _acknowledgement = null;
      if (mounted) setState(() => _performingRefresh = false);
    }
  }

  @override
  void dispose() {
    _endPointer(cancelled: true);
    _ackTimer?.cancel();
    final acknowledgement = _acknowledgement;
    if (acknowledgement != null && !acknowledgement.isCompleted) {
      acknowledgement.complete();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final file = ref.watch(yarnRefreshFileProvider).asData?.value;
    return Listener(
      onPointerDown: (event) {
        if (_pointer != null) {
          _endPointer(cancelled: true);
          return;
        }
        _pointer = event.pointer;
        _release = Completer<void>();
        _cancelled = false;
        _succeeded = false;
      },
      onPointerUp: (event) {
        if (event.pointer != _pointer) return;
        _pointer = null;
        _endPointer(cancelled: _cancelled);
      },
      onPointerCancel: (event) {
        if (event.pointer != _pointer) return;
        _pointer = null;
        _endPointer(cancelled: true);
      },
      child: NotificationListener<ScrollNotification>(
        onNotification: (notification) {
          if (notification.depth == 0) {
            _allowed = widget.notificationPredicate(notification);
          }
          return false;
        },
        child: CustomScrollView(
          key: widget.scrollViewKey,
          controller: widget.controller,
          physics: const BouncingScrollPhysics(
            parent: AlwaysScrollableScrollPhysics(),
          ),
          slivers: [
            ...widget.slivers.take(widget.headerSliverCount),
            CupertinoSliverRefreshControl(
              refreshTriggerPullDistance: _trigger,
              // Arming must not reserve space while the finger can still
              // reverse. Reserve the loading gap only after a valid release.
              refreshIndicatorExtent: _performingRefresh ? 96 : 0,
              onRefresh: _refresh,
              builder: (context, mode, extent, trigger, heldExtent) {
                _extent = extent;
                if (!_allowed || extent <= 0) return const SizedBox.shrink();
                final loading =
                    (mode == RefreshIndicatorMode.armed ||
                        mode == RefreshIndicatorMode.refresh) &&
                    (_release?.isCompleted ?? true);
                return Semantics(
                  label: MaterialLocalizations.of(
                    context,
                  ).refreshIndicatorSemanticLabel,
                  liveRegion: loading,
                  child: IgnorePointer(
                    child: ExcludeSemantics(
                      child: SizedBox(
                        height: extent,
                        child: Center(
                          child: file == null
                              ? SizedBox.square(
                                  dimension: 32,
                                  child: _succeeded
                                      ? const Icon(Icons.check)
                                      : loading
                                      ? const StitchProgressIndicator(size: 32)
                                      : const Icon(Icons.arrow_downward),
                                )
                              : _YarnGraphic(
                                  key: const Key('yarn-refresh-graphic'),
                                  file: file,
                                  extent: extent,
                                  loading: loading && !_succeeded,
                                  succeeded: _succeeded,
                                  inkColor: Theme.of(context)
                                      .colorScheme
                                      .primary,
                                  reduced: MediaQuery.disableAnimationsOf(
                                    context,
                                  ),
                                ),
                        ),
                      ),
                    ),
                  ),
                );
              },
            ),
            ...widget.slivers.skip(widget.headerSliverCount),
          ],
        ),
      ),
    );
  }
}

class _YarnGraphic extends StatefulWidget {
  const new({
    required this.file,
    required this.extent,
    required this.loading,
    required this.succeeded,
    required this.reduced,
    required this.inkColor,
    super.key,
  });

  final rive.File file;
  final double extent;
  final bool loading;
  final bool succeeded;
  final bool reduced;
  final Color inkColor;

  @override
  State<_YarnGraphic> createState() => _YarnGraphicState();
}

@visibleForTesting
abstract interface class YarnRefreshGraphicStateForTesting {
  double get strandLength;
  double get ballScale;
  double get opacity;
  double get rotation;
  double get successOpacity;
  bool get showsSuccess;
}

class _YarnGraphicState extends State<_YarnGraphic>
    with TickerProviderStateMixin
    implements YarnRefreshGraphicStateForTesting {
  late final rive.RiveWidgetController _rive;
  late final rive.ViewModelInstance _model;
  late final rive.ViewModelInstanceColor _inkColor;
  late final Map<String, rive.ViewModelInstanceNumber> _numbers;
  late final AnimationController _turn;
  late final AnimationController _successFade;

  @override
  double get strandLength => _numbers['strandLength']!.value;
  @override
  double get ballScale => _numbers['ballScale']!.value;
  @override
  double get opacity => _numbers['yarnOpacity']!.value;
  @override
  double get rotation => _numbers['rotation']!.value;
  @override
  double get successOpacity => _numbers['checkOpacity']!.value;
  @override
  bool get showsSuccess => successOpacity == 1;

  @override
  void initState() {
    super.initState();
    _rive = rive.RiveWidgetController(widget.file)..active = false;
    _model = _rive.dataBind(rive.DataBind.auto());
    _inkColor = _model.color('inkColor')!;
    _numbers = {
      for (final name in [
        'strandLength',
        'strandFirstY',
        'strandSecondY',
        'strandHandle',
        'ballScale',
        'rotation',
        'yarnOpacity',
        'checkOpacity',
      ])
        name: _model.number(name)!,
    };
    _turn = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 900),
    )..addListener(_applyPose);
    _successFade = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 120),
      value: widget.succeeded ? 1 : 0,
    )..addListener(_applyPose);
    _updateMotion();
  }

  @override
  void didUpdateWidget(_YarnGraphic oldWidget) {
    super.didUpdateWidget(oldWidget);
    _updateMotion();
  }

  void _updateMotion() {
    final target = widget.succeeded ? 1.0 : 0.0;
    if (widget.reduced) {
      _successFade.value = target;
    } else if (_successFade.value != target) {
      _successFade.animateTo(target);
    }
    if (widget.loading && !widget.reduced) {
      if (!_turn.isAnimating) _turn.repeat();
    } else {
      _turn.stop();
    }
    _applyPose();
  }

  void _applyPose() {
    _inkColor.value = widget.inkColor;
    final pull = (widget.extent / 112).clamp(0.0, 1.0);
    _rive.artboard.height = math.max(40, widget.extent);
    final strandLength = math.max<double>(0, widget.extent - 22);
    _numbers['strandLength']!.value = strandLength;
    _numbers['strandFirstY']!.value = strandLength / 3;
    _numbers['strandSecondY']!.value = strandLength * 2 / 3;
    _numbers['strandHandle']!.value = strandLength / 6;
    _numbers['ballScale']!.value = 1;
    _numbers['rotation']!.value = widget.reduced
        ? 0
        : (pull + (widget.loading ? _turn.value : 0)) * math.pi * 2;
    _numbers['yarnOpacity']!.value =
        (widget.extent / 56).clamp(0.0, 1.0) * (1 - _successFade.value);
    _numbers['checkOpacity']!.value = _successFade.value;
    _rive.stateMachine.advanceAndApply(0);
    _rive.scheduleRepaint();
  }

  @override
  void dispose() {
    _successFade.dispose();
    _turn.dispose();
    for (final value in _numbers.values) {
      value.dispose();
    }
    _inkColor.dispose();
    _rive.dispose();
    _model.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => ClipRect(
    child: SizedBox(
      width: 80,
      height: widget.extent,
      child: rive.RiveWidget(
        controller: _rive,
        fit: rive.Fit.none,
        alignment: Alignment.topCenter,
      ),
    ),
  );
}
