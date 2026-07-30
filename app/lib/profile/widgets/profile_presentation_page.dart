import 'package:craftsky_app/profile/widgets/profile_customisation_theme.dart';
import 'package:flutter/material.dart';

/// Describes a profile route that should initially appear as a summary card.
class ProfilePresentationRequest {
  const ProfilePresentationRequest.compact({
    this.primaryColor,
    this.backgroundIllustration,
    this.avatarFrame,
  });

  bool get startsCompact => true;
  final Color? primaryColor;
  final ProfileBackgroundIllustration? backgroundIllustration;
  final ProfileAvatarFrame? avatarFrame;
}

/// A non-opaque page that can present both the compact and expanded profile.
///
/// The route stays non-opaque so the originating screen remains visible behind
/// the compact card. Its expanded child paints an ordinary full-screen
/// scaffold over that screen.
class ProfilePresentationPage extends Page<void> {
  const ProfilePresentationPage({
    required this.startsCompact,
    required this.child,
    super.key,
  });

  final bool startsCompact;
  final Widget child;

  /// Switches a compact presentation to the platform's Material page
  /// transition once its card has expanded into the full profile.
  static void markExpanded(BuildContext context) {
    final route = ModalRoute.of(context);
    if (route is _ProfilePresentationRoute) {
      route.markExpanded();
    }
  }

  @override
  Route<void> createRoute(BuildContext context) =>
      _ProfilePresentationRoute(settings: this);
}

class _ProfilePresentationRoute extends PageRoute<void>
    with MaterialRouteTransitionMixin<void> {
  _ProfilePresentationRoute({required ProfilePresentationPage settings})
    : _usesMaterialTransition = !settings.startsCompact,
      super(settings: settings);

  ProfilePresentationPage get _page => settings as ProfilePresentationPage;
  bool _usesMaterialTransition;

  void markExpanded() {
    if (_usesMaterialTransition) return;
    _usesMaterialTransition = true;
    final animationController = controller;
    if (animationController != null) {
      animationController
        ..duration = transitionDuration
        ..reverseDuration = reverseTransitionDuration;
    }
    changedInternalState();
  }

  @override
  bool get opaque => false;

  @override
  bool get barrierDismissible => false;

  @override
  Color? get barrierColor => null;

  @override
  String? get barrierLabel => null;

  @override
  bool get maintainState => true;

  @override
  DelegatedTransitionBuilder? get delegatedTransition =>
      _usesMaterialTransition ? super.delegatedTransition : null;

  @override
  bool canTransitionFrom(TransitionRoute<dynamic> previousRoute) {
    return _usesMaterialTransition && super.canTransitionFrom(previousRoute);
  }

  @override
  Duration get transitionDuration => _usesMaterialTransition
      ? super.transitionDuration
      : const Duration(milliseconds: 240);

  @override
  Duration get reverseTransitionDuration => _usesMaterialTransition
      ? super.reverseTransitionDuration
      : const Duration(milliseconds: 220);

  @override
  Widget buildContent(BuildContext context) => _page.child;

  @override
  Widget buildTransitions(
    BuildContext context,
    Animation<double> animation,
    Animation<double> secondaryAnimation,
    Widget child,
  ) {
    if (_usesMaterialTransition) {
      return super.buildTransitions(
        context,
        animation,
        secondaryAnimation,
        child,
      );
    }

    final curved = CurvedAnimation(
      parent: animation,
      curve: Curves.easeOutCubic,
      reverseCurve: Curves.easeInCubic,
    );
    return FadeTransition(opacity: curved, child: child);
  }
}
