import 'dart:math' as math;

import 'package:craftsky_app/profile/widgets/profile_avatar_hero.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:material_ui/material_ui.dart';

/// One visible craft icon grows into its matching profile chip. Overflowed
/// crafts have no source hero and appear normally with the profile page.
class ProfileCraftHero extends StatelessWidget {
  const new({
    required this.tag,
    required this.craft,
    required this.child,
    this.label,
    this.heroKey,
    super.key,
  });

  final Object tag;
  final String craft;
  final Widget child;
  final String? label;
  final GlobalKey? heroKey;

  static String idFor(String craft) =>
      craft.split('#').last.trim().toLowerCase();

  static Widget source(
    BuildContext context, {
    required String craft,
    required Widget child,
  }) {
    final sourceTag = ProfileAvatarHeroSource.maybeTagOf(context);
    if (sourceTag == null) return child;
    final id = idFor(craft);
    return ProfileCraftHero(
      tag: (sourceTag, 'craft', id),
      craft: id,
      child: child,
    );
  }

  static Widget destination(
    BuildContext context, {
    required String craft,
    required String label,
    required Widget child,
  }) {
    final transition = ProfileAvatarHeroDestination.maybeOf(context);
    final id = idFor(craft);
    if (transition == null || CraftIcon.assetPathFor(id) == null) return child;
    return ProfileCraftHero(
      tag: (transition.tag, 'craft', id),
      craft: id,
      label: label,
      heroKey: transition.craftDestinationKey(id),
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
      child: _CraftHeroContent(craft: craft, label: label, child: child),
      flightShuttleBuilder: (context, animation, direction, from, to) {
        final chipContext = direction == HeroFlightDirection.push ? to : from;
        final iconContext = direction == HeroFlightDirection.push ? from : to;
        final chip = (chipContext.widget as Hero).child as _CraftHeroContent;
        final theme = Theme.of(chipContext);
        final spacing = theme.extension<SpacingTheme>()!;
        final radius = theme.extension<RadiusTheme>()!.rPill;
        final iconColor =
            IconTheme.of(iconContext).color ?? theme.colorScheme.onSurface;
        final chipColor = theme.colorScheme.onPrimaryContainer;
        final sourceSurface = Theme.of(iconContext).scaffoldBackgroundColor;
        final textScaler = MediaQuery.textScalerOf(chipContext);
        return AnimatedBuilder(
          animation: animation,
          builder: (context, _) {
            // Hero supplies the route animation: 0 -> 1 on entry, 1 -> 0
            // on Back. The chip follows that same expansion in both directions.
            final progress = animation.value;
            // On return, shed the chip appearance first, then carry the plain
            // glyph back to the name. Geometry still follows the entire flight.
            final appearance = direction == HeroFlightDirection.push
                ? progress
                : const Interval(.8, 1).transform(progress);
            final background = theme.colorScheme.primaryContainer.withValues(
              alpha: theme.colorScheme.primaryContainer.a * appearance,
            );
            final foreground = direction == HeroFlightDirection.push
                ? Color.lerp(iconColor, chipColor, progress)!
                : _returnIconColor(
                    iconColor: iconColor,
                    chipColor: chipColor,
                    background: background,
                    surface: sourceSurface,
                  );
            return Material(
              type: MaterialType.transparency,
              child: ClipRRect(
                key: ValueKey('profile-craft-flight-${chip.craft}'),
                borderRadius: BorderRadius.circular(radius * progress),
                child: ColoredBox(
                  color: background,
                  child: Padding(
                    padding: EdgeInsets.symmetric(
                      horizontal: spacing.sp3 * progress,
                      vertical: 6 * progress,
                    ),
                    child: LayoutBuilder(
                      builder: (context, constraints) => OverflowBox(
                        // The flight may briefly be smaller than its glyph.
                        // Keep the glyph full-size and explicitly allow that
                        // overflow; the hero's clip still controls its reveal.
                        alignment: AlignmentDirectional.centerStart,
                        minWidth: 16,
                        maxWidth: math.max(16, constraints.maxWidth),
                        minHeight: 16,
                        maxHeight: math.max(16, constraints.maxHeight),
                        child: Row(
                          children: [
                            CraftIcon(
                              craft: chip.craft,
                              size: 16,
                              color: foreground,
                            ),
                            Expanded(
                              child: ClipRect(
                                child: Padding(
                                  padding: EdgeInsetsDirectional.only(
                                    start: spacing.sp1 * progress,
                                  ),
                                  child: Opacity(
                                    opacity: appearance,
                                    child: Text(
                                      chip.label!,
                                      textScaler: textScaler,
                                      maxLines: 1,
                                      softWrap: false,
                                      overflow: TextOverflow.clip,
                                      style: theme.textTheme.labelMedium
                                          ?.copyWith(
                                            color: foreground,
                                          ),
                                    ),
                                  ),
                                ),
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            );
          },
        );
      },
    ),
  );
}

class _CraftHeroContent extends StatelessWidget {
  const new({required this.craft, required this.label, required this.child});

  final String craft;
  final String? label;
  final Widget child;

  @override
  Widget build(BuildContext context) => child;
}

// Interpolating dark chip ink into a light dark-mode glyph can match the
// fading background midway through the flight. Keep the higher-contrast ink
// until the chip is gone, then use exactly the source icon's colour.
Color _returnIconColor({
  required Color iconColor,
  required Color chipColor,
  required Color background,
  required Color surface,
}) {
  if (background.a == 0) return iconColor;
  final backdrop = Color.alphaBlend(background, surface).computeLuminance();
  double contrast(Color color) {
    final luminance = color.computeLuminance();
    return (math.max(luminance, backdrop) + .05) /
        (math.min(luminance, backdrop) + .05);
  }

  return contrast(iconColor) >= contrast(chipColor) ? iconColor : chipColor;
}
