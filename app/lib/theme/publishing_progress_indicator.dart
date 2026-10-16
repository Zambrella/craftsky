import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';
import 'package:rive/rive.dart' as rive;

/// Shared while a composer is open, so publishing need not wait for decoding.
/// The provider owns the file; each indicator owns its playback controller.
final FutureProvider<rive.File>
publishingAnimationFileProvider = FutureProvider.autoDispose<rive.File>(
  (ref) async {
    rive.File? file;
    ref.onDispose(() => file?.dispose());
    // On web, the Flutter factory needs the WASM runtime before it is read.
    await rive.RiveNative.init();
    final loaded = await rive.File.asset(
      'assets/animations/publishing.riv',
      // This vector-only asset needs no Rive GPU effects. Drawing into Flutter
      // also avoids a separate render texture for a single small indicator.
      riveFactory: rive.Factory.flutter,
    );
    if (loaded == null) {
      throw const FormatException('Could not decode publishing animation');
    }
    if (!ref.mounted) {
      // A composer can close while decoding; no widget will consume this file.
      loaded.dispose();
    } else {
      file = loaded;
    }
    return loaded;
  },
  name: 'publishingAnimationFileProvider',
);

/// Decorative publishing motion. The containing overlay announces its status.
class PublishingProgressIndicator extends ConsumerWidget {
  const new({this.size = 48, super.key});

  final double size;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final file = ref.watch(publishingAnimationFileProvider).asData?.value;
    final reduceMotion = MediaQuery.disableAnimationsOf(context);
    final threadColor = Theme.of(context).colorScheme.primary;
    return ExcludeSemantics(
      child: IgnorePointer(
        child: SizedBox.square(
          dimension: size,
          child: file == null
              ? StitchProgressIndicator(size: size)
              : _PublishingAnimation(
                  key: ValueKey((file, reduceMotion, threadColor)),
                  file: file,
                  reduceMotion: reduceMotion,
                  threadColor: threadColor,
                ),
        ),
      ),
    );
  }
}

class _PublishingAnimation extends StatefulWidget {
  const new({
    required this.file,
    required this.reduceMotion,
    required this.threadColor,
    super.key,
  });

  final rive.File file;
  final bool reduceMotion;
  final Color threadColor;

  @override
  State<_PublishingAnimation> createState() => _PublishingAnimationState();
}

class _PublishingAnimationState extends State<_PublishingAnimation> {
  late final rive.RiveWidgetController _controller;
  late final rive.ViewModelInstance _theme;
  late final rive.ViewModelInstanceColor _threadColor;

  @override
  void initState() {
    super.initState();
    _controller = rive.RiveWidgetController(widget.file);
    _theme = _controller.dataBind(rive.DataBind.auto());
    _threadColor = _theme.color('threadColor')!..value = widget.threadColor;
    if (widget.reduceMotion) {
      // Hold a readable pose with the arrow visible, rather than its fade-in.
      _controller.stateMachine.advanceAndApply(0.2);
      _controller.active = false;
    }
  }

  @override
  void dispose() {
    _threadColor.dispose();
    _controller.dispose();
    _theme.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => TickerMode(
    enabled: !widget.reduceMotion,
    child: rive.RiveWidget(controller: _controller),
  );
}
