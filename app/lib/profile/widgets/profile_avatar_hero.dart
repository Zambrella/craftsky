import 'dart:async';

import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:material_ui/material_ui.dart';

/// In-memory navigation context for one particular avatar on the source page.
/// A per-source tag lets a feed contain several avatars for the same person.
class ProfileAvatarTransition {
  new({required this.tag, required this.avatar});

  final Object tag;
  final ProfileAvatar avatar;

  // Preserve the placeholder when a response replaces the loading header
  // during the flight. This key belongs only to the destination route.
  final GlobalKey destinationKey = GlobalKey(debugLabel: 'profile-avatar-hero');
}

/// Supplies the destination tag to both the loading and loaded profile header.
class ProfileAvatarHeroDestination extends InheritedWidget {
  const new({required this.transition, required super.child, super.key});

  final ProfileAvatarTransition? transition;

  static ProfileAvatarTransition? maybeOf(BuildContext context) => context
      .dependOnInheritedWidgetOfExactType<ProfileAvatarHeroDestination>()
      ?.transition;

  @override
  bool updateShouldNotify(ProfileAvatarHeroDestination oldWidget) =>
      transition != oldWidget.transition;
}

/// Connects an avatar and its surrounding tap targets to the same hero flight.
class ProfileAvatarHeroSource extends StatefulWidget {
  const new({
    required this.did,
    required this.avatar,
    required this.builder,
    this.beforeOpen,
    super.key,
  });

  final Future<void> Function()? beforeOpen;
  final Did did;
  final ProfileAvatar avatar;
  final Widget Function(BuildContext, Widget avatar, VoidCallback openProfile)
  builder;

  @override
  State<ProfileAvatarHeroSource> createState() =>
      _ProfileAvatarHeroSourceState();
}

class _ProfileAvatarHeroSourceState extends State<ProfileAvatarHeroSource> {
  Object _tag = Object();

  @override
  void didUpdateWidget(ProfileAvatarHeroSource oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.did != widget.did) _tag = Object();
  }

  @override
  Widget build(BuildContext context) => widget.builder(
    context,
    ProfileAvatarHero(tag: _tag, child: widget.avatar),
    () => unawaited(_openProfile(context)),
  );

  Future<void> _openProfile(BuildContext context) async {
    final did = widget.did;
    final avatar = widget.avatar;
    final tag = _tag;
    await widget.beforeOpen?.call();
    if (!context.mounted) return;
    await UserProfileRoute(
      did: did,
      $extra: MediaQuery.disableAnimationsOf(context)
          ? null
          : ProfileAvatarTransition(tag: tag, avatar: avatar),
    ).push<void>(context);
  }
}

/// Scales the whole circular avatar during flight, including its border and
/// fallback initial, rather than stretching its fixed-size contents.
class ProfileAvatarHero extends StatelessWidget {
  const new({required this.tag, required this.child, this.heroKey, super.key});

  final Object tag;
  final Widget child;
  final GlobalKey? heroKey;

  @override
  Widget build(BuildContext context) => HeroMode(
    enabled: !MediaQuery.disableAnimationsOf(context),
    child: Hero(
      key: heroKey,
      tag: tag,
      transitionOnUserGestures: true,
      flightShuttleBuilder: (context, animation, direction, from, to) {
        final hero =
            (direction == HeroFlightDirection.push ? to : from).widget as Hero;
        return Material(
          key: const Key('profile-avatar-flight'),
          type: MaterialType.transparency,
          child: FittedBox(child: hero.child),
        );
      },
      child: child,
    ),
  );
}
