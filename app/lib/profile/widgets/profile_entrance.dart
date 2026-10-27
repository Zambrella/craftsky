import 'package:material_ui/material_ui.dart';

/// The profile details arrive in reading order, all within 800 ms.
enum ProfileEntranceStage {
  summary,
  actions,
  mutualFollowers,
  bio,
  tabs,
  content,
}

class ProfileEntrance extends StatefulWidget {
  const new({required this.child, super.key});

  final Widget child;

  static const duration = Duration(milliseconds: 800);

  @override
  State<ProfileEntrance> createState() => _ProfileEntranceState();
}

class _ProfileEntranceState extends State<ProfileEntrance>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller = AnimationController(
    vsync: this,
    duration: ProfileEntrance.duration,
  );
  bool _started = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (MediaQuery.disableAnimationsOf(context)) {
      _controller.value = 1;
      _started = true;
    } else if (!_started) {
      _started = true;
      _controller.forward();
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => _ProfileEntranceScope(
    animation: _controller,
    child: widget.child,
  );
}

class _ProfileEntranceScope extends InheritedWidget {
  const new({required this.animation, required super.child});

  final Animation<double> animation;

  @override
  bool updateShouldNotify(_ProfileEntranceScope oldWidget) =>
      animation != oldWidget.animation;
}

/// Opacity only: sections keep their layout, hit targets, and scroll positions.
class ProfileEntranceFade extends StatelessWidget {
  const new({required this.stage, required this.child, super.key});

  final ProfileEntranceStage stage;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final scope = context
        .dependOnInheritedWidgetOfExactType<_ProfileEntranceScope>();
    if (scope == null || MediaQuery.disableAnimationsOf(context)) return child;
    final start = stage.index * .1;
    return FadeTransition(
      opacity: scope.animation.drive(
        CurveTween(curve: Interval(start, start + .5, curve: Curves.easeOut)),
      ),
      child: child,
    );
  }
}
