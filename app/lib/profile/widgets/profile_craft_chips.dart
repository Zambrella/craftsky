import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Pill chips listing the crafts a user works in. Renders nothing when
/// the list is empty so callers don't need to gate visibility.
class ProfileCraftChips extends StatelessWidget {
  const ProfileCraftChips({required this.crafts, super.key});

  final List<String> crafts;

  @override
  Widget build(BuildContext context) {
    if (crafts.isEmpty) return const SizedBox.shrink();
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return Wrap(
      spacing: spacing.sp2,
      runSpacing: spacing.sp2,
      children: [for (final craft in crafts) _CraftChip(label: craft)],
    );
  }
}

/// A single craft pill: ink-bordered paper with the craft glyph (TODO)
/// and a sentence-cased label. Iconography is pending the custom craft
/// icon set, so the leading slot is left blank for now.
class _CraftChip extends StatelessWidget {
  const _CraftChip({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final display = _toSentenceCase(label);
    return Container(
      padding: EdgeInsets.symmetric(horizontal: spacing.sp3, vertical: 6),
      decoration: BoxDecoration(
        color: swatches.paper3,
        borderRadius: BorderRadius.circular(radii.rPill),
        border: Border.all(color: theme.colorScheme.onSurface, width: 1.5),
      ),
      child: Text(
        display,
        style: theme.textTheme.labelMedium?.copyWith(
          color: theme.colorScheme.onSurface,
        ),
      ),
    );
  }

  String _toSentenceCase(String value) {
    if (value.isEmpty) return value;
    return value[0].toUpperCase() + value.substring(1);
  }
}
