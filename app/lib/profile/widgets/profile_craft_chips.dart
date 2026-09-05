import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Pill chips listing the crafts a user works in. Renders nothing when
/// the list is empty so callers don't need to gate visibility.
class ProfileCraftChips extends StatelessWidget {
  const ProfileCraftChips({
    required this.crafts,
    this.alignment = WrapAlignment.start,
    super.key,
  });

  final List<String> crafts;
  final WrapAlignment alignment;

  @override
  Widget build(BuildContext context) {
    if (crafts.isEmpty) return const SizedBox.shrink();
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return Wrap(
      alignment: alignment,
      spacing: spacing.sp2,
      runSpacing: spacing.sp2,
      children: [for (final craft in crafts) _CraftChip(label: craft)],
    );
  }
}

/// A single craft pill using the active profile colour theme.
class _CraftChip extends StatelessWidget {
  const _CraftChip({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final display = _toSentenceCase(label);
    return Container(
      padding: EdgeInsets.symmetric(horizontal: spacing.sp3, vertical: 6),
      decoration: BoxDecoration(
        color: theme.colorScheme.primaryContainer,
        borderRadius: BorderRadius.circular(radii.rPill),
      ),
      child: CraftIconLabel(
        craft: label,
        label: display,
        iconSize: 16,
        gap: spacing.sp1,
        flexibleLabel: true,
        style: theme.textTheme.labelMedium?.copyWith(
          color: theme.colorScheme.onPrimaryContainer,
        ),
      ),
    );
  }

  String _toSentenceCase(String value) {
    if (value.isEmpty) return value;
    return value[0].toUpperCase() + value.substring(1);
  }
}
