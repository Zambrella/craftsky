import 'package:craftsky_app/profile/widgets/profile_avatar_hero.dart';
import 'package:material_ui/material_ui.dart';

/// Carries the name-leading storefront into the expanded profile identity.
class ProfileBusinessHero extends StatelessWidget {
  const new({
    required this.tag,
    required this.child,
    this.heroKey,
    super.key,
  });

  final Object tag;
  final Widget child;
  final GlobalKey? heroKey;

  static Widget source(BuildContext context, {required Widget child}) {
    final tag = ProfileAvatarHeroSource.maybeTagOf(context);
    if (tag == null) return child;
    return ProfileBusinessHero(tag: (tag, 'business'), child: child);
  }

  static Widget destination(BuildContext context, {required Widget child}) {
    final transition = ProfileAvatarHeroDestination.maybeOf(context);
    if (transition == null) return child;
    return ProfileBusinessHero(
      tag: (transition.tag, 'business'),
      heroKey: transition.businessDestinationKey,
      child: child,
    );
  }

  @override
  Widget build(BuildContext context) => HeroMode(
    enabled: !MediaQuery.disableAnimationsOf(context),
    child: Hero(
      key: heroKey,
      tag: tag,
      transitionOnUserGestures: true,
      flightShuttleBuilder: (context, animation, direction, from, to) {
        final expandedContext = direction == HeroFlightDirection.push
            ? to
            : from;
        final hero = expandedContext.widget as Hero;
        return Theme(
          data: Theme.of(expandedContext),
          child: Material(
            key: const Key('profile-business-flight'),
            type: MaterialType.transparency,
            child: FittedBox(child: hero.child),
          ),
        );
      },
      child: child,
    ),
  );
}
