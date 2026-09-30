import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';

/// Returns the navigator that should present a full-screen modal.
///
/// Compact modals continue to cover the entire application, including bottom
/// navigation. Large-screen modals stay within the content navigator so the
/// persistent navigation rail remains visible.
NavigatorState responsiveModalNavigator(BuildContext context) => Navigator.of(
  context,
  rootNavigator: FormFactor.fromWidth(MediaQuery.sizeOf(context).width).isSmall,
);

/// Full-screen task route: Android's default Material transition is
/// horizontal, so use the same bottom-up modal presentation on both platforms.
class FullscreenModalRoute<T> extends MaterialPageRoute<T> {
  FullscreenModalRoute({required super.builder})
    : super(fullscreenDialog: true);

  @override
  Widget buildTransitions(
    BuildContext context,
    Animation<double> animation,
    Animation<double> secondaryAnimation,
    Widget child,
  ) {
    if (Theme.of(context).platform != TargetPlatform.android) {
      return super.buildTransitions(
        context,
        animation,
        secondaryAnimation,
        child,
      );
    }
    return SlideTransition(
      position: Tween<Offset>(
        begin: const Offset(0, 1),
        end: Offset.zero,
      ).chain(CurveTween(curve: Curves.easeInOutCubic)).animate(animation),
      child: child,
    );
  }
}
