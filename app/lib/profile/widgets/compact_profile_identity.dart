import 'dart:math' as math;

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/data/crafts_catalog.dart';
import 'package:craftsky_app/profile/models/profile_handle.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar_hero.dart';
import 'package:craftsky_app/profile/widgets/profile_craft_hero.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:material_ui/material_ui.dart';

/// The text beside a compact profile picture: name and crafts, then handle.
/// Callers own the picture and tap target so account and post actions retain
/// their existing behavior.
class CompactProfileIdentity extends StatelessWidget {
  const new({
    required this.handle,
    this.displayName,
    this.crafts = const [],
    this.badge,
    super.key,
  });

  final String handle;
  final String? displayName;
  final List<String> crafts;
  final Widget? badge;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    final identity = ProfileHandle(handle);
    final name = identity.displayLabel(
      displayName: displayName,
      unavailableLabel: l10n.handleUnavailable,
    );
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        LayoutBuilder(
          builder: (context, rowConstraints) => Row(
            children: [
              Expanded(
                child: LayoutBuilder(
                  builder: (context, constraints) {
                    final nameWidth = _textWidth(
                      context,
                      name,
                      theme.textTheme.titleSmall,
                    );
                    // The name takes priority over craft glyphs. Reserve only
                    // the overflow count when the full name and crafts cannot
                    // share the row.
                    final iconBudget = math.min(
                      constraints.maxWidth,
                      math.max(
                        constraints.maxWidth - nameWidth,
                        8 + _craftCountWidth(context, crafts.length),
                      ),
                    );
                    return Row(
                      children: [
                        Flexible(
                          child: ProfileIdentityTextHero.source(
                            context,
                            part: ProfileIdentityHeroPart.name,
                            child: Text(
                              name,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: theme.textTheme.titleSmall,
                            ),
                          ),
                        ),
                        if (crafts.isNotEmpty)
                          ConstrainedBox(
                            constraints: BoxConstraints(maxWidth: iconBudget),
                            child: _CompactCraftIcons(crafts: crafts),
                          ),
                      ],
                    );
                  },
                ),
              ),
              if (badge case final badge?) ...[
                const SizedBox(width: 8),
                ConstrainedBox(
                  constraints: BoxConstraints(
                    maxWidth: rowConstraints.maxWidth * .4,
                  ),
                  child: badge,
                ),
              ],
            ],
          ),
        ),
        ProfileIdentityTextHero.source(
          context,
          part: ProfileIdentityHeroPart.handle,
          child: Text(
            identity.currentLabel(unavailableLabel: l10n.handleUnavailable),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.outline,
            ),
          ),
        ),
      ],
    );
  }
}

class _CompactCraftIcons extends StatelessWidget {
  const new({required this.crafts});

  final List<String> crafts;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final entries = <String, String>{};
    for (final craft in crafts) {
      final id = craft.split('#').last.trim().toLowerCase();
      if (id.isEmpty) continue;
      final value = Craft.fromId(id);
      entries[id] = value == null ? id : craftLabel(value, l10n);
    }
    if (entries.isEmpty) return const SizedBox.shrink();
    final ids = entries.keys.toList(growable: false);
    final labels = entries.values.join(', ');
    const iconSize = 16.0;
    const gap = 4.0;
    const leadingGap = 8.0;
    return LayoutBuilder(
      builder: (context, constraints) {
        // Measure the count with the user's text scaling, then reserve room
        // for it before choosing icons. Unknown crafts remain in the count
        // and accessible label even when they have no branded glyph.
        final icons = ids
            .where((id) => CraftIcon.assetPathFor(id) != null)
            .toList(growable: false);
        var visible = icons.length;

        while (visible > 0) {
          final hidden = ids.length - visible;
          final width =
              leadingGap +
              visible * iconSize +
              (visible - 1) * gap +
              (hidden > 0 ? gap + _craftCountWidth(context, hidden) : 0);
          if (width <= constraints.maxWidth) break;
          visible--;
        }
        final hidden = ids.length - visible;
        return Tooltip(
          message: labels,
          excludeFromSemantics: true,
          child: Semantics(
            label: labels,
            excludeSemantics: true,
            child: Padding(
              padding: const EdgeInsetsDirectional.only(start: leadingGap),
              child: ClipRect(
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    for (var i = 0; i < visible; i++) ...[
                      if (i > 0) const SizedBox(width: gap),
                      ProfileCraftHero.source(
                        context,
                        craft: icons[i],
                        child: CraftIcon(craft: icons[i], size: iconSize),
                      ),
                    ],
                    if (hidden > 0) ...[
                      if (visible > 0) const SizedBox(width: gap),
                      Flexible(
                        child: Text(
                          '+$hidden',
                          maxLines: 1,
                          overflow: TextOverflow.clip,
                          style: theme.textTheme.bodySmall,
                        ),
                      ),
                    ],
                  ],
                ),
              ),
            ),
          ),
        );
      },
    );
  }
}

double _craftCountWidth(BuildContext context, int count) => _textWidth(
  context,
  '+$count',
  Theme.of(context).textTheme.bodySmall,
);

double _textWidth(BuildContext context, String text, TextStyle? style) {
  final painter = TextPainter(
    text: TextSpan(
      text: text,
      style: style,
    ),
    textDirection: Directionality.of(context),
    textScaler: MediaQuery.textScalerOf(context),
  )..layout();
  final width = painter.width;
  painter.dispose();
  return width;
}
