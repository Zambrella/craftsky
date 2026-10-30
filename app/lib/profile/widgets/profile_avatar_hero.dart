import 'dart:async';

import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:material_ui/material_ui.dart';

/// In-memory navigation context for one particular identity on the source page.
/// A per-source tag lets a feed contain several avatars for the same person.
class ProfileAvatarTransition {
  new({
    required this.tag,
    required this.avatar,
    required this.handle,
    this.displayName,
    this.crafts = const [],
  });

  final Object tag;
  final ProfileAvatar avatar;
  final String handle;
  final String? displayName;
  final List<String> crafts;

  // Preserve the placeholder when a response replaces the loading header
  // during the flight. This key belongs only to the destination route.
  final GlobalKey destinationKey = GlobalKey(debugLabel: 'profile-avatar-hero');
  final GlobalKey nameDestinationKey = GlobalKey(
    debugLabel: 'profile-name-hero',
  );
  final GlobalKey handleDestinationKey = GlobalKey(
    debugLabel: 'profile-handle-hero',
  );
  final Map<String, GlobalKey> _craftDestinationKeys = {};

  GlobalKey craftDestinationKey(String craft) =>
      _craftDestinationKeys.putIfAbsent(
        craft,
        () => GlobalKey(debugLabel: 'profile-craft-$craft-hero'),
      );
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
    required this.handle,
    required this.builder,
    this.displayName,
    this.crafts = const [],
    this.beforeOpen,
    super.key,
  });

  final Future<void> Function()? beforeOpen;
  final Did did;
  final String handle;
  final String? displayName;
  final List<String> crafts;
  final ProfileAvatar avatar;
  final Widget Function(BuildContext, Widget avatar, VoidCallback openProfile)
  builder;

  static Object? maybeTagOf(BuildContext context) => context
      .dependOnInheritedWidgetOfExactType<_ProfileIdentityHeroSourceScope>()
      ?.tag;

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
  Widget build(BuildContext context) => _ProfileIdentityHeroSourceScope(
    tag: _tag,
    hasDisplayName: widget.displayName?.trim().isNotEmpty ?? false,
    child: Builder(
      builder: (context) => widget.builder(
        context,
        ProfileAvatarHero(tag: _tag, child: widget.avatar),
        () => unawaited(_openProfile(context)),
      ),
    ),
  );

  Future<void> _openProfile(BuildContext context) async {
    final did = widget.did;
    final avatar = widget.avatar;
    final tag = _tag;
    final handle = widget.handle;
    final displayName = widget.displayName;
    final crafts = List<String>.unmodifiable(widget.crafts);
    await widget.beforeOpen?.call();
    if (!context.mounted) return;
    await UserProfileRoute(
      did: did,
      $extra: MediaQuery.disableAnimationsOf(context)
          ? null
          : ProfileAvatarTransition(
              tag: tag,
              avatar: avatar,
              handle: handle,
              displayName: displayName,
              crafts: crafts,
            ),
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

enum ProfileIdentityHeroPart { name, handle }

class _ProfileIdentityHeroSourceScope extends InheritedWidget {
  const new({
    required this.tag,
    required this.hasDisplayName,
    required super.child,
  });

  final Object tag;
  final bool hasDisplayName;

  @override
  bool updateShouldNotify(_ProfileIdentityHeroSourceScope oldWidget) =>
      tag != oldWidget.tag || hasDisplayName != oldWidget.hasDisplayName;
}

/// Animates the identity text independently of craft icons and other metadata.
/// Typography and colour follow the route animation.
class ProfileIdentityTextHero extends StatelessWidget {
  const new({
    required this.tag,
    required this.part,
    required this.child,
    this.heroKey,
    this.preservePlaceholderBaseline = false,
    super.key,
  });

  final Object tag;
  final ProfileIdentityHeroPart part;
  final Text child;
  final GlobalKey? heroKey;
  final bool preservePlaceholderBaseline;

  static Widget source(
    BuildContext context, {
    required ProfileIdentityHeroPart part,
    required Text child,
    bool preservePlaceholderBaseline = false,
  }) {
    final scope = context
        .dependOnInheritedWidgetOfExactType<_ProfileIdentityHeroSourceScope>();
    if (scope == null ||
        (part == ProfileIdentityHeroPart.handle && !scope.hasDisplayName)) {
      return child;
    }
    return ProfileIdentityTextHero(
      tag: (scope.tag, part),
      part: part,
      preservePlaceholderBaseline: preservePlaceholderBaseline,
      child: child,
    );
  }

  static Widget destination(
    BuildContext context, {
    required ProfileIdentityHeroPart part,
    required Text child,
  }) {
    final transition = ProfileAvatarHeroDestination.maybeOf(context);
    if (transition == null) return child;
    return ProfileIdentityTextHero(
      tag: (transition.tag, part),
      part: part,
      heroKey: part == ProfileIdentityHeroPart.name
          ? transition.nameDestinationKey
          : transition.handleDestinationKey,
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
        final fromText = (from.widget as Hero).child as Text;
        final toText = (to.widget as Hero).child as Text;
        final fromStyle = DefaultTextStyle.of(from).style.merge(fromText.style);
        final toStyle = DefaultTextStyle.of(to).style.merge(toText.style);
        final text = direction == HeroFlightDirection.push ? toText : fromText;
        final textScaler = MediaQuery.textScalerOf(context);
        return AnimatedBuilder(
          animation: animation,
          builder: (context, _) => Material(
            key: Key('profile-${part.name}-flight'),
            type: MaterialType.transparency,
            child: FittedBox(
              child: Text(
                text.data!,
                style: TextStyle.lerp(
                  fromStyle,
                  toStyle,
                  direction == HeroFlightDirection.push
                      ? animation.value
                      : 1 - animation.value,
                ),
                textScaler: textScaler,
                maxLines: 1,
                softWrap: false,
              ),
            ),
          ),
        );
      },
      // An empty box preserves size but loses the baseline of inline text,
      // making its surrounding paragraph reflow until the hero lands.
      placeholderBuilder: preservePlaceholderBaseline
          ? (context, size, child) => SizedBox.fromSize(
              size: size,
              child: ExcludeSemantics(
                child: Opacity(opacity: 0, child: child),
              ),
            )
          : null,
      child: child,
    ),
  );
}
